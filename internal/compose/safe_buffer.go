package compose

import (
	"bytes"
	"sync"
)

// SafeBuffer is a thread-safe buffer for capturing output in tests.
// It wraps bytes.Buffer with a mutex to prevent data races during
// concurrent writes from Docker operations.
//
// Thread-safe: All methods can be called concurrently from multiple goroutines.
//
// Example:
//
//	buf := NewSafeBuffer()
//	manager.SetOutputStreams(buf, buf)
//	manager.Up(ctx, "")
//	output := buf.String() // Safe to call while Docker command is running
type SafeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// NewSafeBuffer creates a new thread-safe buffer.
// The buffer is initially empty and ready to accept writes.
func NewSafeBuffer() *SafeBuffer {
	return &SafeBuffer{}
}

// Write writes p to the buffer in a thread-safe manner.
// Implements io.Writer interface.
//
// Parameters:
//   - p: Byte slice to write to the buffer
//
// Returns:
//   - n: Number of bytes written (always len(p))
//   - err: Always nil (bytes.Buffer.Write never returns an error)
//
// Thread-safe: Can be called concurrently with other SafeBuffer methods.
func (sb *SafeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

// String returns the contents of the buffer as a string.
// Safe to call while concurrent writes are happening.
//
// Returns:
//   - The accumulated buffer contents as a string
//
// Thread-safe: Can be called concurrently with other SafeBuffer methods.
func (sb *SafeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

// Reset clears the buffer, discarding all accumulated data.
// After Reset, the buffer is empty and ready to accept new writes.
//
// Thread-safe: Can be called concurrently with other SafeBuffer methods.
func (sb *SafeBuffer) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.buf.Reset()
}
