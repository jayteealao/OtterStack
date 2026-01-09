package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/jayteealao/otterstack/internal/tui"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history <project>",
	Short: "Show deployment history",
	Long: `Show the deployment history for a project.

Displays recent deployments with their SHA, ref, status, and timestamp.`,
	Args: cobra.ExactArgs(1),
	RunE: runHistory,
}

var (
	historyLimitFlag int
	historyJSONFlag  bool
)

func init() {
	rootCmd.AddCommand(historyCmd)

	historyCmd.Flags().IntVarP(&historyLimitFlag, "limit", "n", 20, "number of deployments to show")
	historyCmd.Flags().BoolVar(&historyJSONFlag, "json", false, "output in JSON format")
}

type historyEntry struct {
	ID          string  `json:"id"`
	GitSHA      string  `json:"git_sha"`
	ShortSHA    string  `json:"short_sha"`
	GitRef      string  `json:"git_ref,omitempty"`
	Status      string  `json:"status"`
	StartedAt   string  `json:"started_at"`
	FinishedAt  *string `json:"finished_at,omitempty"`
	Error       string  `json:"error,omitempty"`
	WorktreePath string `json:"worktree_path,omitempty"`
}

func runHistory(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	projectName := args[0]

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Get project
	project, err := store.GetProject(ctx, projectName)
	if err != nil {
		if err == errors.ErrProjectNotFound {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	// Get deployments
	deployments, err := store.ListDeployments(ctx, project.ID, historyLimitFlag)
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments) == 0 {
		fmt.Printf("No deployments found for project %q.\n", projectName)
		return nil
	}

	if historyJSONFlag {
		return outputHistoryJSON(deployments)
	}

	return outputHistoryTable(projectName, deployments)
}

func outputHistoryJSON(deployments []*state.Deployment) error {
	var entries []historyEntry
	for _, d := range deployments {
		entry := historyEntry{
			ID:           d.ID,
			GitSHA:       d.GitSHA,
			ShortSHA:     git.ShortSHA(d.GitSHA),
			GitRef:       d.GitRef,
			Status:       d.Status,
			StartedAt:    d.StartedAt.Format("2006-01-02T15:04:05Z"),
			Error:        d.ErrorMessage,
			WorktreePath: d.WorktreePath,
		}
		if d.FinishedAt != nil {
			finishedStr := d.FinishedAt.Format("2006-01-02T15:04:05Z")
			entry.FinishedAt = &finishedStr
		}
		entries = append(entries, entry)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

func outputHistoryTable(projectName string, deployments []*state.Deployment) error {
	fmt.Printf("Deployment history for %s:\n\n", projectName)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  COMMIT\tREF\tSTATUS\tSTARTED\tDURATION")
	fmt.Fprintln(w, "  ------\t---\t------\t-------\t--------")

	for _, d := range deployments {
		ref := d.GitRef
		if ref == "" {
			ref = "-"
		}

		duration := "-"
		if d.FinishedAt != nil {
			dur := d.FinishedAt.Sub(d.StartedAt)
			if dur.Seconds() < 60 {
				duration = fmt.Sprintf("%.0fs", dur.Seconds())
			} else {
				duration = fmt.Sprintf("%.1fm", dur.Minutes())
			}
		}

		statusIcon := tui.GetStatusIcon(d.Status)

		fmt.Fprintf(w, "  %s\t%s\t%s %s\t%s\t%s\n",
			git.ShortSHA(d.GitSHA),
			ref,
			statusIcon,
			d.Status,
			d.StartedAt.Format("2006-01-02 15:04"),
			duration)
	}
	w.Flush()

	return nil
}
