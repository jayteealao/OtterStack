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
// Returns true if all containers are healthy or running (for containers without healthchecks), false otherwise.
func checkHealth(ctx context.Context, composeProject string) (bool, error) {
	// Get container status and health
	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-p", composeProject,
		"ps", "--format", "{{.Name}}\t{{.Status}}\t{{.Health}}")

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
		if len(parts) < 3 {
			continue // Need at least Name, Status, Health
		}

		status := strings.TrimSpace(parts[1])
		health := strings.TrimSpace(parts[2])

		// If container has healthcheck defined (health not empty)
		if health != "" {
			// Must be healthy (not starting or unhealthy)
			if health == "starting" || health == "unhealthy" {
				return false, nil // Health check exists but not healthy
			}
			// health == "healthy" → continue to next container
		} else {
			// No healthcheck defined - check if container is at least running
			// Status can be: "Up", "Up X seconds", "running", etc.
			// Docker uses "Up" for compose ps, "running" for docker ps
			if !strings.HasPrefix(status, "Up") && status != "running" {
				return false, nil // Container not running
			}
		}
	}

	return true, nil // All containers ready (either healthy or running)
}
