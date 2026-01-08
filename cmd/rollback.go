package cmd

import (
	"fmt"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
	"github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <project>",
	Short: "Rollback to the previous deployment",
	Long: `Rollback a project to its previous successful deployment.

This stops the current deployment and starts the previous one.`,
	Args: cobra.ExactArgs(1),
	RunE: runRollback,
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	projectName := args[0]

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
		if err == errors.ErrProjectNotFound {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	// Get current active deployment
	currentDeployment, err := store.GetActiveDeployment(ctx, project.ID)
	if err != nil {
		if err == errors.ErrNoActiveDeployment {
			return fmt.Errorf("no active deployment to rollback from")
		}
		return err
	}

	// Get previous deployment
	previousDeployment, err := store.GetPreviousDeployment(ctx, project.ID)
	if err != nil {
		if err == errors.ErrNoPreviousDeployment {
			return fmt.Errorf("no previous deployment to rollback to")
		}
		return err
	}

	fmt.Printf("Rolling back %s from %s to %s\n",
		projectName,
		git.ShortSHA(currentDeployment.GitSHA),
		git.ShortSHA(previousDeployment.GitSHA))

	// Check that previous worktree still exists
	gitMgr := git.NewManager(project.RepoPath)
	if !gitMgr.CommitExists(ctx, previousDeployment.GitSHA) {
		return fmt.Errorf("previous deployment commit %s no longer exists in repository", git.ShortSHA(previousDeployment.GitSHA))
	}

	// Start the previous deployment
	previousProjectName := compose.GenerateProjectName(projectName, git.ShortSHA(previousDeployment.GitSHA))
	composeMgr := compose.NewManager(previousDeployment.WorktreePath, project.ComposeFile, previousProjectName)

	// Validate compose file
	if err := composeMgr.Validate(ctx); err != nil {
		return fmt.Errorf("compose validation failed: %w", err)
	}

	fmt.Println("Starting previous deployment...")
	if err := composeMgr.Up(ctx); err != nil {
		return fmt.Errorf("failed to start previous deployment: %w", err)
	}

	// Stop current deployment
	currentProjectName := compose.GenerateProjectName(projectName, git.ShortSHA(currentDeployment.GitSHA))
	fmt.Println("Stopping current deployment...")
	if err := compose.StopProjectByName(ctx, currentProjectName, 30*time.Second); err != nil {
		printVerbose("Warning: failed to stop current deployment: %v", err)
	}

	// Update database state
	// Mark current as rolled_back
	if err := store.UpdateDeploymentStatus(ctx, currentDeployment.ID, "rolled_back", nil); err != nil {
		printVerbose("Warning: failed to update current deployment status: %v", err)
	}

	// Create new deployment record for the rollback
	rollbackDeployment := &state.Deployment{
		ProjectID:    project.ID,
		GitSHA:       previousDeployment.GitSHA,
		GitRef:       previousDeployment.GitRef,
		WorktreePath: previousDeployment.WorktreePath,
		Status:       "active",
	}
	if err := store.CreateDeployment(ctx, rollbackDeployment); err != nil {
		return fmt.Errorf("failed to create rollback deployment record: %w", err)
	}

	fmt.Printf("Rollback successful! %s now running at %s\n", projectName, git.ShortSHA(previousDeployment.GitSHA))

	return nil
}
