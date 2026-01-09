// Package orchestrator provides deployment orchestration for OtterStack.
package orchestrator

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/state"
)

// DeployOptions contains options for a deployment.
type DeployOptions struct {
	GitRef    string
	Timeout   time.Duration
	SkipPull  bool
	DataDir   string
	OnStatus  func(msg string) // Callback for status messages
	OnVerbose func(msg string) // Callback for verbose messages
}

// DeployResult contains the result of a deployment.
type DeployResult struct {
	Deployment *state.Deployment
	ShortSHA   string
}

// Deployer orchestrates deployments.
type Deployer struct {
	store  state.StateStore
	gitMgr git.GitOperations
}

// NewDeployer creates a new Deployer for the given project.
func NewDeployer(store state.StateStore, gitMgr git.GitOperations) *Deployer {
	return &Deployer{
		store:  store,
		gitMgr: gitMgr,
	}
}

// Deploy performs a deployment for the given project.
func (d *Deployer) Deploy(ctx context.Context, project *state.Project, opts DeployOptions) (*DeployResult, error) {
	// Default callbacks
	onStatus := opts.OnStatus
	if onStatus == nil {
		onStatus = func(msg string) { fmt.Println(msg) }
	}
	onVerbose := opts.OnVerbose
	if onVerbose == nil {
		onVerbose = func(msg string) {}
	}

	// Fetch latest changes for remote repos
	if project.RepoType == "remote" {
		onStatus("Fetching latest changes...")
		if err := d.gitMgr.Fetch(ctx); err != nil {
			return nil, fmt.Errorf("failed to fetch: %w", err)
		}
	}

	// Resolve git reference
	gitRef := opts.GitRef
	if gitRef == "" {
		defaultBranch, err := d.gitMgr.GetDefaultBranch(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default branch: %w", err)
		}
		gitRef = defaultBranch
		onStatus(fmt.Sprintf("Using default branch: %s", gitRef))
	}

	fullSHA, err := d.gitMgr.ResolveRef(ctx, gitRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ref %q: %w", gitRef, err)
	}

	shortSHA := git.ShortSHA(fullSHA)
	onStatus(fmt.Sprintf("Deploying %s (%s -> %s)", project.Name, gitRef, shortSHA))

	// Create deployment record
	worktreePath := git.GetWorktreePath(opts.DataDir, project.Name, fullSHA)
	deployment := &state.Deployment{
		ProjectID:    project.ID,
		GitSHA:       fullSHA,
		GitRef:       gitRef,
		WorktreePath: worktreePath,
		Status:       "deploying",
	}

	if err := d.store.CreateDeployment(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Set up cleanup on failure
	success := false
	defer func() {
		if !success {
			errMsg := "deployment interrupted"
			d.store.UpdateDeploymentStatus(ctx, deployment.ID, "failed", &errMsg)
		}
	}()

	// Create worktree
	onVerbose(fmt.Sprintf("Creating worktree at %s...", worktreePath))
	if _, err := os.Stat(worktreePath); err == nil {
		onVerbose("Worktree already exists, reusing...")
	} else {
		if err := d.gitMgr.CreateWorktree(ctx, worktreePath, fullSHA); err != nil {
			return nil, fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Initialize compose manager
	composeProjectName := compose.GenerateProjectName(project.Name, shortSHA)
	composeMgr := compose.NewManager(worktreePath, project.ComposeFile, composeProjectName)

	// Validate compose file
	onVerbose("Validating compose file...")
	if err := composeMgr.Validate(ctx); err != nil {
		return nil, fmt.Errorf("compose validation failed: %w", err)
	}

	// Pull images if not skipped
	if !opts.SkipPull {
		onStatus("Pulling images...")
		if err := composeMgr.Pull(ctx); err != nil {
			onVerbose(fmt.Sprintf("Warning: pull failed (continuing): %v", err))
		}
	}

	// Check context before starting services
	if ctx.Err() != nil {
		return nil, fmt.Errorf("deployment cancelled: %w", ctx.Err())
	}

	// Start services with timeout
	onStatus("Starting services...")
	deployCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if err := composeMgr.Up(deployCtx); err != nil {
		return nil, fmt.Errorf("failed to start services: %w", err)
	}

	// Deactivate previous deployments
	if err := d.store.DeactivatePreviousDeployments(ctx, project.ID, deployment.ID); err != nil {
		onVerbose(fmt.Sprintf("Warning: failed to deactivate previous deployments: %v", err))
	}

	// Stop previous deployment's containers
	previousDeployment, err := d.store.GetPreviousDeployment(ctx, project.ID)
	if err == nil && previousDeployment != nil {
		oldProjectName := compose.GenerateProjectName(project.Name, git.ShortSHA(previousDeployment.GitSHA))
		onVerbose(fmt.Sprintf("Stopping previous deployment %s...", git.ShortSHA(previousDeployment.GitSHA)))
		if err := compose.StopProjectByName(ctx, oldProjectName, 30*time.Second); err != nil {
			onVerbose(fmt.Sprintf("Warning: failed to stop previous deployment: %v", err))
		}
	}

	// Mark deployment as active
	if err := d.store.UpdateDeploymentStatus(ctx, deployment.ID, "active", nil); err != nil {
		return nil, fmt.Errorf("failed to update deployment status: %w", err)
	}

	success = true

	return &DeployResult{
		Deployment: deployment,
		ShortSHA:   shortSHA,
	}, nil
}

// CleanupOldWorktrees removes worktrees beyond the retention limit.
func (d *Deployer) CleanupOldWorktrees(ctx context.Context, project *state.Project, dataDir string, onVerbose func(string)) error {
	if onVerbose == nil {
		onVerbose = func(msg string) {}
	}

	deployments, err := d.store.ListDeployments(ctx, project.ID, 100)
	if err != nil {
		return err
	}

	if len(deployments) <= project.WorktreeRetention {
		return nil
	}

	for i := project.WorktreeRetention; i < len(deployments); i++ {
		dep := deployments[i]
		if dep.WorktreePath == "" {
			continue
		}
		if dep.Status == "active" || dep.Status == "deploying" {
			continue
		}

		onVerbose(fmt.Sprintf("Removing old worktree: %s", dep.WorktreePath))
		if err := d.gitMgr.RemoveWorktree(ctx, dep.WorktreePath); err != nil {
			onVerbose(fmt.Sprintf("Warning: failed to remove worktree %s: %v", dep.WorktreePath, err))
		}
	}

	return nil
}
