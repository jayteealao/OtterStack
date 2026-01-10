# OtterStack Examples

This directory contains practical examples demonstrating how to use custom output handlers with OtterStack's compose manager.

## Overview

OtterStack's compose manager provides a `SetOutputStreams(stdout, stderr io.Writer)` method that allows you to redirect or wrap Docker Compose output. This enables:

- Custom logging formats
- Progress tracking and parsing
- Multi-destination output
- Real-time event processing
- Integration with monitoring systems

## Examples

### 1. Custom Logger - Timestamped Output

**Directory**: `custom_logger/`

Add ISO 8601 timestamps to every line of Docker output.

```go
type TimestampedWriter struct {
    prefix string
    out    io.Writer
}

manager.SetOutputStreams(
    NewTimestampedWriter("DOCKER", os.Stdout),
    NewTimestampedWriter("ERROR", os.Stderr),
)
```

**Output**:
```
[2025-01-10 14:32:15.234] DOCKER: Pulling myapp-web
[2025-01-10 14:32:15.456] DOCKER: [+] Running 2/2
[2025-01-10 14:32:15.789] DOCKER:  ✔ Container myapp-db-1 Started
```

**Use cases**:
- Production deployments
- Debugging timing issues
- Correlating with application logs
- Audit trails

### 2. Progress Parser - Event Extraction

**Directory**: `progress_parser/`

Parse Docker output into structured progress events for custom UIs.

```go
type ProgressEvent struct {
    Timestamp  time.Time
    EventType  string  // "pulling", "starting", "running", "healthy"
    Service    string
    Message    string
    IsComplete bool
}

manager.SetOutputStreams(
    NewProgressParser(os.Stdout, onEvent),
    NewProgressParser(os.Stderr, onEvent),
)
```

**Output**:
```
🔄 [pulling] Pulling myapp-web
✅ [network] Created network myapp_default
✅ [running] Container myapp-db-1 started
✅ [progress] Running 2/2 containers
```

**Use cases**:
- Web deployment dashboards
- Progress bars
- Slack/Discord notifications
- Metrics collection
- Integration testing

### 3. File Logger - Multi-Destination Output

**Directory**: `file_logger/`

Log Docker output to console AND files simultaneously using `io.MultiWriter`.

```go
stdout := io.MultiWriter(os.Stdout, stdoutFile, combinedFile)
stderr := io.MultiWriter(os.Stderr, stderrFile, combinedFile)

manager.SetOutputStreams(stdout, stderr)
```

**Files created**:
```
logs/
├── combined_2025-01-10_14-32-15.log    # All output
├── stdout_2025-01-10_14-32-15.log      # Standard output only
└── stderr_2025-01-10_14-32-15.log      # Error output only
```

**Use cases**:
- Permanent deployment records
- Compliance/auditing
- CI/CD artifacts
- Post-mortem debugging
- Log analysis

## Running the Examples

All examples follow the same pattern:

```bash
# Navigate to example directory
cd examples/custom_logger

# Run with your Docker Compose project
go run main.go <working-dir> <compose-file>

# Example
go run main.go /srv/myapp docker-compose.yml
```

### Requirements

- Go 1.21 or later
- Docker with Compose plugin
- A Docker Compose project to test with

## Architecture

### The SetOutputStreams Pattern

OtterStack's compose manager uses Go's `io.Writer` interface for output:

```go
type Manager struct {
    workingDir  string
    composeFile string
    projectName string
    stdout      io.Writer  // Default: os.Stdout
    stderr      io.Writer  // Default: os.Stderr
}

func (m *Manager) SetOutputStreams(stdout, stderr io.Writer) {
    m.stdout = stdout
    m.stderr = stderr
}
```

All Docker commands stream output to these writers:

```go
cmd := exec.Command("docker", "compose", "up")
cmd.Stdout = m.stdout  // Your custom writer
cmd.Stderr = m.stderr  // Your custom writer
cmd.Run()
```

### Why io.Writer?

The `io.Writer` interface is simple and composable:

```go
type Writer interface {
    Write(p []byte) (n int, err error)
}
```

This enables:

1. **Chaining**: Wrap writers to add functionality
2. **Multiplexing**: Use `io.MultiWriter` for multiple outputs
3. **Filtering**: Transform or redact output
4. **Buffering**: Control when output is flushed
5. **Testing**: Easy to mock with `bytes.Buffer`

## Common Patterns

### Pattern 1: Wrap and Forward

```go
type CustomWriter struct {
    out io.Writer
}

func (w *CustomWriter) Write(p []byte) (n int, err error) {
    // Transform or inspect p
    modified := transform(p)

    // Forward to underlying writer
    return w.out.Write(modified)
}
```

### Pattern 2: Multi-Destination

```go
// Write to multiple places
multi := io.MultiWriter(
    os.Stdout,           // Console
    logFile,             // File
    networkLogger,       // Remote
)

manager.SetOutputStreams(multi, multi)
```

### Pattern 3: Parse and Callback

```go
type Parser struct {
    out     io.Writer
    onEvent func(Event)
}

func (p *Parser) Write(data []byte) (n int, err error) {
    // Parse data
    if event := p.parse(data); event != nil {
        p.onEvent(event)
    }

    // Still show original output
    return p.out.Write(data)
}
```

### Pattern 4: Filter and Redact

```go
type RedactWriter struct {
    out     io.Writer
    pattern *regexp.Regexp
}

func (w *RedactWriter) Write(p []byte) (n int, err error) {
    // Remove sensitive data
    redacted := w.pattern.ReplaceAll(p, []byte("[REDACTED]"))
    return w.out.Write(redacted)
}
```

## Building Your Own Handler

Follow these guidelines when creating custom output handlers:

### 1. Implement io.Writer Correctly

```go
func (w *CustomWriter) Write(p []byte) (n int, err error) {
    // Return the number of bytes from the INPUT (p)
    // Even if you write different bytes to the output

    modified := transform(p)
    _, err = w.out.Write(modified)

    return len(p), err  // Return len(p), not len(modified)
}
```

### 2. Handle Partial Writes

Docker output may arrive in chunks. Handle incomplete lines:

```go
type LineBufferedWriter struct {
    out    io.Writer
    buffer []byte
}

func (w *LineBufferedWriter) Write(p []byte) (n int, err error) {
    w.buffer = append(w.buffer, p...)

    // Process complete lines
    for {
        idx := bytes.IndexByte(w.buffer, '\n')
        if idx == -1 {
            break
        }

        line := w.buffer[:idx+1]
        w.processLine(line)
        w.buffer = w.buffer[idx+1:]
    }

    return len(p), nil
}
```

### 3. Be Thread-Safe

If your handler maintains state, protect it:

```go
type StatefulWriter struct {
    mu     sync.Mutex
    state  map[string]int
    out    io.Writer
}

func (w *StatefulWriter) Write(p []byte) (n int, err error) {
    w.mu.Lock()
    defer w.mu.Unlock()

    // Update state
    w.state["lines"]++

    return w.out.Write(p)
}
```

### 4. Clean Up Resources

If your handler opens files or connections:

```go
type ResourceWriter struct {
    file *os.File
}

func (w *ResourceWriter) Write(p []byte) (n int, err error) {
    return w.file.Write(p)
}

func (w *ResourceWriter) Close() error {
    return w.file.Close()
}

// Usage
writer := NewResourceWriter()
defer writer.Close()
```

## Testing Your Handler

Test your handler with realistic Docker output:

```go
func TestCustomWriter(t *testing.T) {
    var buf bytes.Buffer
    writer := NewCustomWriter(&buf)

    // Simulate Docker output
    input := []byte("[+] Running 2/2\n ✔ Container app Started\n")
    n, err := writer.Write(input)

    assert.NoError(t, err)
    assert.Equal(t, len(input), n)
    assert.Contains(t, buf.String(), "expected output")
}
```

## Advanced Use Cases

### Combine Multiple Handlers

```go
// Timestamp + Parse + Log to file
timestamped := NewTimestampedWriter("DOCKER", os.Stdout)
parser := NewProgressParser(timestamped, onEvent)
multi := io.MultiWriter(parser, logFile)

manager.SetOutputStreams(multi, multi)
```

### Conditional Output

```go
type ConditionalWriter struct {
    out       io.Writer
    condition func([]byte) bool
}

func (w *ConditionalWriter) Write(p []byte) (n int, err error) {
    if w.condition(p) {
        return w.out.Write(p)
    }
    return len(p), nil  // Discard
}

// Only show errors
onlyErrors := &ConditionalWriter{
    out: os.Stderr,
    condition: func(p []byte) bool {
        return bytes.Contains(p, []byte("error")) ||
               bytes.Contains(p, []byte("failed"))
    },
}
```

### Rate-Limited Output

```go
import "golang.org/x/time/rate"

type RateLimitedWriter struct {
    out     io.Writer
    limiter *rate.Limiter
}

func (w *RateLimitedWriter) Write(p []byte) (n int, err error) {
    // Wait for rate limiter
    w.limiter.Wait(context.Background())
    return w.out.Write(p)
}
```

## Integration with OtterStack

These examples work with any part of OtterStack that uses the compose manager:

```go
// In a deployment
manager := compose.NewManager(workingDir, composeFile, projectName)
manager.SetOutputStreams(customStdout, customStderr)

// All operations use your handlers
manager.Validate(ctx)      // Validation output
manager.Pull(ctx)          // Image pull progress
manager.Up(ctx, envFile)   // Container startup
manager.Down(ctx, false)   // Shutdown messages
```

## Further Reading

- [Go io package](https://pkg.go.dev/io)
- [Docker Compose output formats](https://docs.docker.com/compose/reference/)
- [OtterStack compose manager](../internal/compose/orchestrator.go)

## Contributing

Have a useful output handler pattern? Please submit a PR with:

1. A new example directory
2. Complete, runnable code
3. README explaining the pattern
4. Update to this main README

## Questions?

- Check the example READMEs for detailed explanations
- Review the source code - examples are well-commented
- Open an issue on GitHub
