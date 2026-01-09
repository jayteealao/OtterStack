package lock

import "context"

// LockOperations defines the interface for lock management.
type LockOperations interface {
	Acquire(ctx context.Context, project string) (*Lock, error)
	TryAcquire(project string) (*Lock, error)
	IsLocked(project string) (bool, int, error)
}

// Ensure Manager implements LockOperations
var _ LockOperations = (*Manager)(nil)
