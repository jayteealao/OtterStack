package cmd

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jayteealao/otterstack/internal/tui"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Launch the TUI dashboard",
	Long: `Launch an interactive terminal dashboard for monitoring all projects.

The dashboard shows:
- All registered projects
- Current deployment status
- Service health status
- Real-time updates

Navigation:
  ↑/↓     Navigate projects
  Enter   View project details
  Esc     Go back
  r       Refresh
  q       Quit`,
	RunE: runMonitor,
}

var monitorRefreshFlag time.Duration

func init() {
	rootCmd.AddCommand(monitorCmd)

	monitorCmd.Flags().DurationVar(&monitorRefreshFlag, "refresh", 5*time.Second, "refresh interval")
}

func runMonitor(cmd *cobra.Command, args []string) error {
	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	model := tui.NewModel(cmd.Context(), store, monitorRefreshFlag)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}
