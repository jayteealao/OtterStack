// Package traefik provides Traefik integration for OtterStack.
package traefik

import (
	"context"
	"os/exec"
	"strings"
)

// IsRunning checks if a Traefik container is currently running.
// Returns true if Traefik is detected, false otherwise.
// Does not return an error if Traefik is not found - degraded mode is supported.
func IsRunning(ctx context.Context) (bool, error) {
	// Check for Traefik container
	cmd := exec.CommandContext(ctx, "docker", "ps",
		"--filter", "name=traefik",
		"--filter", "status=running",
		"--format", "{{.Names}}")

	output, err := cmd.Output()
	if err != nil {
		// Traefik not found is not an error - deployment can proceed without routing
		return false, nil
	}

	// Check if "traefik" appears in the output
	return strings.Contains(string(output), "traefik"), nil
}
