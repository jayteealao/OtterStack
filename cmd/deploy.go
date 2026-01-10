package cmd

import (
	"errors"
	"fmt"
	"time"

	apperrors "github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/orchestrator"
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
		if project.Status == "unconfigured" {
			return fmt.Errorf("project %q is not ready for deployment\n\nThe project needs validation first:\n  1. Set environment variables: otterstack env set %s KEY=value\n  2. Validate configuration: otterstack project validate %s\n  3. Then deploy: otterstack deploy %s", projectName, projectName, projectName, projectName)
		}
		return fmt.Errorf("project is not ready (status: %s)", project.Status)
	}

	// Get data directory
	dataDir, err := getDataDir()
	if err != nil {
		return err
	}

	// Initialize git manager and deployer
	gitMgr := git.NewManager(project.RepoPath)
	deployer := orchestrator.NewDeployer(store, gitMgr)

	// Deploy
	result, err := deployer.Deploy(ctx, project, orchestrator.DeployOptions{
		GitRef:    gitRef,
		Timeout:   deployTimeoutFlag,
		SkipPull:  skipPullFlag,
		DataDir:   dataDir,
		OnStatus:  func(msg string) { fmt.Println(msg) },
		OnVerbose: func(msg string) { printVerbose("%s", msg) },
	})
	if err != nil {
		return err
	}

	fmt.Printf("Deployment successful! %s deployed at %s\n", projectName, result.ShortSHA)

	// Clean up old worktrees if retention limit exceeded
	if project.WorktreeRetention > 0 {
		if err := deployer.CleanupOldWorktrees(ctx, project, dataDir, func(msg string) { printVerbose("%s", msg) }); err != nil {
			printVerbose("Warning: failed to cleanup old worktrees: %v", err)
		}
	}

	return nil
}

