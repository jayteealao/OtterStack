// Package lock provides file-based locking for OtterStack operations.
//
// DEPRECATED: The DeploymentLock implementation in this file is deprecated.
// Use lock.Manager instead, which provides more robust PID-based stale detection
// and cross-platform file locking via flock.
package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DeploymentLock represents a file-based lock for deployment operations.
//
// Deprecated: Use lock.Manager instead, which provides more robust locking
// with PID-based stale detection and cross-platform support via flock.
type DeploymentLock struct {
	path string
	file *os.File
}

// Retry configuration constants
const (
	defaultMaxRetries     = 5
	defaultInitialBackoff = 10 * time.Millisecond
	defaultMaxBackoff     = 1 * time.Second
	defaultBackoffFactor  = 2
)

// AcquireDeploymentLock attempts to acquire a deployment lock for a project with automatic retry.
// Retries on transient TOCTOU race conditions using exponential backoff.
// Returns an error if a lock already exists (deployment in progress).
//
// Deprecated: Use lock.Manager.Acquire() instead, which provides more robust
// PID-based stale detection and cross-platform file locking via flock.
func AcquireDeploymentLock(dataDir, projectName string) (*DeploymentLock, error) {
	return AcquireDeploymentLockWithRetry(dataDir, projectName, defaultMaxRetries)
}

// AcquireDeploymentLockWithRetry attempts lock acquisition with exponential backoff.
// It retries on transient errors like TOCTOU race conditions.
//
// Parameters:
//   - dataDir: The data directory containing the locks subdirectory
//   - projectName: The name of the project to lock
//   - maxAttempts: Maximum number of acquisition attempts (including first try)
//
// Returns the lock on success, or an error after all retries are exhausted.
func AcquireDeploymentLockWithRetry(dataDir, projectName string, maxAttempts int) (*DeploymentLock, error) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	backoff := defaultInitialBackoff

	for attempt := 0; attempt < maxAttempts; attempt++ {
		lock, err := tryAcquireDeploymentLock(dataDir, projectName)
		if err == nil {
			return lock, nil
		}

		lastErr = err

		// Only retry on TOCTOU errors
		if !isRetryableError(err) {
			return nil, err
		}

		// Don't sleep on last attempt
		if attempt < maxAttempts-1 {
			time.Sleep(backoff)
			backoff = backoff * defaultBackoffFactor
			if backoff > defaultMaxBackoff {
				backoff = defaultMaxBackoff
			}
		}
	}

	return nil, fmt.Errorf("failed to acquire lock after %d attempts: %w", maxAttempts, lastErr)
}

// isRetryableError checks if an error represents a transient condition that should trigger a retry.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "lock file was deleted during acquisition") ||
		strings.Contains(errMsg, "concurrent operation detected")
}

// tryAcquireDeploymentLock is the internal implementation of lock acquisition.
// This function contains the core locking logic and is called by the retry wrapper.
func tryAcquireDeploymentLock(dataDir, projectName string) (*DeploymentLock, error) {
	// Create lock directory if needed
	lockDir := filepath.Join(dataDir, "locks")
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	lockPath := filepath.Join(lockDir, projectName+".lock")

	// Try to create exclusive lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		// Check if lock file exists and is stale
		if os.IsExist(err) {
			// Try to read and check if the lock is stale
			if info, statErr := os.Stat(lockPath); statErr == nil {
				// Check age of lock file
				age := time.Since(info.ModTime())
				if age > 30*time.Minute {
					// Lock is stale, attempt to remove it
					if removeErr := os.Remove(lockPath); removeErr == nil {
						// Try to acquire again
						file, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
						if err != nil {
							return nil, fmt.Errorf("deployment in progress (lock file exists)")
						}
					} else {
						return nil, fmt.Errorf("deployment in progress (stale lock cannot be removed)")
					}
				} else {
					return nil, fmt.Errorf("deployment in progress (lock file exists, age: %v)", age.Round(time.Second))
				}
			} else if os.IsNotExist(statErr) {
				// TOCTOU race: file was deleted between OpenFile and Stat
				// This is a transient condition that can be retried
				return nil, fmt.Errorf("lock file was deleted during acquisition (concurrent operation detected)")
			} else {
				// Stat failed for another reason (permissions, I/O error, etc.)
				return nil, fmt.Errorf("failed to check lock file: %w", statErr)
			}
		} else {
			return nil, fmt.Errorf("failed to create lock file: %w", err)
		}
	}

	// Write PID and timestamp to lock file
	fmt.Fprintf(file, "%d\n", os.Getpid())
	fmt.Fprintf(file, "%s\n", time.Now().Format(time.RFC3339))

	return &DeploymentLock{
		path: lockPath,
		file: file,
	}, nil
}

// Release releases the deployment lock and removes the lock file.
func (l *DeploymentLock) Release() {
	if l == nil {
		return
	}
	if l.file != nil {
		l.file.Close()
	}
	if l.path != "" {
		os.Remove(l.path)
	}
}
