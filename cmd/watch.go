package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
	"github.com/jayteealao/otterstack/internal/git"
	"github.com/jayteealao/otterstack/internal/notify"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [project]",
	Short: "Continuously monitor project health",
	Long: `Continuously monitor project health and send notifications on status changes.

If no project is specified, all projects are monitored.

Notifications can be sent via:
  - Webhook: --webhook-url <url>
  - Discord: --discord-webhook <url>
  - Slack:   --slack-webhook <url>

Examples:
  otterstack watch                              # Watch all projects
  otterstack watch myapp                        # Watch specific project
  otterstack watch --interval 10s               # Check every 10 seconds
  otterstack watch --webhook-url http://...     # Send webhook notifications
  otterstack watch --discord-webhook https://...# Send Discord notifications`,
	RunE: runWatch,
}

var (
	watchIntervalFlag      time.Duration
	watchWebhookURLFlag    string
	watchDiscordFlag       string
	watchSlackFlag         string
	watchSlackChannelFlag  string
)

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().DurationVar(&watchIntervalFlag, "interval", 30*time.Second, "health check interval")
	watchCmd.Flags().StringVar(&watchWebhookURLFlag, "webhook-url", "", "webhook URL for notifications")
	watchCmd.Flags().StringVar(&watchDiscordFlag, "discord-webhook", "", "Discord webhook URL")
	watchCmd.Flags().StringVar(&watchSlackFlag, "slack-webhook", "", "Slack webhook URL")
	watchCmd.Flags().StringVar(&watchSlackChannelFlag, "slack-channel", "", "Slack channel (optional)")
}

// ServiceState tracks the state of a service for change detection.
type ServiceState struct {
	Status string
	Health string
}

// ProjectState tracks the state of a project's services.
type ProjectState struct {
	ProjectName string
	Services    map[string]ServiceState
}

func runWatch(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Initialize store
	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Initialize notification manager
	notifyMgr := notify.NewManager()
	defer notifyMgr.Close()

	if watchWebhookURLFlag != "" {
		notifyMgr.Register(notify.NewWebhookNotifier(watchWebhookURLFlag, nil))
		printVerbose("Registered webhook notifier")
	}

	if watchDiscordFlag != "" {
		notifyMgr.Register(notify.NewDiscordNotifier(watchDiscordFlag, "OtterStack"))
		printVerbose("Registered Discord notifier")
	}

	if watchSlackFlag != "" {
		notifyMgr.Register(notify.NewSlackNotifier(watchSlackFlag, watchSlackChannelFlag, "OtterStack"))
		printVerbose("Registered Slack notifier")
	}

	// Determine which projects to watch
	var projectFilter string
	if len(args) > 0 {
		projectFilter = args[0]
	}

	// Track previous state for change detection
	previousState := make(map[string]*ProjectState)

	fmt.Printf("Starting health watch (interval: %s)\n", watchIntervalFlag)
	if notifyMgr.Count() > 0 {
		fmt.Printf("Notifications enabled: %d backend(s)\n", notifyMgr.Count())
	} else {
		fmt.Println("No notification backends configured (use --webhook-url, --discord-webhook, or --slack-webhook)")
	}
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	ticker := time.NewTicker(watchIntervalFlag)
	defer ticker.Stop()

	// Do initial check immediately
	if err := checkHealth(ctx, store, notifyMgr, projectFilter, previousState); err != nil {
		printVerbose("Health check error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := checkHealth(ctx, store, notifyMgr, projectFilter, previousState); err != nil {
				printVerbose("Health check error: %v", err)
			}
		}
	}
}

func checkHealth(ctx context.Context, store *state.Store, notifyMgr *notify.Manager, projectFilter string, previousState map[string]*ProjectState) error {
	projects, err := store.ListProjects(ctx)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	timestamp := time.Now().Format("15:04:05")

	for _, project := range projects {
		// Filter if specified
		if projectFilter != "" && project.Name != projectFilter {
			continue
		}

		// Get active deployment
		deployment, err := store.GetActiveDeployment(ctx, project.ID)
		if err != nil {
			continue // No active deployment
		}

		// Get service status
		projectName := compose.GenerateProjectName(project.Name, git.ShortSHA(deployment.GitSHA))
		services, err := compose.GetProjectStatus(ctx, projectName)
		if err != nil {
			printVerbose("[%s] %s: failed to get status: %v", timestamp, project.Name, err)
			continue
		}

		// Initialize previous state if needed
		if previousState[project.Name] == nil {
			previousState[project.Name] = &ProjectState{
				ProjectName: project.Name,
				Services:    make(map[string]ServiceState),
			}
		}
		prevState := previousState[project.Name]

		// Check each service for changes
		for _, svc := range services {
			prev, exists := prevState.Services[svc.Name]
			current := ServiceState{Status: svc.Status, Health: svc.Health}

			// Detect status changes
			if exists && (prev.Status != current.Status || prev.Health != current.Health) {
				event := detectEvent(project.Name, svc.Name, prev, current)
				if event != nil {
					// Log the change
					fmt.Printf("[%s] %s/%s: %s -> %s",
						timestamp, project.Name, svc.Name, prev.Status, current.Status)
					if current.Health != "" {
						fmt.Printf(" (health: %s)", current.Health)
					}
					fmt.Println()

					// Send notification
					if notifyMgr.Count() > 0 {
						if err := notifyMgr.Notify(ctx, *event); err != nil {
							printVerbose("Notification error: %v", err)
						}
					}
				}
			} else if !exists {
				// First time seeing this service
				fmt.Printf("[%s] %s/%s: %s", timestamp, project.Name, svc.Name, current.Status)
				if current.Health != "" {
					fmt.Printf(" (health: %s)", current.Health)
				}
				fmt.Println()
			}

			// Update state
			prevState.Services[svc.Name] = current
		}

		// Check for removed services
		for name := range prevState.Services {
			found := false
			for _, svc := range services {
				if svc.Name == name {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("[%s] %s/%s: service removed\n", timestamp, project.Name, name)
				delete(prevState.Services, name)
			}
		}
	}

	return nil
}

func detectEvent(projectName, serviceName string, prev, current ServiceState) *notify.Event {
	// Detect service going down
	if compose.IsServiceRunning(prev.Status) && !compose.IsServiceRunning(current.Status) {
		return &notify.Event{
			Type:    notify.EventServiceDown,
			Project: projectName,
			Service: serviceName,
			Status:  current.Status,
			Message: fmt.Sprintf("Service went from %s to %s", prev.Status, current.Status),
		}
	}

	// Detect service coming up
	if !compose.IsServiceRunning(prev.Status) && compose.IsServiceRunning(current.Status) {
		return &notify.Event{
			Type:    notify.EventServiceUp,
			Project: projectName,
			Service: serviceName,
			Status:  current.Status,
			Message: fmt.Sprintf("Service went from %s to %s", prev.Status, current.Status),
		}
	}

	// Detect health changes
	if prev.Health != current.Health {
		if current.Health == "unhealthy" {
			return &notify.Event{
				Type:    notify.EventServiceUnhealthy,
				Project: projectName,
				Service: serviceName,
				Status:  current.Status,
				Message: fmt.Sprintf("Service health: %s -> %s", prev.Health, current.Health),
			}
		}
		if current.Health == "healthy" && prev.Health == "unhealthy" {
			return &notify.Event{
				Type:    notify.EventServiceRecovered,
				Project: projectName,
				Service: serviceName,
				Status:  current.Status,
				Message: fmt.Sprintf("Service health: %s -> %s", prev.Health, current.Health),
			}
		}
	}

	return nil
}
