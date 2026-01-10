// Package main demonstrates logging Docker output to both console and file simultaneously.
//
// This example uses io.MultiWriter to send Docker output to multiple destinations,
// making it easy to keep deployment logs for auditing or debugging.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
)

// LogManager handles logging Docker output to multiple destinations.
type LogManager struct {
	logDir     string
	logFile    *os.File
	stdoutFile *os.File
	stderrFile *os.File
	stdout     io.Writer
	stderr     io.Writer
}

// NewLogManager creates a log manager that writes to both console and files.
func NewLogManager(logDir string) (*LogManager, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create timestamped log files
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	combinedPath := filepath.Join(logDir, fmt.Sprintf("combined_%s.log", timestamp))
	stdoutPath := filepath.Join(logDir, fmt.Sprintf("stdout_%s.log", timestamp))
	stderrPath := filepath.Join(logDir, fmt.Sprintf("stderr_%s.log", timestamp))

	// Open combined log file (for both stdout and stderr)
	logFile, err := os.OpenFile(combinedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create combined log file: %w", err)
	}

	// Open stdout log file
	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to create stdout log file: %w", err)
	}

	// Open stderr log file
	stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile.Close()
		stdoutFile.Close()
		return nil, fmt.Errorf("failed to create stderr log file: %w", err)
	}

	// Create multi-writers
	// stdout: console + stdout.log + combined.log
	stdout := io.MultiWriter(os.Stdout, stdoutFile, logFile)

	// stderr: console + stderr.log + combined.log
	stderr := io.MultiWriter(os.Stderr, stderrFile, logFile)

	return &LogManager{
		logDir:     logDir,
		logFile:    logFile,
		stdoutFile: stdoutFile,
		stderrFile: stderrFile,
		stdout:     stdout,
		stderr:     stderr,
	}, nil
}

// Stdout returns the stdout multi-writer.
func (m *LogManager) Stdout() io.Writer {
	return m.stdout
}

// Stderr returns the stderr multi-writer.
func (m *LogManager) Stderr() io.Writer {
	return m.stderr
}

// Close closes all log files.
func (m *LogManager) Close() error {
	var errs []error

	if err := m.logFile.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close combined log: %w", err))
	}

	if err := m.stdoutFile.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close stdout log: %w", err))
	}

	if err := m.stderrFile.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close stderr log: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing log files: %v", errs)
	}

	return nil
}

// WriteHeader writes a header to all log files.
func (m *LogManager) WriteHeader(header string) error {
	headerLine := fmt.Sprintf("=== %s ===\n", header)
	timestamp := fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
	separator := "===========================================\n\n"

	content := headerLine + timestamp + separator

	// Write to all files (not console)
	writers := []io.Writer{m.logFile, m.stdoutFile, m.stderrFile}
	for _, w := range writers {
		if _, err := w.Write([]byte(content)); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	return nil
}

func main() {
	// This example requires a local Docker Compose project
	// Usage: go run main.go <working-dir> <compose-file> [log-dir]

	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: go run main.go <working-dir> <compose-file> [log-dir]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, "  go run main.go /path/to/project docker-compose.yml")
		fmt.Fprintln(os.Stderr, "  go run main.go /path/to/project docker-compose.yml /var/log/deployments")
		os.Exit(1)
	}

	workingDir := os.Args[1]
	composeFile := os.Args[2]

	// Default log directory
	logDir := "./logs"
	if len(os.Args) >= 4 {
		logDir = os.Args[3]
	}

	// Create log manager
	logManager, err := NewLogManager(logDir)
	if err != nil {
		log.Fatalf("Failed to create log manager: %v", err)
	}
	defer logManager.Close()

	// Write deployment header to log files
	header := fmt.Sprintf("Docker Compose Deployment - %s", filepath.Base(workingDir))
	if err := logManager.WriteHeader(header); err != nil {
		log.Fatalf("Failed to write log header: %v", err)
	}

	// Create compose manager
	projectName := fmt.Sprintf("file-logger-example-%d", time.Now().Unix())
	manager := compose.NewManager(workingDir, composeFile, projectName)

	// Configure output to log to both console and files
	manager.SetOutputStreams(logManager.Stdout(), logManager.Stderr())

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("Starting Docker Compose with file logging...")
	log.Println("Working directory:", workingDir)
	log.Println("Compose file:", composeFile)
	log.Println("Log directory:", logDir)
	log.Println("Project name:", projectName)
	log.Println("")
	log.Println("Output will be displayed on console AND saved to log files:")
	log.Printf("  - %s/combined_*.log (all output)\n", logDir)
	log.Printf("  - %s/stdout_*.log (standard output)\n", logDir)
	log.Printf("  - %s/stderr_*.log (error output)\n", logDir)
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
	log.Println("---")
	if err := manager.Pull(ctx); err != nil {
		log.Printf("Warning: Pull failed: %v", err)
	}
	log.Println("---")
	log.Println("")

	// Start services
	log.Println("Starting services...")
	log.Println("---")
	if err := manager.Up(ctx, ""); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}
	log.Println("---")
	log.Println("")

	log.Println("Services started successfully!")
	log.Println("")
	log.Println("Log files created:")
	files, _ := filepath.Glob(filepath.Join(logDir, "*.log"))
	for _, file := range files {
		info, err := os.Stat(file)
		if err == nil {
			log.Printf("  %s (%d bytes)", file, info.Size())
		}
	}
	log.Println("")
	log.Printf("To stop services: docker compose -p %s down", projectName)
}
