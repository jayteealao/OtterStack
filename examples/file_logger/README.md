# File Logger Example

This example demonstrates logging Docker Compose output to both console and files simultaneously using `io.MultiWriter`.

## What It Does

The `LogManager` uses Go's `io.MultiWriter` to send output to multiple destinations:

- **Console**: Real-time output for the user
- **Combined log**: All stdout and stderr in one file
- **Stdout log**: Only standard output
- **Stderr log**: Only error output

All files are timestamped for easy identification.

## Key Features

- **Multiple outputs**: Write to console and files simultaneously
- **Zero overhead**: No buffering or parsing, direct stream splitting
- **Timestamped files**: Each run creates new log files with timestamps
- **Separate streams**: Keep stdout and stderr separated in dedicated files
- **Production-ready**: Handles errors, ensures files are closed properly

## Usage

```bash
# From the file_logger directory

# Use default ./logs directory
go run main.go /path/to/project docker-compose.yml

# Specify custom log directory
go run main.go /path/to/project docker-compose.yml /var/log/deployments
```

## Log Files Created

Each run creates three timestamped log files:

```
logs/
├── combined_2025-01-10_14-32-15.log    # All output (stdout + stderr)
├── stdout_2025-01-10_14-32-15.log      # Only stdout
└── stderr_2025-01-10_14-32-15.log      # Only stderr
```

## How It Works

1. **Create log files**: Opens three files with timestamps
2. **Build multi-writers**:
   - `stdout = console + stdout.log + combined.log`
   - `stderr = console + stderr.log + combined.log`
3. **Connect to compose**: `manager.SetOutputStreams(stdout, stderr)`
4. **Automatic writing**: Everything written to stdout/stderr goes to all destinations

## Implementation Details

```go
type LogManager struct {
    logDir     string
    logFile    *os.File  // combined.log
    stdoutFile *os.File  // stdout.log
    stderrFile *os.File  // stderr.log
    stdout     io.Writer // MultiWriter
    stderr     io.Writer // MultiWriter
}

// Create multi-writers using io.MultiWriter
stdout := io.MultiWriter(os.Stdout, stdoutFile, logFile)
stderr := io.MultiWriter(os.Stderr, stderrFile, logFile)
```

## When To Use

- **Production deployments**: Keep permanent records of all deployments
- **Compliance/Auditing**: Maintain logs for regulatory requirements
- **Debugging**: Review deployment output after the fact
- **CI/CD pipelines**: Save logs as build artifacts
- **Troubleshooting**: Compare successful vs failed deployments

## Advantages of io.MultiWriter

- **No buffering**: Output appears immediately on all destinations
- **Zero parsing**: No need to parse or modify output
- **Thread-safe**: Safe for concurrent writes
- **Composable**: Easy to add or remove destinations
- **Efficient**: Direct writes, minimal overhead

## Adapting This Example

### Add More Destinations

```go
// Add syslog
syslogWriter, _ := syslog.New(syslog.LOG_INFO, "otterstack")

// Add network logger
networkWriter := NewNetworkLogger("https://logs.example.com/api")

// Combine all writers
stdout := io.MultiWriter(
    os.Stdout,
    stdoutFile,
    logFile,
    syslogWriter,
    networkWriter,
)
```

### Add Filtering

```go
// Filter out sensitive data
type FilterWriter struct {
    writer  io.Writer
    pattern *regexp.Regexp
}

func (f *FilterWriter) Write(p []byte) (n int, err error) {
    // Redact secrets before writing
    filtered := f.pattern.ReplaceAll(p, []byte("[REDACTED]"))
    return f.writer.Write(filtered)
}

// Use in multi-writer
filtered := &FilterWriter{
    writer:  logFile,
    pattern: regexp.MustCompile(`password=\S+`),
}

stdout := io.MultiWriter(os.Stdout, filtered)
```

### Add Compression

```go
import "compress/gzip"

// Create gzip writer
gzipFile, _ := os.Create("combined.log.gz")
gzipWriter := gzip.NewWriter(gzipFile)
defer gzipWriter.Close()

// Include in multi-writer
stdout := io.MultiWriter(
    os.Stdout,
    logFile,           // Uncompressed
    gzipWriter,        // Compressed
)
```

### Add Structured Logging

```go
// Wrap with structured logger
type StructuredLogger struct {
    writer io.Writer
}

func (s *StructuredLogger) Write(p []byte) (n int, err error) {
    // Convert to JSON
    entry := map[string]interface{}{
        "timestamp": time.Now().Unix(),
        "source":    "docker",
        "message":   string(p),
    }
    data, _ := json.Marshal(entry)
    return s.writer.Write(append(data, '\n'))
}
```

## Example Output

```
Starting Docker Compose with file logging...
Working directory: /srv/myapp
Compose file: docker-compose.yml
Log directory: ./logs
Project name: file-logger-example-1704902400

Output will be displayed on console AND saved to log files:
  - ./logs/combined_*.log (all output)
  - ./logs/stdout_*.log (standard output)
  - ./logs/stderr_*.log (error output)

Validating compose file...
Validation successful!

Pulling images...
---
[+] Pulling 2/2
 ✔ library/nginx:alpine Pulled
 ✔ library/postgres:15 Pulled
---

Starting services...
---
[+] Running 3/3
 ✔ Network myapp_default    Created
 ✔ Container myapp-db-1     Started
 ✔ Container myapp-web-1    Started
---

Services started successfully!

Log files created:
  ./logs/combined_2025-01-10_14-32-15.log (4523 bytes)
  ./logs/stdout_2025-01-10_14-32-15.log (4321 bytes)
  ./logs/stderr_2025-01-10_14-32-15.log (202 bytes)

To stop services: docker compose -p file-logger-example-1704902400 down
```

## Log Rotation

For production use, consider adding log rotation:

```go
import "gopkg.in/natefinch/lumberjack.v2"

logFile := &lumberjack.Logger{
    Filename:   filepath.Join(logDir, "combined.log"),
    MaxSize:    100, // megabytes
    MaxBackups: 10,
    MaxAge:     30, // days
    Compress:   true,
}

stdout := io.MultiWriter(os.Stdout, logFile)
```

## Related Examples

- `custom_logger/` - Add timestamps to each line
- `progress_parser/` - Parse output for progress events
