// Package lock provides file-based locking with PID-based stale detection.
package lock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/jayteealao/otterstack/internal/errors"
)

// Lock represents a file lock for a project.
type Lock struct {
	flock    *flock.Flock
	pidFile  string
	lockPath string
	project  string
}

// Manager manages project locks.
type Manager struct {
	lockDir string
}

// NewManager creates a new lock manager.
func NewManager(dataDir string) (*Manager, error) {
	lockDir := filepath.Join(dataDir, "locks")
	if err := os.MkdirAll(lockDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}
	return &Manager{lockDir: lockDir}, nil
}

// Acquire attempts to acquire a lock for a project.
// It will check for stale locks and remove them before attempting to acquire.
func (m *Manager) Acquire(ctx context.Context, project string) (*Lock, error) {
	lockPath := filepath.Join(m.lockDir, project+".lock")
	pidFile := filepath.Join(m.lockDir, project+".pid")

	// Check for stale lock
	if err := m.cleanStaleLock(pidFile, lockPath); err != nil {
		return nil, err
	}

	fl := flock.New(lockPath)

	// Try to acquire lock with timeout
	locked, err := fl.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		// Check if another process holds it
		if pid, err := readPIDFile(pidFile); err == nil {
			return nil, fmt.Errorf("%w: held by PID %d", errors.ErrProjectLocked, pid)
		}
		return nil, errors.ErrProjectLocked
	}

	// Write our PID file
	if err := writePIDFile(pidFile); err != nil {
		fl.Unlock()
		return nil, fmt.Errorf("failed to write PID file: %w", err)
	}

	return &Lock{
		flock:    fl,
		pidFile:  pidFile,
		lockPath: lockPath,
		project:  project,
	}, nil
}

// TryAcquire attempts to acquire a lock without waiting.
// Returns nil if lock cannot be acquired immediately.
func (m *Manager) TryAcquire(project string) (*Lock, error) {
	lockPath := filepath.Join(m.lockDir, project+".lock")
	pidFile := filepath.Join(m.lockDir, project+".pid")

	// Check for stale lock
	if err := m.cleanStaleLock(pidFile, lockPath); err != nil {
		return nil, err
	}

	fl := flock.New(lockPath)

	locked, err := fl.TryLock()
	if err != nil {
		return nil, fmt.Errorf("failed to try lock: %w", err)
	}
	if !locked {
		return nil, nil // Lock not acquired, but no error
	}

	// Write our PID file
	if err := writePIDFile(pidFile); err != nil {
		fl.Unlock()
		return nil, fmt.Errorf("failed to write PID file: %w", err)
	}

	return &Lock{
		flock:    fl,
		pidFile:  pidFile,
		lockPath: lockPath,
		project:  project,
	}, nil
}

// IsLocked checks if a project is currently locked.
func (m *Manager) IsLocked(project string) (bool, int, error) {
	lockPath := filepath.Join(m.lockDir, project+".lock")
	pidFile := filepath.Join(m.lockDir, project+".pid")

	fl := flock.New(lockPath)

	// Try to acquire briefly to check if locked
	locked, err := fl.TryLock()
	if err != nil {
		return false, 0, fmt.Errorf("failed to check lock: %w", err)
	}

	if locked {
		// We got the lock, release it and report not locked
		fl.Unlock()
		return false, 0, nil
	}

	// Lock is held, try to read PID
	pid, err := readPIDFile(pidFile)
	if err != nil {
		return true, 0, nil // Locked but unknown PID
	}

	return true, pid, nil
}

// cleanStaleLock checks if an existing lock is stale and removes it.
func (m *Manager) cleanStaleLock(pidFile, lockPath string) error {
	pid, err := readPIDFile(pidFile)
	if err != nil {
		// No PID file or can't read it, nothing to clean
		return nil
	}

	// Check if process is still running
	if isProcessRunning(pid) {
		return nil // Process is alive, lock is valid
	}

	// Process is dead, clean up stale lock
	os.Remove(pidFile)
	os.Remove(lockPath)
	return nil
}

// Release releases the lock.
func (l *Lock) Release() error {
	// Remove PID file first
	os.Remove(l.pidFile)

	// Release the lock
	if err := l.flock.Unlock(); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Remove lock file
	os.Remove(l.lockPath)

	return nil
}

// Project returns the project name for this lock.
func (l *Lock) Project() string {
	return l.project
}

// writePIDFile writes the current process PID to the given file.
func writePIDFile(path string) error {
	pid := os.Getpid()
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

// readPIDFile reads a PID from the given file.
func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

// isProcessRunning checks if a process with the given PID is running.
func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds. We need to send signal 0 to check.
	// On Windows, FindProcess only succeeds if the process exists.
	// We use a cross-platform approach by trying to get process info.
	err = proc.Signal(os.Signal(nil))
	if err == nil {
		return true
	}

	// Check if error indicates process doesn't exist
	errStr := err.Error()
	if strings.Contains(errStr, "process already finished") ||
		strings.Contains(errStr, "no such process") ||
		strings.Contains(errStr, "Access is denied") {
		return false
	}

	// If we can't determine, assume it's running (safer)
	return true
}
