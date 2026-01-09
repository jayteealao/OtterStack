package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
	apperrors "github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/jayteealao/otterstack/internal/validate"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy <project> [ref]",
	Short: "Deploy a project",
	Long: `Deploy a project to a specific git reference (tag, branch, or commit).

If no reference is specified, the default branch (main/master) is used.

Examples:
  otterstack deploy myapp v1.0.0
  otterstack deploy myapp main
  otterstack deploy myapp abc123d`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runDeploy,
}

var (
	deployTimeoutFlag time.Duration
	skipPullFlag      bool
)

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().DurationVar(&deployTimeoutFlag, "timeout", 5*time.Minute, "deployment timeout")
	deployCmd.Flags().BoolVar(&skipPullFlag, "skip-pull", false, "skip pulling images before deployment")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	projectName := args[0]

	var gitRef string
	if len(args) > 1 {
		gitRef = args[1]
		if err := validate.GitRef(gitRef); err != nil {
			return fmt.Errorf("invalid git ref: %w", err)
		}
	}

	// Initialize store
	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Initialize lock manager
	lockMgr, err := initLockManager()
	if err != nil {
		return err
	}

	// Acquire project lock
	printVerbose("Acquiring lock for project %s...", projectName)
	lock, err := lockMgr.Acquire(ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lock.Release()

	// Get project
	project, err := store.GetProject(ctx, projectName)
	if err != nil {
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	if project.Status != "ready" {
		return fmt.Errorf("project is not ready (status: %s)", project.Status)
	}

	// Initialize git manager
	gitMgr := git.NewManager(project.RepoPath)

	// Fetch latest changes for remote repos
	if project.RepoType == "remote" {
		fmt.Println("Fetching latest changes...")
		if err := gitMgr.Fetch(ctx); err != nil {
			return fmt.Errorf("failed to fetch: %w", err)
		}
	}

	// Resolve git reference
	if gitRef == "" {
		// Use default branch
		defaultBranch, err := gitMgr.GetDefaultBranch(ctx)
		if err != nil {
			return fmt.Errorf("failed to get default branch: %w", err)
		}
		gitRef = defaultBranch
		fmt.Printf("Using default branch: %s\n", gitRef)
	}

	fullSHA, err := gitMgr.ResolveRef(ctx, gitRef)
	if err != nil {
		return fmt.Errorf("failed to resolve ref %q: %w", gitRef, err)
	}

	shortSHA := git.ShortSHA(fullSHA)
	fmt.Printf("Deploying %s (%s -> %s)\n", projectName, gitRef, shortSHA)

	// Get data directory for worktree path
	dataDir, err := getDataDir()
	if err != nil {
		return err
	}

	// Create deployment record
	worktreePath := git.GetWorktreePath(dataDir, projectName, fullSHA)
	deployment := &state.Deployment{
		ProjectID:    project.ID,
		GitSHA:       fullSHA,
		GitRef:       gitRef,
		WorktreePath: worktreePath,
		Status:       "deploying",
	}

	if err := store.CreateDeployment(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Set up cleanup on failure
	success := false
	defer func() {
		if !success {
			// Mark deployment as failed
			errMsg := "deployment interrupted"
			store.UpdateDeploymentStatus(ctx, deployment.ID, "failed", &errMsg)
		}
	}()

	// Create worktree
	printVerbose("Creating worktree at %s...", worktreePath)
	if _, err := os.Stat(worktreePath); err == nil {
		// Worktree already exists, reuse it
		printVerbose("Worktree already exists, reusing...")
	} else {
		if err := gitMgr.CreateWorktree(ctx, worktreePath, fullSHA); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Initialize compose manager
	composeProjectName := compose.GenerateProjectName(projectName, shortSHA)
	composeMgr := compose.NewManager(worktreePath, project.ComposeFile, composeProjectName)

	// Validate compose file
	printVerbose("Validating compose file...")
	if err := composeMgr.Validate(ctx); err != nil {
		return fmt.Errorf("compose validation failed: %w", err)
	}

	// Pull images if not skipped
	if !skipPullFlag {
		fmt.Println("Pulling images...")
		if err := composeMgr.Pull(ctx); err != nil {
			// Non-fatal, images might be local
			printVerbose("Warning: pull failed (continuing): %v", err)
		}
	}

	// Check context before starting services
	if err := checkContext(ctx); err != nil {
		return fmt.Errorf("deployment cancelled: %w", err)
	}

	// Start services with timeout
	fmt.Println("Starting services...")
	deployCtx, cancel := context.WithTimeout(ctx, deployTimeoutFlag)
	defer cancel()

	if err := composeMgr.Up(deployCtx); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	// Deactivate previous deployments
	if err := store.DeactivatePreviousDeployments(ctx, project.ID, deployment.ID); err != nil {
		printVerbose("Warning: failed to deactivate previous deployments: %v", err)
	}

	// Stop previous deployment's containers
	previousDeployment, err := store.GetPreviousDeployment(ctx, project.ID)
	if err == nil && previousDeployment != nil {
		oldProjectName := compose.GenerateProjectName(projectName, git.ShortSHA(previousDeployment.GitSHA))
		printVerbose("Stopping previous deployment %s...", git.ShortSHA(previousDeployment.GitSHA))
		if err := compose.StopProjectByName(ctx, oldProjectName, 30*time.Second); err != nil {
			printVerbose("Warning: failed to stop previous deployment: %v", err)
		}
	}

	// Mark deployment as active
	if err := store.UpdateDeploymentStatus(ctx, deployment.ID, "active", nil); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	success = true
	fmt.Printf("Deployment successful! %s deployed at %s\n", projectName, shortSHA)

	// Clean up old worktrees if retention limit exceeded
	if project.WorktreeRetention > 0 {
		if err := cleanupOldWorktrees(ctx, store, gitMgr, project, dataDir); err != nil {
			printVerbose("Warning: failed to cleanup old worktrees: %v", err)
		}
	}

	return nil
}

func cleanupOldWorktrees(ctx context.Context, store *state.Store, gitMgr *git.Manager, project *state.Project, dataDir string) error {
	// Get all deployments for this project
	deployments, err := store.ListDeployments(ctx, project.ID, 100)
	if err != nil {
		return err
	}

	// Count how many worktrees we have
	if len(deployments) <= project.WorktreeRetention {
		return nil // Nothing to clean up
	}

	// Remove excess worktrees (oldest first)
	for i := project.WorktreeRetention; i < len(deployments); i++ {
		d := deployments[i]
		if d.WorktreePath == "" {
			continue
		}

		// Skip if status is active or deploying
		if d.Status == "active" || d.Status == "deploying" {
			continue
		}

		printVerbose("Removing old worktree: %s", d.WorktreePath)
		if err := gitMgr.RemoveWorktree(ctx, d.WorktreePath); err != nil {
			printVerbose("Warning: failed to remove worktree %s: %v", d.WorktreePath, err)
		}
	}

	return nil
}
