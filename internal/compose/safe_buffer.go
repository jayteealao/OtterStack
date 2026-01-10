package compose

import (
	"bytes"
	"sync"
)

// SafeBuffer is a thread-safe buffer for capturing output in tests.
// It wraps bytes.Buffer with a mutex to prevent data races during
// concurrent writes from Docker operations.
type SafeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// NewSafeBuffer creates a new thread-safe buffer.
func NewSafeBuffer() *SafeBuffer {
	return &SafeBuffer{}
}

// Write writes p to the buffer in a thread-safe manner.
func (sb *SafeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

// String returns the contents of the buffer as a string.
func (sb *SafeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

// Reset clears the buffer.
func (sb *SafeBuffer) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.buf.Reset()
}
