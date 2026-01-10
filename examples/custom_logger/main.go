// Package main demonstrates a custom output handler that adds timestamps to Docker output.
//
// This example shows how to wrap Docker Compose output streams with custom formatting,
// making it easier to track when events occur during deployment.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
)

// TimestampedWriter wraps an io.Writer to add timestamps to each line.
// It buffers output line by line and prefixes each line with an ISO 8601 timestamp.
type TimestampedWriter struct {
	prefix string    // Prefix to distinguish stdout from stderr
	out    io.Writer // Underlying writer (os.Stdout, file, etc)
}

// NewTimestampedWriter creates a writer that prefixes each line with a timestamp.
func NewTimestampedWriter(prefix string, out io.Writer) *TimestampedWriter {
	return &TimestampedWriter{
		prefix: prefix,
		out:    out,
	}
}

// Write implements io.Writer by adding timestamps to each line.
func (w *TimestampedWriter) Write(p []byte) (n int, err error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Split input into lines
	scanner := bufio.NewScanner(bufio.NewReader(io.NopCloser(io.LimitReader(
		&lineReader{data: p},
		int64(len(p)),
	))))

	scanner.Split(bufio.ScanLines)
	written := 0

	for scanner.Scan() {
		line := scanner.Text()
		formatted := fmt.Sprintf("[%s] %s: %s\n", timestamp, w.prefix, line)
		_, writeErr := w.out.Write([]byte(formatted))
		if writeErr != nil {
			return written, writeErr
		}
		written += len(line) + 1 // Count original bytes including newline

		// Update timestamp for next line
		timestamp = time.Now().Format("2006-01-02 15:04:05.000")
	}

	if err := scanner.Err(); err != nil {
		return written, err
	}

	// Return the number of bytes from the original input
	return len(p), nil
}

// lineReader is a helper to convert []byte to an io.Reader
type lineReader struct {
	data []byte
	pos  int
}

func (r *lineReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func main() {
	// This example requires a local Docker Compose project
	// Usage: go run main.go <working-dir> <compose-file>

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: go run main.go <working-dir> <compose-file>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  go run main.go /path/to/project docker-compose.yml")
		os.Exit(1)
	}

	workingDir := os.Args[1]
	composeFile := os.Args[2]

	// Create compose manager
	projectName := fmt.Sprintf("example-%d", time.Now().Unix())
	manager := compose.NewManager(workingDir, composeFile, projectName)

	// Configure custom output handlers with timestamps
	stdoutHandler := NewTimestampedWriter("DOCKER", os.Stdout)
	stderrHandler := NewTimestampedWriter("ERROR", os.Stderr)

	manager.SetOutputStreams(stdoutHandler, stderrHandler)

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("Starting Docker Compose with timestamped output...")
	log.Println("Working directory:", workingDir)
	log.Println("Compose file:", composeFile)
	log.Println("")

	// Validate compose file
	log.Println("Validating compose file...")
	if err := manager.Validate(ctx); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	log.Println("Validation successful!")
	log.Println("")

	// Pull images
	log.Println("Pulling images...")
	if err := manager.Pull(ctx); err != nil {
		log.Printf("Warning: Pull failed: %v", err)
	}
	log.Println("")

	// Start services
	log.Println("Starting services (output will be timestamped)...")
	log.Println("---")
	if err := manager.Up(ctx, ""); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}
	log.Println("---")
	log.Println("")

	log.Println("Services started successfully!")
	log.Printf("Project name: %s", projectName)
	log.Println("")
	log.Printf("To stop services: docker compose -p %s down", projectName)
}
