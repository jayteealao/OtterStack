package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jayteealao/otterstack/internal/compose"
	"github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [project]",
	Short: "Show deployment status",
	Long: `Show the status of deployments.

Without arguments, shows status of all projects.
With a project name, shows detailed status for that project.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

var (
	statusServicesFlag bool
)

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVarP(&statusServicesFlag, "services", "s", false, "show service status")
}

func runStatus(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return showAllProjectsStatus(cmd)
	}

	return showProjectStatus(cmd, args[0])
}

func showAllProjectsStatus(cmd *cobra.Command) error {
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
	fmt.Fprintln(w, "PROJECT\tSTATUS\tDEPLOYMENT\tREF\tSERVICES")
	fmt.Fprintln(w, "-------\t------\t----------\t---\t--------")

	for _, p := range projects {
		deployment, err := store.GetActiveDeployment(ctx, p.ID)
		if err != nil && err != errors.ErrNoActiveDeployment {
			return fmt.Errorf("failed to get deployment for %s: %w", p.Name, err)
		}

		deploymentInfo := "-"
		refInfo := "-"
		servicesInfo := "-"

		if deployment != nil {
			deploymentInfo = git.ShortSHA(deployment.GitSHA)
			if deployment.GitRef != "" {
				refInfo = deployment.GitRef
			}

			// Get service status
			projectName := compose.GenerateProjectName(p.Name, git.ShortSHA(deployment.GitSHA))
			services, err := compose.GetProjectStatus(ctx, projectName)
			if err == nil && len(services) > 0 {
				running := 0
				for _, s := range services {
					if isServiceRunning(s.Status) {
						running++
					}
				}
				servicesInfo = fmt.Sprintf("%d/%d running", running, len(services))
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			p.Name, p.Status, deploymentInfo, refInfo, servicesInfo)
	}
	w.Flush()

	return nil
}

func showProjectStatus(cmd *cobra.Command, projectName string) error {
	ctx := cmd.Context()

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	project, err := store.GetProject(ctx, projectName)
	if err != nil {
		if err == errors.ErrProjectNotFound {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	fmt.Printf("Project: %s\n", project.Name)
	fmt.Printf("Status:  %s\n", project.Status)
	fmt.Printf("Type:    %s\n", project.RepoType)
	fmt.Printf("Path:    %s\n", project.RepoPath)
	if project.RepoURL != "" {
		fmt.Printf("URL:     %s\n", project.RepoURL)
	}
	fmt.Printf("Compose: %s\n", project.ComposeFile)
	fmt.Println()

	// Get active deployment
	deployment, err := store.GetActiveDeployment(ctx, project.ID)
	if err != nil {
		if err == errors.ErrNoActiveDeployment {
			fmt.Println("No active deployment.")
			return nil
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	fmt.Println("Active Deployment:")
	fmt.Printf("  Commit:     %s\n", git.ShortSHA(deployment.GitSHA))
	if deployment.GitRef != "" {
		fmt.Printf("  Ref:        %s\n", deployment.GitRef)
	}
	fmt.Printf("  Started:    %s\n", deployment.StartedAt.Format("2006-01-02 15:04:05"))
	if deployment.FinishedAt != nil {
		fmt.Printf("  Finished:   %s\n", deployment.FinishedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("  Worktree:   %s\n", deployment.WorktreePath)
	fmt.Println()

	// Get service status
	if statusServicesFlag {
		fmt.Println("Services:")
		composeProjectName := compose.GenerateProjectName(projectName, git.ShortSHA(deployment.GitSHA))
		services, err := compose.GetProjectStatus(ctx, composeProjectName)
		if err != nil {
			fmt.Printf("  Error getting services: %v\n", err)
		} else if len(services) == 0 {
			fmt.Println("  No services running.")
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  NAME\tSTATUS\tHEALTH")
			for _, s := range services {
				health := s.Health
				if health == "" {
					health = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\n", s.Name, s.Status, health)
			}
			w.Flush()
		}
		fmt.Println()
	}

	// Show recent deployments
	deployments, err := store.ListDeployments(ctx, project.ID, 5)
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments) > 1 {
		fmt.Println("Recent Deployments:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  COMMIT\tSTATUS\tREF\tSTARTED")
		for _, d := range deployments {
			ref := d.GitRef
			if ref == "" {
				ref = "-"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
				git.ShortSHA(d.GitSHA),
				d.Status,
				ref,
				d.StartedAt.Format("2006-01-02 15:04:05"))
		}
		w.Flush()
	}

	return nil
}

func isServiceRunning(status string) bool {
	return status == "running" || status == "Up" || len(status) > 0 && status[0] == 'U'
}
