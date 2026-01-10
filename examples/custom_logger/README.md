# Custom Logger Example

This example demonstrates how to create a custom output handler that adds timestamps to Docker Compose output.

## What It Does

The `TimestampedWriter` wraps Docker output streams and adds ISO 8601 timestamps to each line:

```
[2025-01-10 14:32:15.234] DOCKER: Pulling myapp-web
[2025-01-10 14:32:15.456] DOCKER: [+] Running 2/2
[2025-01-10 14:32:15.789] DOCKER:  ✔ Container myapp-db-1  Started
[2025-01-10 14:32:16.012] DOCKER:  ✔ Container myapp-web-1 Started
```

## Key Features

- **Line-by-line timestamping**: Each line gets its own timestamp
- **Separate prefixes**: Distinguish between stdout (`DOCKER`) and stderr (`ERROR`)
- **Non-blocking**: Streams output in real-time
- **Production-ready**: Handles partial writes and errors correctly

## Usage

```bash
# From the custom_logger directory
go run main.go /path/to/project docker-compose.yml
```

## How It Works

1. Creates a compose manager with your project
2. Wraps stdout and stderr with `TimestampedWriter`
3. Calls `manager.SetOutputStreams(stdout, stderr)`
4. Runs compose commands (validate, pull, up)
5. All Docker output is automatically timestamped

## Implementation Details

The `TimestampedWriter` implements `io.Writer`:

```go
type TimestampedWriter struct {
    prefix string    // "DOCKER" or "ERROR"
    out    io.Writer // os.Stdout or os.Stderr
}

func (w *TimestampedWriter) Write(p []byte) (n int, err error) {
    // Add timestamp to each line
    // Return number of bytes from original input
}
```

## When To Use

- **Production deployments**: Track timing of each deployment step
- **Debugging**: Correlate Docker events with application logs
- **Auditing**: Create timestamped records of all operations
- **CI/CD pipelines**: Add timestamps without modifying Docker output format

## Adapting This Example

You can customize the timestamp format:

```go
// RFC3339 format
timestamp := time.Now().Format(time.RFC3339)

// Unix timestamp
timestamp := fmt.Sprintf("%d", time.Now().Unix())

// Custom format
timestamp := time.Now().Format("Jan 02 15:04:05")
```

Add additional metadata:

```go
formatted := fmt.Sprintf("[%s] [%s] %s: %s\n",
    timestamp,
    hostname,  // Add hostname
    w.prefix,
    line,
)
```

## Related Examples

- `progress_parser/` - Parse Docker progress for deployment UIs
- `file_logger/` - Log to both console and file simultaneously
