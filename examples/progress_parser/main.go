// Package main demonstrates parsing Docker Compose output to track deployment progress.
//
// This example shows how to intercept Docker output and extract meaningful progress
// information for display in custom UIs, dashboards, or progress bars.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jayteealao/otterstack/internal/compose"
)

// ProgressEvent represents a parsed progress event from Docker output.
type ProgressEvent struct {
	Timestamp  time.Time
	EventType  string // "pulling", "starting", "running", "healthy", "error"
	Service    string
	Message    string
	IsComplete bool
}

// ProgressParser parses Docker Compose output and extracts progress events.
type ProgressParser struct {
	out       io.Writer             // Original output destination
	onEvent   func(event ProgressEvent) // Callback for each event
	patterns  map[string]*regexp.Regexp
}

// NewProgressParser creates a parser that extracts progress information from Docker output.
func NewProgressParser(out io.Writer, onEvent func(ProgressEvent)) *ProgressParser {
	return &ProgressParser{
		out:     out,
		onEvent: onEvent,
		patterns: map[string]*regexp.Regexp{
			// Match: [+] Pulling myapp-web
			"pulling": regexp.MustCompile(`\[\+\] Pulling (.+)`),

			// Match: ✔ Container myapp-web-1 Started
			// Match: ⠿ Container myapp-web-1 Starting
			"container": regexp.MustCompile(`[✔⠿]\s+Container\s+(\S+)\s+(\w+)`),

			// Match: [+] Running 3/5
			"running": regexp.MustCompile(`\[\+\] Running (\d+)/(\d+)`),

			// Match: Creating network "myapp_default"
			"network": regexp.MustCompile(`Creating network "(.+)"`),

			// Match: Pulling from library/nginx
			"image": regexp.MustCompile(`Pulling from (.+)`),

			// Match: Error response from daemon
			"error": regexp.MustCompile(`(?i)error|failed`),
		},
	}
}

// Write implements io.Writer by parsing each line and extracting progress events.
func (p *ProgressParser) Write(data []byte) (n int, err error) {
	// Always write to the underlying output first
	n, err = p.out.Write(data)
	if err != nil {
		return n, err
	}

	// Parse lines for progress events
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if event := p.parseLine(line); event != nil {
			if p.onEvent != nil {
				p.onEvent(*event)
			}
		}
	}

	return n, nil
}

// parseLine extracts a ProgressEvent from a single line of Docker output.
func (p *ProgressParser) parseLine(line string) *ProgressEvent {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	now := time.Now()

	// Check for errors first
	if p.patterns["error"].MatchString(line) {
		return &ProgressEvent{
			Timestamp:  now,
			EventType:  "error",
			Message:    line,
			IsComplete: false,
		}
	}

	// Check for pulling images
	if matches := p.patterns["pulling"].FindStringSubmatch(line); matches != nil {
		return &ProgressEvent{
			Timestamp:  now,
			EventType:  "pulling",
			Service:    matches[1],
			Message:    fmt.Sprintf("Pulling %s", matches[1]),
			IsComplete: false,
		}
	}

	// Check for container status
	if matches := p.patterns["container"].FindStringSubmatch(line); matches != nil {
		containerName := matches[1]
		status := strings.ToLower(matches[2])

		eventType := "starting"
		isComplete := false

		switch status {
		case "started":
			eventType = "running"
			isComplete = true
		case "healthy":
			eventType = "healthy"
			isComplete = true
		case "created":
			eventType = "created"
			isComplete = true
		}

		return &ProgressEvent{
			Timestamp:  now,
			EventType:  eventType,
			Service:    containerName,
			Message:    fmt.Sprintf("Container %s %s", containerName, status),
			IsComplete: isComplete,
		}
	}

	// Check for running progress (X/Y containers)
	if matches := p.patterns["running"].FindStringSubmatch(line); matches != nil {
		return &ProgressEvent{
			Timestamp:  now,
			EventType:  "progress",
			Message:    fmt.Sprintf("Running %s/%s containers", matches[1], matches[2]),
			IsComplete: matches[1] == matches[2],
		}
	}

	// Check for network creation
	if matches := p.patterns["network"].FindStringSubmatch(line); matches != nil {
		return &ProgressEvent{
			Timestamp:  now,
			EventType:  "network",
			Service:    matches[1],
			Message:    fmt.Sprintf("Created network %s", matches[1]),
			IsComplete: true,
		}
	}

	// Check for image pull progress
	if matches := p.patterns["image"].FindStringSubmatch(line); matches != nil {
		return &ProgressEvent{
			Timestamp:  now,
			EventType:  "image",
			Service:    matches[1],
			Message:    fmt.Sprintf("Pulling image %s", matches[1]),
			IsComplete: false,
		}
	}

	return nil
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

	// Track deployment progress
	var events []ProgressEvent
	totalEvents := 0
	completedEvents := 0

	// Event handler that prints progress
	onEvent := func(event ProgressEvent) {
		totalEvents++
		if event.IsComplete {
			completedEvents++
		}

		// Store event
		events = append(events, event)

		// Print formatted event
		status := "🔄"
		if event.IsComplete {
			status = "✅"
		}
		if event.EventType == "error" {
			status = "❌"
		}

		fmt.Printf("%s [%s] %s\n",
			status,
			event.EventType,
			event.Message,
		)
	}

	// Create compose manager
	projectName := fmt.Sprintf("progress-example-%d", time.Now().Unix())
	manager := compose.NewManager(workingDir, composeFile, projectName)

	// Configure progress parser
	stdoutParser := NewProgressParser(os.Stdout, onEvent)
	stderrParser := NewProgressParser(os.Stderr, onEvent)

	manager.SetOutputStreams(stdoutParser, stderrParser)

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("Starting Docker Compose with progress tracking...")
	log.Println("Working directory:", workingDir)
	log.Println("Compose file:", composeFile)
	log.Println("Project name:", projectName)
	log.Println("")

	// Validate compose file
	log.Println("Validating compose file...")
	if err := manager.Validate(ctx); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}
	log.Println("")

	// Pull images
	log.Println("Pulling images (progress will be tracked)...")
	log.Println("---")
	if err := manager.Pull(ctx); err != nil {
		log.Printf("Warning: Pull failed: %v", err)
	}
	log.Println("---")
	log.Println("")

	// Reset counters for the main deployment
	totalEvents = 0
	completedEvents = 0

	// Start services
	log.Println("Starting services (progress will be tracked)...")
	log.Println("---")
	if err := manager.Up(ctx, ""); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}
	log.Println("---")
	log.Println("")

	// Print summary
	log.Println("=== Deployment Summary ===")
	log.Printf("Total events: %d", totalEvents)
	log.Printf("Completed: %d", completedEvents)
	log.Println("")

	// Group events by type
	eventsByType := make(map[string]int)
	for _, event := range events {
		eventsByType[event.EventType]++
	}

	log.Println("Events by type:")
	for eventType, count := range eventsByType {
		log.Printf("  %s: %d", eventType, count)
	}
	log.Println("")

	log.Println("Services started successfully!")
	log.Printf("To stop services: docker compose -p %s down", projectName)
}
