package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jayteealao/otterstack/internal/compose"
	"github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/jayteealao/otterstack/internal/validate"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  `Add, list, and remove projects from OtterStack.`,
}

var projectAddCmd = &cobra.Command{
	Use:   "add <name> <repo-path-or-url>",
	Short: "Add a new project",
	Long: `Add a new project to OtterStack.

The repository can be:
- A local path to an existing git repository (e.g., /srv/myapp)
- A remote git URL (https:// or git@) which will be cloned

Examples:
  otterstack project add myapp /srv/myapp
  otterstack project add myapp https://github.com/user/repo.git
  otterstack project add myapp git@github.com:user/repo.git`,
	Args: cobra.ExactArgs(2),
	RunE: runProjectAdd,
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all projects",
	RunE:    runProjectList,
}

var projectRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove a project",
	Long: `Remove a project from OtterStack.

This stops any running services and removes the project from tracking.
Use --force to also remove worktrees and cloned repositories.`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectRemove,
}

var (
	composeFileFlag string
	retentionFlag   int
	forceFlag       bool
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectRemoveCmd)

	// Add flags
	projectAddCmd.Flags().StringVarP(&composeFileFlag, "compose-file", "f", "", "compose file name (default: auto-detect)")
	projectAddCmd.Flags().IntVar(&retentionFlag, "retention", 3, "number of worktrees to retain")

	projectRemoveCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "force removal including worktrees and cloned repos")
}

func runProjectAdd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]
	repoArg := args[1]

	// Validate project name
	if err := validate.ProjectName(name); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Check if project already exists
	_, err = store.GetProject(ctx, name)
	if err == nil {
		return fmt.Errorf("project %q already exists", name)
	}
	if err != errors.ErrProjectNotFound {
		return err
	}

	// Determine if URL or path
	isRemote := validate.IsURL(repoArg)

	var repoType, repoURL, repoPath string

	if isRemote {
		// Remote repository
		if err := validate.RepoURL(repoArg); err != nil {
			return fmt.Errorf("invalid repository URL: %w", err)
		}

		repoType = "remote"
		repoURL = repoArg

		// Clone path will be managed by OtterStack
		dataDir, err := getDataDir()
		if err != nil {
			return err
		}
		repoPath = fmt.Sprintf("%s/repos/%s", dataDir, name)

		// Clone the repository
		fmt.Printf("Cloning repository %s...\n", repoURL)
		gitMgr := git.NewManager(repoPath)
		if err := gitMgr.Clone(ctx, repoURL); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		fmt.Println("Clone complete.")
	} else {
		// Local repository
		if err := validate.RepoPath(repoArg); err != nil {
			return fmt.Errorf("invalid repository path: %w", err)
		}

		repoType = "local"
		repoPath = repoArg
	}

	// Find compose file
	composeFile, err := validate.FindComposeFile(repoPath, composeFileFlag)
	if err != nil {
		return fmt.Errorf("compose file error: %w", err)
	}

	// Validate compose file
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	printVerbose("Validating compose file: %s", composeFile)
	composeMgr := compose.NewManager(repoPath, composeFile, name)
	if err := composeMgr.Validate(ctx); err != nil {
		return fmt.Errorf("compose validation failed: %w", err)
	}

	// Create project record
	project := &state.Project{
		Name:              name,
		RepoType:          repoType,
		RepoURL:           repoURL,
		RepoPath:          repoPath,
		ComposeFile:       composeFile,
		WorktreeRetention: retentionFlag,
		Status:            "ready",
	}

	if err := store.CreateProject(ctx, project); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	fmt.Printf("Project %q added successfully.\n", name)
	fmt.Printf("  Repository: %s (%s)\n", repoPath, repoType)
	fmt.Printf("  Compose file: %s\n", composeFile)
	fmt.Printf("  Worktree retention: %d\n", retentionFlag)

	return nil
}

func runProjectList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	projects, err := store.ListProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects configured.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tSTATUS\tPATH")
	fmt.Fprintln(w, "----\t----\t------\t----")

	for _, p := range projects {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.RepoType, p.Status, p.RepoPath)
	}
	w.Flush()

	return nil
}

func runProjectRemove(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Get project
	project, err := store.GetProject(ctx, name)
	if err != nil {
		if err == errors.ErrProjectNotFound {
			return fmt.Errorf("project %q not found", name)
		}
		return err
	}

	// Check for active deployment
	activeDeployment, err := store.GetActiveDeployment(ctx, project.ID)
	if err != nil && err != errors.ErrNoActiveDeployment {
		return fmt.Errorf("failed to check active deployment: %w", err)
	}

	if activeDeployment != nil {
		fmt.Printf("Stopping active deployment %s...\n", git.ShortSHA(activeDeployment.GitSHA))

		// Stop the compose services
		projectName := compose.GenerateProjectName(name, git.ShortSHA(activeDeployment.GitSHA))
		if err := compose.StopProjectByName(ctx, projectName, 0); err != nil {
			if !forceFlag {
				return fmt.Errorf("failed to stop services: %w (use --force to continue)", err)
			}
			fmt.Fprintf(os.Stderr, "Warning: failed to stop services: %v\n", err)
		}
	}

	if forceFlag {
		// Clean up worktrees
		printVerbose("Cleaning up worktrees...")
		dataDir, _ := getDataDir()
		worktreeDir := fmt.Sprintf("%s/worktrees/%s", dataDir, name)
		if err := os.RemoveAll(worktreeDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove worktrees: %v\n", err)
		}

		// If remote repo, remove cloned repo
		if project.RepoType == "remote" {
			printVerbose("Removing cloned repository...")
			if err := os.RemoveAll(project.RepoPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove cloned repo: %v\n", err)
			}
		}
	}

	// Delete project from database
	if err := store.DeleteProject(ctx, name); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	fmt.Printf("Project %q removed successfully.\n", name)
	return nil
}
