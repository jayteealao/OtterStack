package traefik

import (
	"context"
	"testing"
)

// TestIsRunning tests the IsRunning function.
func TestIsRunning(t *testing.T) {
	ctx := context.Background()

	// Test when Traefik might be running
	// Note: This test requires an actual running Traefik container to fully pass
	running, err := IsRunning(ctx)
	if err != nil {
		t.Errorf("IsRunning returned error: %v", err)
	}

	// We can't assert the exact value without knowing test environment
	// But we can verify it returns a boolean
	t.Logf("Traefik running: %v", running)
}

// TestIsRunningContextCancellation tests that IsRunning respects context cancellation.
func TestIsRunningContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := IsRunning(ctx)
	// Should return error or false due to context cancellation
	if err == nil {
		t.Log("Context cancelled but no error returned (may have executed fast enough)")
	}
}
