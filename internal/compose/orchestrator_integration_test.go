//go:build integration
// +build integration

package compose

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jayteealao/otterstack/internal/errors"
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

// TestRealDockerPull_OutputContent validates Docker pull output contains expected content
func TestRealDockerPull_OutputContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

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

	manager := NewManager(tmpDir, "compose.yaml", "test-pull-content")
	manager.SetOutputStreams(stdout, stderr)

	ctx := context.Background()
	err := manager.Pull(ctx)

	require.NoError(t, err)

	output := stderr.String()
	t.Logf("Pull output: %s", output)

	// Verify output contains expected Docker messages
	assert.Contains(t, output, "alpine",
		"Should show image name being pulled")

	// Verify output shows pull status
	// Docker may say "Pull complete", "Already exists", or similar messages
	hasStatusIndicator :=
		strings.Contains(output, "Pull complete") ||
			strings.Contains(output, "Already exists") ||
			strings.Contains(output, "Pulled") ||
			strings.Contains(output, "pull")

	assert.True(t, hasStatusIndicator,
		"Should show pull status in output: %s", output)
}

// TestRealDockerUp_OutputContent validates Docker up output contains expected content
func TestRealDockerUp_OutputContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()

	composeContent := `
services:
  web:
    image: alpine:latest
    command: sleep 10
    container_name: test-output-web
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()

	manager := NewManager(tmpDir, "compose.yaml", "test-up-content")
	manager.SetOutputStreams(stdout, stderr)

	ctx := context.Background()

	err := manager.Up(ctx, "")
	require.NoError(t, err)

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager.Down(cleanupCtx, true)
	}()

	output := stderr.String()
	t.Logf("Up output: %s", output)

	// Verify service name appears in output
	assert.Contains(t, output, "web",
		"Should show service name in output")

	// Verify startup messages appear
	// Docker compose may show "Started", "Running", "Created", etc.
	hasStartupIndicator :=
		strings.Contains(output, "Started") ||
			strings.Contains(output, "Running") ||
			strings.Contains(output, "Created") ||
			strings.Contains(output, "start")

	assert.True(t, hasStartupIndicator,
		"Should show container startup status: %s", output)
}

// TestRealDockerDown_OutputContent validates Docker down output contains expected content
func TestRealDockerDown_OutputContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()

	composeContent := `
services:
  web:
    image: alpine:latest
    command: sleep 30
    container_name: test-down-output-web
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-down-content")

	ctx := context.Background()

	// Start the service first (without capturing output)
	err := manager.Up(ctx, "")
	require.NoError(t, err)

	// Now test Down with output capture
	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()
	manager.SetOutputStreams(stdout, stderr)

	err = manager.Down(ctx, true)
	require.NoError(t, err)

	output := stderr.String()
	t.Logf("Down output: %s", output)

	// Verify service name appears
	assert.Contains(t, output, "web",
		"Should show service name being stopped")

	// Verify removal/stop messages appear
	// Docker compose may show "Removed", "Stopped", "Removing", etc.
	hasRemovalIndicator :=
		strings.Contains(output, "Removed") ||
			strings.Contains(output, "Stopped") ||
			strings.Contains(output, "Removing") ||
			strings.Contains(output, "Stopping") ||
			strings.Contains(output, "removed") ||
			strings.Contains(output, "stopped")

	assert.True(t, hasRemovalIndicator,
		"Should show container removal/stop status: %s", output)
}

// TestErrorOutput_IncludesDockerDiagnostics validates error output includes Docker diagnostics
func TestErrorOutput_IncludesDockerDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()

	// Use a non-existent image to trigger a clear error
	composeContent := `
services:
  test:
    image: this-image-definitely-does-not-exist-test-12345:nonexistent
    pull_policy: always
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	stdout := NewSafeBuffer()
	stderr := NewSafeBuffer()

	manager := NewManager(tmpDir, "compose.yaml", "test-error-diagnostics")
	manager.SetOutputStreams(stdout, stderr)

	ctx := context.Background()
	err := manager.Up(ctx, "")

	require.Error(t, err)

	errMsg := err.Error()
	t.Logf("Error message: %s", errMsg)

	// Error should mention the operation that failed
	assert.Contains(t, errMsg, "compose up failed",
		"Should indicate which operation failed")

	// Error message should include diagnostic details from Docker
	stderrContent := stderr.String()
	t.Logf("Stderr content: %s", stderrContent)

	// Check that we captured some diagnostic output
	if stderrContent != "" {
		// If we got stderr output, verify it contains something useful
		hasImageReference := strings.Contains(stderrContent, "image") ||
			strings.Contains(stderrContent, "this-image-definitely-does-not-exist")

		hasErrorIndicator := strings.Contains(stderrContent, "error") ||
			strings.Contains(stderrContent, "Error") ||
			strings.Contains(stderrContent, "failed") ||
			strings.Contains(stderrContent, "not found")

		assert.True(t, hasImageReference || hasErrorIndicator,
			"Stderr should contain diagnostic information about the error")
	}
}

// TestRealDockerRestart_OutputContent validates Docker restart output contains expected content
func TestRealDockerRestart_OutputContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()

	composeContent := `
services:
  web:
    image: alpine:latest
    command: sleep 60
    container_name: test-restart-output-web
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-restart-content")

	ctx := context.Background()

	// Start the service first
	err := manager.Up(ctx, "")
	require.NoError(t, err)

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
	require.NoError(t, err)

	output := stderr.String()
	t.Logf("Restart output: %s", output)

	// Verify service name appears
	assert.Contains(t, output, "web",
		"Should show service name being restarted")

	// Verify restart indicators appear
	// Docker compose may show "Restarted", "Restarting", "restart", etc.
	hasRestartIndicator :=
		strings.Contains(output, "Restart") ||
			strings.Contains(output, "restart") ||
			strings.Contains(output, "Restarting") ||
			strings.Contains(output, "Restarted")

	assert.True(t, hasRestartIndicator,
		"Should show container restart status: %s", output)
}

// TestContextTimeout_Up tests that Up returns ErrComposeTimeout on deadline exceeded
func TestContextTimeout_Up(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()
	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 300
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-timeout-up")
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager.Down(cleanupCtx, true)
	}()

	// Use very short timeout to trigger DeadlineExceeded
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := manager.Up(ctx, "")

	// Should return timeout error
	assert.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrComposeTimeout, "Should return ErrComposeTimeout for timeouts")

	// Check the error message contains timeout indication
	assert.Contains(t, err.Error(), "timed out", "Error should mention timeout")
}

// TestContextCancellation_Up tests that Up returns cancellation error (NOT timeout) on manual cancel
func TestContextCancellation_Up(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()
	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 300
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-cancel-up")
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager.Down(cleanupCtx, true)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := manager.Up(ctx, "")

	// Should return cancellation error (NOT timeout)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "Should wrap context.Canceled")
	assert.Contains(t, err.Error(), "cancelled", "Error message should mention cancellation")

	// Should NOT be a timeout error
	assert.NotContains(t, err.Error(), "timed out", "Should not mention timeout")
	assert.NotErrorIs(t, err, errors.ErrComposeTimeout, "Should not return ErrComposeTimeout for cancellation")
}

// TestContextTimeout_Down tests that Down handles timeout correctly
func TestContextTimeout_Down(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()
	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 60
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-timeout-down")

	// Start the service first
	ctx := context.Background()
	err := manager.Up(ctx, "")
	require.NoError(t, err)

	// Use very short timeout for Down
	downCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = manager.Down(downCtx, true)

	// Down should handle timeout (wrapping context error)
	// Note: Down currently wraps ctx.Err() for both timeout and cancellation
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded, "Should wrap DeadlineExceeded if timeout occurs")
		assert.Contains(t, err.Error(), "cancelled", "Error message should mention context cancellation")
	}

	// Cleanup with proper timeout
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()
	manager.Down(cleanupCtx, true)
}

// TestContextCancellation_Down tests that Down returns cancellation error on manual cancel
func TestContextCancellation_Down(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()
	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 60
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-cancel-down")

	// Start the service first
	ctx := context.Background()
	err := manager.Up(ctx, "")
	require.NoError(t, err)

	downCtx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err = manager.Down(downCtx, true)

	// Should return cancellation error if it occurs
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled, "Should wrap context.Canceled")
		assert.Contains(t, err.Error(), "cancelled", "Error message should mention cancellation")
	}

	// Cleanup with proper timeout
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()
	manager.Down(cleanupCtx, true)
}

// TestContextTimeout_Pull tests that Pull handles timeout correctly
func TestContextTimeout_Pull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()
	composeContent := `
services:
  test:
    image: alpine:latest
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-timeout-pull")

	// Use very short timeout to trigger DeadlineExceeded
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := manager.Pull(ctx)

	// Pull should handle timeout (wrapping context error)
	// Note: Pull currently wraps ctx.Err() for both timeout and cancellation
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded, "Should wrap DeadlineExceeded if timeout occurs")
		assert.Contains(t, err.Error(), "cancelled", "Error message should mention context cancellation")
	}
}

// TestContextCancellation_Pull tests that Pull returns cancellation error on manual cancel
func TestContextCancellation_Pull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()
	composeContent := `
services:
  test:
    image: alpine:latest
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-cancel-pull")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := manager.Pull(ctx)

	// Should return cancellation error if it occurs
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled, "Should wrap context.Canceled")
		assert.Contains(t, err.Error(), "cancelled", "Error message should mention cancellation")
	}
}

// TestContextTimeout_Restart tests that Restart handles timeout correctly
func TestContextTimeout_Restart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()
	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 60
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-timeout-restart")

	// Start the service first
	ctx := context.Background()
	err := manager.Up(ctx, "")
	require.NoError(t, err)

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager.Down(cleanupCtx, true)
	}()

	// Use very short timeout for Restart
	restartCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = manager.Restart(restartCtx)

	// Restart should handle timeout (wrapping context error)
	// Note: Restart currently wraps ctx.Err() for both timeout and cancellation
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded, "Should wrap DeadlineExceeded if timeout occurs")
		assert.Contains(t, err.Error(), "cancelled", "Error message should mention context cancellation")
	}
}

// TestContextCancellation_Restart tests that Restart returns cancellation error on manual cancel
func TestContextCancellation_Restart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	skipIfNoDocker(t)

	tmpDir := t.TempDir()
	composeContent := `
services:
  test:
    image: alpine:latest
    command: sleep 60
`
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	require.NoError(t, os.WriteFile(composeFile, []byte(composeContent), 0644))

	manager := NewManager(tmpDir, "compose.yaml", "test-cancel-restart")

	// Start the service first
	ctx := context.Background()
	err := manager.Up(ctx, "")
	require.NoError(t, err)

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		manager.Down(cleanupCtx, true)
	}()

	restartCtx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err = manager.Restart(restartCtx)

	// Should return cancellation error if it occurs
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled, "Should wrap context.Canceled")
		assert.Contains(t, err.Error(), "cancelled", "Error message should mention cancellation")
	}
}
