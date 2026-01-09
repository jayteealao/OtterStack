// Package orchestrator provides deployment orchestration for OtterStack.
package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

	// Write env file if project has env vars
	envVars, err := d.store.GetEnvVars(ctx, project.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get env vars: %w", err)
	}

	envFilePath, err := writeEnvFile(opts.DataDir, project.Name, envVars)
	if err != nil {
		return nil, fmt.Errorf("failed to write env file: %w", err)
	}
	if envFilePath != "" {
		onVerbose(fmt.Sprintf("Using env file: %s", envFilePath))
	}

	// Start services with timeout
	onStatus("Starting services...")
	deployCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if err := composeMgr.Up(deployCtx, envFilePath); err != nil {
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

// writeEnvFile writes environment variables to a file in dotenv format.
// Returns the file path if env vars exist, empty string otherwise.
func writeEnvFile(dataDir, projectName string, vars map[string]string) (string, error) {
	if len(vars) == 0 {
		return "", nil
	}

	// Create envfiles directory if needed
	envDir := filepath.Join(dataDir, "envfiles")
	if err := os.MkdirAll(envDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create envfiles directory: %w", err)
	}

	// Write env file with 0600 permissions
	envPath := filepath.Join(envDir, projectName+".env")
	f, err := os.OpenFile(envPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create env file: %w", err)
	}
	defer f.Close()

	// Sort keys for deterministic output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// Write in KEY=value format, quoting values that contain special characters
		v := vars[k]
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, v); err != nil {
			return "", fmt.Errorf("failed to write env var: %w", err)
		}
	}

	return envPath, nil
}
