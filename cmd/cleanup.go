package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
	apperrors "github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up orphaned resources",
	Long: `Clean up orphaned worktrees, containers, and reconcile state.

This command:
1. Marks interrupted deployments as failed
2. Removes orphaned worktrees not referenced by any deployment
3. Stops containers from failed deployments
4. Prunes git worktree references`,
	RunE: runCleanup,
}

var (
	cleanupDryRunFlag bool
)

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().BoolVar(&cleanupDryRunFlag, "dry-run", false, "show what would be cleaned without making changes")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	dataDir, err := getDataDir()
	if err != nil {
		return err
	}

	fmt.Println("Starting cleanup...")
	if cleanupDryRunFlag {
		fmt.Println("(dry run mode - no changes will be made)")
	}
	fmt.Println()

	// 1. Mark interrupted deployments as failed
	fmt.Println("Checking for interrupted deployments...")
	interrupted, err := store.GetInterruptedDeployments(ctx)
	if err != nil {
		return fmt.Errorf("failed to get interrupted deployments: %w", err)
	}

	for _, d := range interrupted {
		fmt.Printf("  Found interrupted deployment: %s (status: %s)\n", git.ShortSHA(d.GitSHA), d.Status)
		if !cleanupDryRunFlag {
			errMsg := "marked as interrupted during cleanup"
			if err := store.UpdateDeploymentStatus(ctx, d.ID, "interrupted", &errMsg); err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: failed to update status: %v\n", err)
			}
		}
	}
	fmt.Println()

	// 2. Get all projects and clean up orphaned worktrees
	fmt.Println("Checking for orphaned worktrees...")
	projects, err := store.ListProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	worktreesDir := filepath.Join(dataDir, "worktrees")
	if _, err := os.Stat(worktreesDir); err == nil {
		// List project directories
		entries, err := os.ReadDir(worktreesDir)
		if err != nil {
			return fmt.Errorf("failed to read worktrees directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			projectName := entry.Name()

			// Check if project exists in database
			_, err := store.GetProject(ctx, projectName)
			if errors.Is(err, apperrors.ErrProjectNotFound) {
				fmt.Printf("  Found orphaned project directory: %s\n", projectName)
				if !cleanupDryRunFlag {
					orphanDir := filepath.Join(worktreesDir, projectName)
					if err := os.RemoveAll(orphanDir); err != nil {
						fmt.Fprintf(os.Stderr, "    Warning: failed to remove: %v\n", err)
					}
				}
				continue
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to check project %s: %v\n", projectName, err)
				continue
			}

			// TODO: Per-project worktree cleanup can be added here as a future enhancement
		}
	}
	fmt.Println()

	// 3. Stop containers from failed deployments
	fmt.Println("Checking for orphaned containers...")
	for _, project := range projects {
		// Find running compose projects that match our naming pattern
		runningProjects, err := compose.FindRunningProjects(ctx, project.Name+"-")
		if err != nil {
			printVerbose("  Warning: failed to find running projects for %s: %v", project.Name, err)
			continue
		}

		// Get active deployment
		activeDeployment, err := store.GetActiveDeployment(ctx, project.ID)
		activeProjectName := ""
		if err == nil && activeDeployment != nil {
			activeProjectName = compose.GenerateProjectName(project.Name, git.ShortSHA(activeDeployment.GitSHA))
		}

		for _, runningProject := range runningProjects {
			if runningProject == activeProjectName {
				continue // Skip active deployment
			}

			fmt.Printf("  Found orphaned compose project: %s\n", runningProject)
			if !cleanupDryRunFlag {
				if err := compose.StopProjectByName(ctx, runningProject, 30*time.Second); err != nil {
					fmt.Fprintf(os.Stderr, "    Warning: failed to stop: %v\n", err)
				}
			}
		}
	}
	fmt.Println()

	// 4. Prune git worktree references
	fmt.Println("Pruning git worktree references...")
	for _, project := range projects {
		gitMgr := git.NewManager(project.RepoPath)
		if !gitMgr.IsGitRepo(ctx) {
			continue
		}

		if !cleanupDryRunFlag {
			if err := gitMgr.PruneWorktrees(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to prune worktrees for %s: %v\n", project.Name, err)
			}
		}
	}

	fmt.Println()
	fmt.Println("Cleanup complete.")

	return nil
}

