package lock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAcquireDeploymentLock tests acquiring a deployment lock.
func TestAcquireDeploymentLock(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "otterstack-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "test-project"

	// Acquire lock
	lock, err := AcquireDeploymentLock(tmpDir, projectName)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Verify lock file was created
	lockPath := filepath.Join(tmpDir, "locks", projectName+".lock")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}

	// Read lock file content
	content, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	t.Logf("Lock file content: %s", content)

	// Release lock
	lock.Release()

	// Verify lock file was removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("Lock file was not removed after release")
	}
}

// TestAcquireDeploymentLockConcurrent tests that concurrent lock acquisition fails.
func TestAcquireDeploymentLockConcurrent(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "otterstack-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "test-project-concurrent"

	// Acquire first lock
	lock1, err := AcquireDeploymentLock(tmpDir, projectName)
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}
	defer lock1.Release()

	// Try to acquire second lock (should fail)
	_, err = AcquireDeploymentLock(tmpDir, projectName)
	if err == nil {
		t.Error("Expected error when acquiring lock that is already held")
	}
	t.Logf("Got expected error for concurrent lock: %v", err)
}

// TestAcquireDeploymentLockStale tests stale lock detection and removal.
func TestAcquireDeploymentLockStale(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "otterstack-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "test-project-stale"
	lockDir := filepath.Join(tmpDir, "locks")

	// Create lock directory
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		t.Fatalf("Failed to create lock dir: %v", err)
	}

	// Create a stale lock file (older than 30 minutes)
	lockPath := filepath.Join(lockDir, projectName+".lock")
	oldTime := time.Now().Add(-31 * time.Minute)
	if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
		// Create the file first if it doesn't exist
		f, err := os.Create(lockPath)
		if err != nil {
			t.Fatalf("Failed to create stale lock file: %v", err)
		}
		f.Close()
		if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
			t.Fatalf("Failed to set lock file time: %v", err)
		}
	}

	// Try to acquire lock - should succeed and remove stale lock
	lock, err := AcquireDeploymentLock(tmpDir, projectName)
	if err != nil {
		t.Fatalf("Failed to acquire lock (should have removed stale lock): %v", err)
	}
	defer lock.Release()

	t.Log("Successfully acquired lock after removing stale lock")
}

// TestDeploymentLockRelease tests releasing a lock.
func TestDeploymentLockRelease(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "otterstack-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "test-project-release"

	// Acquire lock
	lock, err := AcquireDeploymentLock(tmpDir, projectName)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Verify lock file exists
	lockPath := filepath.Join(tmpDir, "locks", projectName+".lock")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("Lock file was not created")
	}

	// Release lock
	lock.Release()

	// Verify lock file was removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("Lock file was not removed after release")
	}

	// Should be able to acquire lock again
	lock2, err := AcquireDeploymentLock(tmpDir, projectName)
	if err != nil {
		t.Fatalf("Failed to acquire lock after release: %v", err)
	}
	defer lock2.Release()

	t.Log("Successfully re-acquired lock after release")
}

// TestReleaseNilLock tests that Release on a nil lock doesn't panic.
func TestReleaseNilLock(t *testing.T) {
	var lock *DeploymentLock
	// Should not panic
	lock.Release()
}

// TestAcquireDeploymentLockRaceCondition simulates TOCTOU race condition
func TestAcquireDeploymentLockRaceCondition(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "otterstack-test-race-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "test-race"
	lockDir := filepath.Join(tmpDir, "locks")
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		t.Fatalf("Failed to create lock dir: %v", err)
	}
	lockPath := filepath.Join(lockDir, projectName+".lock")

	// Create lock that will be deleted during acquisition
	f, err := os.Create(lockPath)
	if err != nil {
		t.Fatalf("Failed to create lock file: %v", err)
	}
	f.Close()

	// Delete concurrently
	go func() {
		time.Sleep(5 * time.Millisecond)
		os.Remove(lockPath)
	}()

	// Should succeed after retry
	lock, err := AcquireDeploymentLock(tmpDir, projectName)
	if lock != nil {
		defer lock.Release()
	}

	// Should either succeed or fail with accurate message
	if err != nil && strings.Contains(err.Error(), "lock file exists") {
		// Verify file actually exists if error claims it does
		if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
			t.Errorf("Error claims lock exists but file doesn't: %v", err)
		}
	}

	t.Logf("Lock acquisition result: %v", err)
}

// TestAcquireDeploymentLockErrorMessages verifies accurate error messages
func TestAcquireDeploymentLockErrorMessages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "otterstack-test-errors-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "test-errors"

	// Acquire first lock
	lock1, err := AcquireDeploymentLock(tmpDir, projectName)
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}
	defer lock1.Release()

	// Try second lock
	_, err = AcquireDeploymentLock(tmpDir, projectName)
	if err == nil {
		t.Fatal("Expected error when acquiring lock that is already held")
	}

	// Error should mention "deployment in progress"
	if !strings.Contains(err.Error(), "deployment in progress") {
		t.Errorf("Error should mention 'deployment in progress', got: %v", err)
	}

	// If error mentions lock exists, verify it actually exists
	if strings.Contains(err.Error(), "lock file exists") {
		lockPath := filepath.Join(tmpDir, "locks", projectName+".lock")
		if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
			t.Errorf("Error claims lock exists but file doesn't exist: %v", err)
		}
	}

	t.Logf("Got expected error: %v", err)
}

// TestAcquireDeploymentLockRetryExhaustion tests retry limit behavior
func TestAcquireDeploymentLockRetryExhaustion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "otterstack-test-retry-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectName := "test-retry"

	// Create a lock that won't be released
	lock1, err := AcquireDeploymentLock(tmpDir, projectName)
	if err != nil {
		t.Fatalf("Failed to acquire initial lock: %v", err)
	}
	defer lock1.Release()

	// Try to acquire with limited retries
	start := time.Now()
	lock2, err := AcquireDeploymentLockWithRetry(tmpDir, projectName, 3)
	duration := time.Since(start)

	if err == nil {
		if lock2 != nil {
			lock2.Release()
		}
		t.Fatal("Expected error when lock is held")
	}

	// Should complete reasonably quickly (< 500ms with 3 retries and short backoffs)
	if duration > 500*time.Millisecond {
		t.Errorf("Retry exhaustion took too long: %v", duration)
	}

	t.Logf("Retry exhaustion completed in %v with error: %v", duration, err)
}
