package traefik

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// DefaultHealthTimeout is the default timeout for health checks.
	DefaultHealthTimeout = 5 * time.Minute
	// healthCheckInterval is how often to poll container health.
	healthCheckInterval = 2 * time.Second
)

// WaitForHealthy waits for all containers in a compose project to become healthy.
// It polls the container health status at regular intervals until the timeout is reached.
func WaitForHealthy(ctx context.Context, composeProject string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check timeout after %v", timeout)
		case <-ticker.C:
			healthy, err := checkHealth(ctx, composeProject)
			if err != nil {
				return fmt.Errorf("health check failed: %w", err)
			}
			if healthy {
				return nil
			}
		}
	}
}

// checkHealth checks the health status of all containers in a compose project.
// Returns true if all containers are healthy, false otherwise.
func checkHealth(ctx context.Context, composeProject string) (bool, error) {
	// Get container health status
	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-p", composeProject,
		"ps", "--format", "{{.Name}}\t{{.Health}}")

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check container health: %w", err)
	}

	// Parse output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		health := strings.TrimSpace(parts[1])
		// Consider: empty (no health check), "starting", or "unhealthy" as not ready
		if health == "" || health == "starting" || health == "unhealthy" {
			return false, nil
		}
	}

	return true, nil // All containers have health status
}
