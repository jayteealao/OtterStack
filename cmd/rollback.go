package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
	apperrors "github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/jayteealao/otterstack/internal/validate"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <project>",
	Short: "Rollback to the previous deployment",
	Long: `Rollback a project to its previous successful deployment.

This stops the current deployment and starts the previous one.

Examples:
  otterstack rollback myapp                  # Rollback to previous deployment
  otterstack rollback myapp --to abc123d     # Rollback to specific SHA`,
	Args: cobra.ExactArgs(1),
	RunE: runRollback,
}

var rollbackToFlag string

func init() {
	rootCmd.AddCommand(rollbackCmd)
	rollbackCmd.Flags().StringVar(&rollbackToFlag, "to", "", "rollback to specific SHA")
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
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	// Get current active deployment
	currentDeployment, err := store.GetActiveDeployment(ctx, project.ID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNoActiveDeployment) {
			return fmt.Errorf("no active deployment to rollback from")
		}
		return err
	}

	// Determine target deployment
	var targetDeployment *state.Deployment

	if rollbackToFlag != "" {
		if err := validate.GitRef(rollbackToFlag); err != nil {
			return fmt.Errorf("invalid git ref for --to flag: %w", err)
		}
		// Rollback to specific SHA
		targetDeployment, err = store.GetDeploymentBySHA(ctx, project.ID, rollbackToFlag)
		if err != nil {
			return fmt.Errorf("cannot find deployment with SHA %s: %w", rollbackToFlag, err)
		}
		if targetDeployment.ID == currentDeployment.ID {
			return fmt.Errorf("cannot rollback to current active deployment")
		}
	} else {
		// Get previous deployment
		targetDeployment, err = store.GetPreviousDeployment(ctx, project.ID)
		if err != nil {
			if errors.Is(err, apperrors.ErrNoPreviousDeployment) {
				return fmt.Errorf("no previous deployment to rollback to")
			}
			return err
		}
	}

	fmt.Printf("Rolling back %s from %s to %s\n",
		projectName,
		git.ShortSHA(currentDeployment.GitSHA),
		git.ShortSHA(targetDeployment.GitSHA))

	// Check that target commit still exists
	gitMgr := git.NewManager(project.RepoPath)
	if !gitMgr.CommitExists(ctx, targetDeployment.GitSHA) {
		return fmt.Errorf("target deployment commit %s no longer exists in repository", git.ShortSHA(targetDeployment.GitSHA))
	}

	// Start the target deployment
	targetProjectName := compose.GenerateProjectName(projectName, git.ShortSHA(targetDeployment.GitSHA))
	composeMgr := compose.NewManager(targetDeployment.WorktreePath, project.ComposeFile, targetProjectName)

	// Validate compose file
	if err := composeMgr.Validate(ctx); err != nil {
		return fmt.Errorf("compose validation failed: %w", err)
	}

	// Write env file if project has env vars
	dataDir, err := getDataDir()
	if err != nil {
		return err
	}

	envVars, err := store.GetEnvVars(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("failed to get env vars: %w", err)
	}

	envFilePath, err := writeRollbackEnvFile(dataDir, projectName, envVars)
	if err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}
	if envFilePath != "" {
		printVerbose("Using env file: %s", envFilePath)
	}

	fmt.Println("Starting target deployment...")
	if err := composeMgr.Up(ctx, envFilePath); err != nil {
		return fmt.Errorf("failed to start target deployment: %w", err)
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
		GitSHA:       targetDeployment.GitSHA,
		GitRef:       targetDeployment.GitRef,
		WorktreePath: targetDeployment.WorktreePath,
		Status:       "active",
	}
	if err := store.CreateDeployment(ctx, rollbackDeployment); err != nil {
		return fmt.Errorf("failed to create rollback deployment record: %w", err)
	}

	fmt.Printf("Rollback successful! %s now running at %s\n", projectName, git.ShortSHA(targetDeployment.GitSHA))

	return nil
}

// writeRollbackEnvFile writes environment variables to a file in dotenv format.
// Returns the file path if env vars exist, empty string otherwise.
func writeRollbackEnvFile(dataDir, projectName string, vars map[string]string) (string, error) {
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
		v := vars[k]
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, v); err != nil {
			return "", fmt.Errorf("failed to write env var: %w", err)
		}
	}

	return envPath, nil
}
