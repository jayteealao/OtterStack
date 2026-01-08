package lock

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestManager(t *testing.T) (*Manager, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "otterstack-lock-test-*")
	require.NoError(t, err)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return manager, cleanup
}

func TestManager_AcquireAndRelease(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("acquire and release lock", func(t *testing.T) {
		lock, err := manager.Acquire(ctx, "test-project")
		require.NoError(t, err)
		require.NotNil(t, lock)
		assert.Equal(t, "test-project", lock.Project())

		// Lock should exist
		locked, _, err := manager.IsLocked("test-project")
		require.NoError(t, err)
		assert.True(t, locked)

		// Release the lock
		err = lock.Release()
		require.NoError(t, err)

		// Lock should not exist
		locked, _, err = manager.IsLocked("test-project")
		require.NoError(t, err)
		assert.False(t, locked)
	})

	t.Run("acquire lock on different projects", func(t *testing.T) {
		lock1, err := manager.Acquire(ctx, "project-a")
		require.NoError(t, err)
		defer lock1.Release()

		lock2, err := manager.Acquire(ctx, "project-b")
		require.NoError(t, err)
		defer lock2.Release()

		// Both should be locked
		locked1, _, _ := manager.IsLocked("project-a")
		locked2, _, _ := manager.IsLocked("project-b")
		assert.True(t, locked1)
		assert.True(t, locked2)
	})
}

func TestManager_TryAcquire(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	t.Run("try acquire succeeds when not locked", func(t *testing.T) {
		lock, err := manager.TryAcquire("test-project")
		require.NoError(t, err)
		require.NotNil(t, lock)
		defer lock.Release()
	})

	t.Run("try acquire returns nil when locked", func(t *testing.T) {
		ctx := context.Background()
		lock1, err := manager.Acquire(ctx, "locked-project")
		require.NoError(t, err)
		defer lock1.Release()

		// TryAcquire should return nil (not error) when can't acquire
		lock2, err := manager.TryAcquire("locked-project")
		require.NoError(t, err)
		assert.Nil(t, lock2)
	})
}

func TestManager_IsLocked(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("not locked initially", func(t *testing.T) {
		locked, pid, err := manager.IsLocked("nonexistent")
		require.NoError(t, err)
		assert.False(t, locked)
		assert.Equal(t, 0, pid)
	})

	t.Run("locked returns current pid", func(t *testing.T) {
		lock, err := manager.Acquire(ctx, "pid-test")
		require.NoError(t, err)
		defer lock.Release()

		locked, pid, err := manager.IsLocked("pid-test")
		require.NoError(t, err)
		assert.True(t, locked)
		assert.Equal(t, os.Getpid(), pid)
	})
}

func TestManager_StaleLockDetection(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	project := "stale-test"

	// Create fake stale lock files with a non-existent PID
	lockPath := filepath.Join(manager.lockDir, project+".lock")
	pidFile := filepath.Join(manager.lockDir, project+".pid")

	// Use PID 2 which is typically not a user process (kthreadd on Linux, or non-existent)
	// Use a very high PID that's unlikely to exist
	require.NoError(t, os.WriteFile(lockPath, []byte{}, 0644))
	require.NoError(t, os.WriteFile(pidFile, []byte("999999999"), 0644))

	// Should be able to acquire because the PID doesn't exist
	lock, err := manager.Acquire(ctx, project)
	require.NoError(t, err)
	require.NotNil(t, lock)
	lock.Release()
}

func TestManager_ContextCancellation(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()

	project := "ctx-test"

	// First, acquire a lock
	lock1, err := manager.TryAcquire(project)
	require.NoError(t, err)
	require.NotNil(t, lock1)
	defer lock1.Release()

	// Try to acquire with a cancelled context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	lock2, err := manager.Acquire(ctx, project)
	assert.Error(t, err) // Should fail due to context timeout
	assert.Nil(t, lock2)
}

func TestReadWritePIDFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pid-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pidFile := filepath.Join(tmpDir, "test.pid")

	t.Run("write and read pid", func(t *testing.T) {
		err := writePIDFile(pidFile)
		require.NoError(t, err)

		pid, err := readPIDFile(pidFile)
		require.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := readPIDFile(filepath.Join(tmpDir, "nonexistent.pid"))
		assert.Error(t, err)
	})
}
