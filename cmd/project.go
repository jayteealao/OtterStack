package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jayteealao/otterstack/internal/compose"
	apperrors "github.com/jayteealao/otterstack/internal/errors"
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

var projectValidateCmd = &cobra.Command{
	Use:   "validate <name>",
	Short: "Validate project compose file and mark as ready",
	Long: `Validate the compose file for a project and mark it as ready for deployment.

This should be run after setting all required environment variables via 'otterstack env set'.

Examples:
  otterstack project validate myapp`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectValidate,
}

var projectCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up orphaned repositories",
	Long: `Scan for and optionally remove repositories that are not tracked as projects.

This happens when project addition fails after cloning but before database insertion.

Examples:
  otterstack project cleanup          # List orphaned repos
  otterstack project cleanup --force  # Remove orphaned repos`,
	RunE: runProjectCleanup,
}

var (
	composeFileFlag      string
	retentionFlag        int
	forceFlag            bool
	traefikRoutingFlag   bool
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectValidateCmd)
	projectCmd.AddCommand(projectCleanupCmd)

	// Add flags
	projectAddCmd.Flags().StringVarP(&composeFileFlag, "compose-file", "f", "", "compose file name (default: auto-detect)")
	projectAddCmd.Flags().IntVar(&retentionFlag, "retention", 3, "number of worktrees to retain")
	projectAddCmd.Flags().BoolVar(&traefikRoutingFlag, "traefik-routing", false, "enable Traefik priority-based routing for zero-downtime deployments")

	projectRemoveCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "force removal including worktrees and cloned repos")
	projectCleanupCmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "remove orphaned repositories")
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
	if !errors.Is(err, apperrors.ErrProjectNotFound) {
		return err
	}

	// Create project record FIRST (enables env var setting and rollback on failure)
	project := &state.Project{
		Name:                  name,
		RepoType:              "",  // Set after determining local/remote
		RepoURL:               "",
		RepoPath:              "",
		ComposeFile:           "",
		WorktreeRetention:     retentionFlag,
		Status:                "cloning", // Initial status
		TraefikRoutingEnabled: traefikRoutingFlag,
	}

	if err := store.CreateProject(ctx, project); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Setup rollback on any future error
	projectCreated := true
	defer func() {
		if !projectCreated {
			// Cleanup on failure
			if err := store.DeleteProject(ctx, name); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to cleanup project: %v\n", err)
			}
		}
	}()

	// Determine if URL or path
	isRemote := validate.IsURL(repoArg)

	var repoType, repoURL, repoPath string

	if isRemote {
		// Remote repository
		if err := validate.RepoURL(repoArg); err != nil {
			projectCreated = false
			return fmt.Errorf("invalid repository URL: %w", err)
		}

		repoType = "remote"
		repoURL = repoArg

		// Pre-flight auth check
		printVerbose("Checking repository access...")
		if err := git.CheckAuth(ctx, repoURL); err != nil {
			projectCreated = false
			return fmt.Errorf("cannot access repository: %w", err)
		}

		// Clone path will be managed by OtterStack
		dataDir, err := getDataDir()
		if err != nil {
			projectCreated = false
			return err
		}
		repoPath = fmt.Sprintf("%s/repos/%s", dataDir, name)

		// Update project with repo details
		if err := store.UpdateProjectRepo(ctx, name, repoType, repoURL, repoPath); err != nil {
			projectCreated = false
			return fmt.Errorf("failed to update project: %w", err)
		}

		// Clone the repository (uses atomic clone with temp dir)
		fmt.Printf("Cloning repository %s...\n", repoURL)
		gitMgr := git.NewManager(repoPath)
		if err := gitMgr.Clone(ctx, repoURL); err != nil {
			projectCreated = false // Trigger rollback
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		fmt.Println("Clone complete.")
	} else {
		// Local repository
		if err := validate.RepoPath(repoArg); err != nil {
			projectCreated = false
			return fmt.Errorf("invalid repository path: %w", err)
		}

		repoType = "local"
		repoPath = repoArg

		// Update project with repo details
		if err := store.UpdateProjectRepo(ctx, name, repoType, repoURL, repoPath); err != nil {
			projectCreated = false
			return fmt.Errorf("failed to update project: %w", err)
		}
	}

	// Find compose file
	composeFile, err := validate.FindComposeFile(repoPath, composeFileFlag)
	if err != nil {
		projectCreated = false
		return fmt.Errorf("compose file error: %w", err)
	}

	// Update project with compose file and status
	if err := store.UpdateProjectCompose(ctx, name, composeFile, "unconfigured"); err != nil {
		projectCreated = false
		return fmt.Errorf("failed to update project: %w", err)
	}

	// Success - don't rollback
	projectCreated = true

	fmt.Printf("Project %q added successfully (status: unconfigured)\n", name)
	fmt.Printf("  Repository: %s (%s)\n", repoPath, repoType)
	fmt.Printf("  Compose file: %s\n", composeFile)

	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Set environment variables: otterstack env set %s KEY=value\n", name)
	fmt.Printf("  2. Validate configuration: otterstack project validate %s\n", name)
	fmt.Printf("  3. Deploy: otterstack deploy %s\n", name)

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
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", name)
		}
		return err
	}

	// Check for active deployment
	activeDeployment, err := store.GetActiveDeployment(ctx, project.ID)
	if err != nil && !errors.Is(err, apperrors.ErrNoActiveDeployment) {
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

func runProjectValidate(cmd *cobra.Command, args []string) error {
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
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", name)
		}
		return err
	}

	// Check status
	if project.Status == "ready" {
		fmt.Printf("Project %q is already validated and ready.\n", name)
		return nil
	}

	if project.Status != "unconfigured" {
		return fmt.Errorf("project %q cannot be validated (status: %s)", name, project.Status)
	}

	// Get env vars to create temp env file
	dataDir, err := getDataDir()
	if err != nil {
		return err
	}

	envFilePath := ""
	envVars, err := store.GetEnvVars(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("failed to get env vars: %w", err)
	}

	if len(envVars) > 0 {
		// Create temp env file for validation
		envFilePath = fmt.Sprintf("%s/envfiles/%s.env", dataDir, name)
		if err := writeEnvFile(envFilePath, envVars); err != nil {
			return fmt.Errorf("failed to write env file: %w", err)
		}
		defer os.Remove(envFilePath)
	}

	// Validate compose file with env vars
	fmt.Printf("Validating compose file: %s\n", project.ComposeFile)
	composeMgr := compose.NewManager(project.RepoPath, project.ComposeFile, name)

	if err := composeMgr.ValidateWithEnv(ctx, envFilePath); err != nil {
		return fmt.Errorf("compose validation failed: %w\n\nTo fix:\n  1. Review the error above\n  2. Set missing env vars: otterstack env set %s KEY=value\n  3. Retry validation: otterstack project validate %s", err, name, name)
	}

	// Update status to ready
	if err := store.UpdateProjectStatus(ctx, name, "ready"); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	fmt.Printf("✓ Project %q validated successfully and marked as ready.\n", name)
	fmt.Printf("  Deploy with: otterstack deploy %s\n", name)
	return nil
}

func runProjectCleanup(cmd *cobra.Command, args []string) error {
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

	reposDir := fmt.Sprintf("%s/repos", dataDir)

	// List all directories in repos/
	entries, err := os.ReadDir(reposDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No repositories directory found.")
			return nil
		}
		return fmt.Errorf("failed to read repos directory: %w", err)
	}

	// Get all projects from database
	projects, err := store.ListProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	// Build map of tracked repos
	trackedRepos := make(map[string]bool)
	for _, p := range projects {
		if p.RepoType == "remote" {
			// Extract repo name from path
			repoName := filepath.Base(p.RepoPath)
			trackedRepos[repoName] = true
		}
	}

	// Find orphaned repos
	var orphaned []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !trackedRepos[entry.Name()] {
			orphaned = append(orphaned, entry.Name())
		}
	}

	if len(orphaned) == 0 {
		fmt.Println("No orphaned repositories found.")
		return nil
	}

	// List orphaned repos
	fmt.Println("Orphaned repositories:")
	for _, name := range orphaned {
		path := fmt.Sprintf("%s/%s", reposDir, name)
		fmt.Printf("  - %s (%s)\n", name, path)
	}

	if !forceFlag {
		fmt.Println("\nTo remove these repositories, run:")
		fmt.Println("  otterstack project cleanup --force")
		return nil
	}

	// Remove orphaned repos
	fmt.Println("\nRemoving orphaned repositories...")
	for _, name := range orphaned {
		path := fmt.Sprintf("%s/%s", reposDir, name)
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", path, err)
		} else {
			fmt.Printf("  ✓ Removed %s\n", name)
		}
	}

	return nil
}

// writeEnvFile writes environment variables to a file in key=value format.
func writeEnvFile(path string, envVars map[string]string) error {
	if len(envVars) == 0 {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create envfiles directory: %w", err)
	}

	// Build env file content (sorted for determinism)
	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var content strings.Builder
	for _, k := range keys {
		content.WriteString(fmt.Sprintf("%s=%s\n", k, envVars[k]))
	}

	// Write with restrictive permissions
	if err := os.WriteFile(path, []byte(content.String()), 0600); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}

	return nil
}
