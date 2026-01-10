# Progress Parser Example

This example demonstrates parsing Docker Compose output to extract progress information for custom UIs or dashboards.

## What It Does

The `ProgressParser` intercepts Docker output and extracts structured progress events:

```
🔄 [pulling] Pulling myapp-web
🔄 [image] Pulling image library/nginx
✅ [network] Created network myapp_default
✅ [created] Container myapp-db-1 created
🔄 [starting] Container myapp-db-1 starting
✅ [running] Container myapp-db-1 started
✅ [progress] Running 2/2 containers
```

## Key Features

- **Event extraction**: Parse Docker output into structured events
- **Progress tracking**: Track which containers are started, running, or healthy
- **Real-time updates**: Events fire as Docker output arrives
- **Pattern matching**: Regex patterns detect different event types
- **Passthrough output**: Original Docker output still displays

## Usage

```bash
# From the progress_parser directory
go run main.go /path/to/project docker-compose.yml
```

## Event Types

The parser recognizes these event types:

| Event Type | Description | Example |
|------------|-------------|---------|
| `pulling` | Pulling service images | `Pulling myapp-web` |
| `image` | Pulling specific image | `Pulling from library/nginx` |
| `network` | Network created | `Created network myapp_default` |
| `created` | Container created | `Container myapp-db-1 created` |
| `starting` | Container starting | `Container myapp-db-1 starting` |
| `running` | Container started | `Container myapp-db-1 started` |
| `healthy` | Container healthy | `Container myapp-db-1 healthy` |
| `progress` | Overall progress | `Running 3/5 containers` |
| `error` | Error occurred | `Error: failed to pull` |

## ProgressEvent Structure

```go
type ProgressEvent struct {
    Timestamp  time.Time  // When event occurred
    EventType  string     // Type of event (see table above)
    Service    string     // Service or container name
    Message    string     // Human-readable message
    IsComplete bool       // Whether this step is complete
}
```

## How It Works

1. **Intercept output**: Wraps stdout/stderr with `ProgressParser`
2. **Line-by-line parsing**: Each line is matched against regex patterns
3. **Event creation**: Matched lines become `ProgressEvent` objects
4. **Callback invocation**: Events are passed to your callback function
5. **Passthrough**: Original output continues to display

## Implementation Details

```go
type ProgressParser struct {
    out      io.Writer                    // Original output
    onEvent  func(event ProgressEvent)    // Your callback
    patterns map[string]*regexp.Regexp    // Detection patterns
}

func (p *ProgressParser) Write(data []byte) (n int, err error) {
    // Write to original output
    n, err = p.out.Write(data)

    // Parse for events
    // Call onEvent for each detected event

    return n, err
}
```

## When To Use

- **Web UIs**: Build real-time deployment dashboards
- **Progress bars**: Show accurate deployment progress
- **Slack/Discord bots**: Send deployment updates to chat
- **Metrics collection**: Track deployment timing and success rates
- **Testing**: Verify specific deployment steps completed

## Adapting This Example

### Add Custom Patterns

```go
parser.patterns["volume"] = regexp.MustCompile(`Creating volume "(.+)"`)
parser.patterns["health_check"] = regexp.MustCompile(`Container .+ is healthy`)
```

### Different UI Output

```go
onEvent := func(event ProgressEvent) {
    if event.IsComplete {
        // Update progress bar
        progressBar.Increment()
    }

    if event.EventType == "error" {
        // Show error notification
        notifyError(event.Message)
    }
}
```

### Send to External System

```go
onEvent := func(event ProgressEvent) {
    // Send to webhook
    webhookClient.SendEvent(event)

    // Store in database
    db.LogDeploymentEvent(event)

    // Publish to message queue
    messageQueue.Publish("deployment.events", event)
}
```

## Example Output

```
Starting Docker Compose with progress tracking...
Working directory: /srv/myapp
Compose file: docker-compose.yml
Project name: progress-example-1704902400

Validating compose file...

Pulling images (progress will be tracked)...
---
🔄 [image] Pulling image library/nginx
🔄 [image] Pulling image library/postgres
---

Starting services (progress will be tracked)...
---
✅ [network] Created network myapp_default
✅ [created] Container myapp-db-1 created
🔄 [starting] Container myapp-db-1 starting
✅ [running] Container myapp-db-1 started
✅ [created] Container myapp-web-1 created
🔄 [starting] Container myapp-web-1 starting
✅ [running] Container myapp-web-1 started
✅ [progress] Running 2/2 containers
---

=== Deployment Summary ===
Total events: 8
Completed: 6

Events by type:
  network: 1
  created: 2
  starting: 2
  running: 2
  progress: 1

Services started successfully!
```

## Related Examples

- `custom_logger/` - Add timestamps to output
- `file_logger/` - Log to both console and file
