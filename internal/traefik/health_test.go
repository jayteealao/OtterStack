package traefik

import (
	"context"
	"testing"
	"time"
)

// TestWaitForHealthyTimeout tests that WaitForHealthy returns error on timeout.
func TestWaitForHealthyTimeout(t *testing.T) {
	ctx := context.Background()

	// Use a very short timeout
	timeout := 100 * time.Millisecond

	// Use a non-existent project name
	err := WaitForHealthy(ctx, "nonexistent-test-project", timeout)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	t.Logf("Got expected error (likely timeout): %v", err)
}

// TestWaitForHealthyContextCancellation tests that WaitForHealthy respects context cancellation.
func TestWaitForHealthyContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// This should return immediately due to context cancellation
	err := WaitForHealthy(ctx, "test-project", DefaultHealthTimeout)
	if err == nil {
		t.Error("Expected error from cancelled context, got nil")
	}
	t.Logf("Got expected error from cancelled context: %v", err)
}

// TestCheckHealth tests the checkHealth function.
func TestCheckHealth(t *testing.T) {
	ctx := context.Background()

	// Test with a non-existent project
	// This should return false or error depending on docker setup
	healthy, err := checkHealth(ctx, "nonexistent-test-project")
	if err != nil {
		t.Logf("checkHealth returned error (expected for non-existent project): %v", err)
	} else {
		t.Logf("checkHealth returned: %v (healthy=%v)", err, healthy)
	}
}

// TestDefaultHealthTimeout verifies the default health timeout is set appropriately.
func TestDefaultHealthTimeout(t *testing.T) {
	expected := 5 * time.Minute
	if DefaultHealthTimeout != expected {
		t.Errorf("Expected DefaultHealthTimeout to be %v, got %v", expected, DefaultHealthTimeout)
	}
}
