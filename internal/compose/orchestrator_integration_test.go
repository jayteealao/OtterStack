//go:build integration
// +build integration

package compose

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRealDockerPull tests image pulling with real Docker
func TestRealDockerPull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	// Create a temporary directory with a simple compose file
	tmpDir := t.TempDir()

	composeContent := `
services:
  test:
    image: alpine:latest
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()

	manager := NewManager(tmpDir, "compose.yaml", "test-pull")
	manager.SetOutputStreams(stdout, stderr)

	ctx := context.Background()
	err := manager.Pull(ctx)

	assert.NoError(t, err)

	// Verify we got Docker output
	output := stderr.String()
	t.Logf("Pull output: %s", output)

	// Docker should output something about pulling or having the image
	assert.True(t, len(output) > 0, "Should have some output from Docker")
}

// TestRealDockerUp tests starting services with real Docker
func TestRealDockerUp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	// Create a simple compose file
	tmpDir := t.TempDir()

	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 10
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()

	manager := NewManager(tmpDir, "compose.yaml", "test-up")
	manager.SetOutputStreams(stdout, stderr)

	ctx := context.Background()

	// Test Up
	err := manager.Up(ctx, "")
	assert.NoError(t, err)

	// Cleanup: always stop containers even if test fails
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager.Down(cleanupCtx, true)
	}()

	output := stderr.String()
	t.Logf("Up output: %s", output)

	// Verify output contains container info
	assert.True(t, len(output) > 0, "Should have output from Docker")

	// Verify service is running
	services, err := manager.Status(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, services, "Should have running services")
}

// TestRealDockerDown tests stopping services
func TestRealDockerDown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	// Create and start a simple service first
	tmpDir := t.TempDir()

	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 30
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-down")

	ctx := context.Background()

	// Start the service first
	err := manager.Up(ctx, "")
	require.NoError(t, err)

	// Now test Down with output capture
	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()
	manager.SetOutputStreams(stdout, stderr)

	err = manager.Down(ctx, true)
	assert.NoError(t, err)

	output := stderr.String()
	t.Logf("Down output: %s", output)

	assert.True(t, len(output) > 0, "Should have output from Docker")
}

// TestContextCancellation tests that context cancellation works properly
func TestContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	// Create a compose file with a service that takes time to start
	tmpDir := t.TempDir()

	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 300
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-cancel")

	// Cleanup: ensure we stop any containers that might have started
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager.Down(cleanupCtx, true)
	}()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 1 second
	go func() {
		time.Sleep(1 * time.Second)
		cancel()
	}()

	err := manager.Up(ctx, "")

	assert.Error(t, err, "Should get error when context is cancelled")
	assert.Contains(t, err.Error(), "cancelled", "Error should mention cancellation")

	// Verify context error is wrapped
	assert.ErrorIs(t, err, context.Canceled)
}

// TestRealDockerRestart tests restarting services
func TestRealDockerRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	// Create and start a service
	tmpDir := t.TempDir()

	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 60
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-restart")

	ctx := context.Background()

	// Start the service first
	err := manager.Up(ctx, "")
	require.NoError(t, err)

	// Cleanup
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager.Down(cleanupCtx, true)
	}()

	// Now test Restart with output capture
	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()
	manager.SetOutputStreams(stdout, stderr)

	err = manager.Restart(ctx)
	assert.NoError(t, err)

	output := stderr.String()
	t.Logf("Restart output: %s", output)

	assert.True(t, len(output) > 0, "Should have output from Docker")
}

// TestErrorHandling tests that errors are properly reported
func TestErrorHandling(t *testing.T) {
	// This test doesn't need Docker
	manager := NewManager("/nonexistent", "compose.yaml", "test")

	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()
	manager.SetOutputStreams(stdout, stderr)

	ctx := context.Background()
	err := manager.Up(ctx, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compose up failed")
}

// TestCancelledContextError tests immediate cancellation
func TestCancelledContextError(t *testing.T) {
	tmpDir := t.TempDir()

	composeContent := `
services:
  test:
    image: alpine:latest
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-cancelled")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := manager.Pull(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	assert.ErrorIs(t, err, context.Canceled)
}
