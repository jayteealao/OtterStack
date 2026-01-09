package lock

import (
	"os"
	"path/filepath"
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
