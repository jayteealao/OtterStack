// Package lock provides file-based locking for OtterStack operations.
package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DeploymentLock represents a file-based lock for deployment operations.
type DeploymentLock struct {
	path string
	file *os.File
}

// AcquireDeploymentLock attempts to acquire a deployment lock for a project.
// Returns an error if a lock already exists (deployment in progress).
func AcquireDeploymentLock(dataDir, projectName string) (*DeploymentLock, error) {
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
			} else {
				return nil, fmt.Errorf("deployment in progress (lock file exists)")
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
	if l.file != nil {
		l.file.Close()
	}
	if l.path != "" {
		os.Remove(l.path)
	}
}
