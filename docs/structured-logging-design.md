# Structured Logging Design

**Status:** Design Only - Not Implemented
**Implementation:** Only implement if users explicitly request it
**Priority:** P3 (Nice to have)

## Overview

This document defines an optional structured logging interface for OtterStack. The design allows users to integrate their preferred logging library (zap, logrus, slog, etc.) without imposing logging dependencies on users who don't need structured logging.

## Goals

- **Optional**: Logging integration is opt-in, not required
- **Zero dependencies**: Core OtterStack has no logging library dependencies
- **Flexible**: Support popular Go logging libraries through a simple interface
- **Production-ready**: Enable structured logging with contextual fields for observability
- **Non-invasive**: Minimal changes to existing codebase

## Non-Goals

- Built-in logging implementation (users bring their own)
- Log rotation or file management (handled by user's logger)
- Log aggregation or shipping (handled by user's infrastructure)
- Performance monitoring or metrics (use dedicated observability tools)

## Design

### Logger Interface

Define a minimal interface that any structured logger can implement:

```go
// Package logging provides optional structured logging integration.
package logging

import "context"

// Logger is the interface for structured logging in OtterStack.
// Users can implement this interface to integrate their preferred logging library.
type Logger interface {
    // Debug logs a debug-level message with optional fields.
    Debug(msg string, fields ...Field)

    // Info logs an info-level message with optional fields.
    Info(msg string, fields ...Field)

    // Warn logs a warning-level message with optional fields.
    Warn(msg string, fields ...Field)

    // Error logs an error-level message with optional fields.
    Error(msg string, fields ...Field)

    // With returns a new Logger with additional fields attached.
    // All subsequent log calls on the returned logger will include these fields.
    With(fields ...Field) Logger
}

// Field represents a key-value pair for structured logging.
type Field struct {
    Key   string
    Value interface{}
}

// Helper functions for creating fields
func String(key, value string) Field {
    return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
    return Field{Key: key, Value: value}
}

func Error(err error) Field {
    return Field{Key: "error", Value: err}
}

func Duration(key string, value time.Duration) Field {
    return Field{Key: key, Value: value}
}

func Any(key string, value interface{}) Field {
    return Field{Key: key, Value: value}
}
```

### No-Op Logger (Default)

Provide a no-op logger that discards all logs. This is the default when no logger is configured:

```go
// NoOpLogger is a logger that discards all log output.
// This is the default logger when no logger is configured.
type NoOpLogger struct{}

func (l *NoOpLogger) Debug(msg string, fields ...Field) {}
func (l *NoOpLogger) Info(msg string, fields ...Field)  {}
func (l *NoOpLogger) Warn(msg string, fields ...Field)  {}
func (l *NoOpLogger) Error(msg string, fields ...Field) {}
func (l *NoOpLogger) With(fields ...Field) Logger       { return l }

// NewNoOpLogger creates a new no-op logger.
func NewNoOpLogger() Logger {
    return &NoOpLogger{}
}
```

### Context Integration

Store logger in context for easy access throughout the call chain:

```go
type contextKey string

const loggerKey contextKey = "logger"

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, logger Logger) context.Context {
    return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from context.
// Returns NoOpLogger if no logger is found.
func FromContext(ctx context.Context) Logger {
    if logger, ok := ctx.Value(loggerKey).(Logger); ok {
        return logger
    }
    return NewNoOpLogger()
}
```

## Integration Examples

### Example 1: Zap Integration

```go
package main

import (
    "context"
    "go.uber.org/zap"
    "github.com/jayteealao/otterstack/internal/logging"
)

// ZapLogger wraps zap.Logger to implement the logging.Logger interface.
type ZapLogger struct {
    logger *zap.Logger
}

func (l *ZapLogger) Debug(msg string, fields ...logging.Field) {
    l.logger.Debug(msg, l.convertFields(fields)...)
}

func (l *ZapLogger) Info(msg string, fields ...logging.Field) {
    l.logger.Info(msg, l.convertFields(fields)...)
}

func (l *ZapLogger) Warn(msg string, fields ...logging.Field) {
    l.logger.Warn(msg, l.convertFields(fields)...)
}

func (l *ZapLogger) Error(msg string, fields ...logging.Field) {
    l.logger.Error(msg, l.convertFields(fields)...)
}

func (l *ZapLogger) With(fields ...logging.Field) logging.Logger {
    return &ZapLogger{
        logger: l.logger.With(l.convertFields(fields)...),
    }
}

func (l *ZapLogger) convertFields(fields []logging.Field) []zap.Field {
    zapFields := make([]zap.Field, len(fields))
    for i, f := range fields {
        zapFields[i] = zap.Any(f.Key, f.Value)
    }
    return zapFields
}

// NewZapLogger creates a logger that wraps zap.
func NewZapLogger(zapLogger *zap.Logger) logging.Logger {
    return &ZapLogger{logger: zapLogger}
}

// Usage
func main() {
    zapLogger, _ := zap.NewProduction()
    defer zapLogger.Sync()

    logger := NewZapLogger(zapLogger)
    ctx := logging.WithLogger(context.Background(), logger)

    // Use with OtterStack operations
    // deployer.Deploy(ctx, project, opts)
}
```

### Example 2: Logrus Integration

```go
package main

import (
    "context"
    "github.com/sirupsen/logrus"
    "github.com/jayteealao/otterstack/internal/logging"
)

// LogrusLogger wraps logrus.Logger to implement the logging.Logger interface.
type LogrusLogger struct {
    logger *logrus.Logger
    fields logrus.Fields
}

func (l *LogrusLogger) Debug(msg string, fields ...logging.Field) {
    l.logger.WithFields(l.mergeFields(fields)).Debug(msg)
}

func (l *LogrusLogger) Info(msg string, fields ...logging.Field) {
    l.logger.WithFields(l.mergeFields(fields)).Info(msg)
}

func (l *LogrusLogger) Warn(msg string, fields ...logging.Field) {
    l.logger.WithFields(l.mergeFields(fields)).Warn(msg)
}

func (l *LogrusLogger) Error(msg string, fields ...logging.Field) {
    l.logger.WithFields(l.mergeFields(fields)).Error(msg)
}

func (l *LogrusLogger) With(fields ...logging.Field) logging.Logger {
    newFields := make(logrus.Fields)
    for k, v := range l.fields {
        newFields[k] = v
    }
    for _, f := range fields {
        newFields[f.Key] = f.Value
    }
    return &LogrusLogger{
        logger: l.logger,
        fields: newFields,
    }
}

func (l *LogrusLogger) mergeFields(fields []logging.Field) logrus.Fields {
    merged := make(logrus.Fields)
    for k, v := range l.fields {
        merged[k] = v
    }
    for _, f := range fields {
        merged[f.Key] = f.Value
    }
    return merged
}

// NewLogrusLogger creates a logger that wraps logrus.
func NewLogrusLogger(logrusLogger *logrus.Logger) logging.Logger {
    return &LogrusLogger{
        logger: logrusLogger,
        fields: make(logrus.Fields),
    }
}

// Usage
func main() {
    logrusLogger := logrus.New()
    logrusLogger.SetFormatter(&logrus.JSONFormatter{})

    logger := NewLogrusLogger(logrusLogger)
    ctx := logging.WithLogger(context.Background(), logger)

    // Use with OtterStack operations
    // deployer.Deploy(ctx, project, opts)
}
```

### Example 3: Standard Library slog Integration

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "github.com/jayteealao/otterstack/internal/logging"
)

// SlogLogger wraps slog.Logger to implement the logging.Logger interface.
type SlogLogger struct {
    logger *slog.Logger
}

func (l *SlogLogger) Debug(msg string, fields ...logging.Field) {
    l.logger.Debug(msg, l.convertFields(fields)...)
}

func (l *SlogLogger) Info(msg string, fields ...logging.Field) {
    l.logger.Info(msg, l.convertFields(fields)...)
}

func (l *SlogLogger) Warn(msg string, fields ...logging.Field) {
    l.logger.Warn(msg, l.convertFields(fields)...)
}

func (l *SlogLogger) Error(msg string, fields ...logging.Field) {
    l.logger.Error(msg, l.convertFields(fields)...)
}

func (l *SlogLogger) With(fields ...logging.Field) logging.Logger {
    return &SlogLogger{
        logger: l.logger.With(l.convertFields(fields)...),
    }
}

func (l *SlogLogger) convertFields(fields []logging.Field) []any {
    attrs := make([]any, 0, len(fields)*2)
    for _, f := range fields {
        attrs = append(attrs, f.Key, f.Value)
    }
    return attrs
}

// NewSlogLogger creates a logger that wraps slog.
func NewSlogLogger(slogLogger *slog.Logger) logging.Logger {
    return &SlogLogger{logger: slogLogger}
}

// Usage
func main() {
    slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    logger := NewSlogLogger(slogLogger)
    ctx := logging.WithLogger(context.Background(), logger)

    // Use with OtterStack operations
    // deployer.Deploy(ctx, project, opts)
}
```

## Integration Points

The following components would benefit from structured logging:

### 1. Deployer (orchestrator/deployer.go)

```go
func (d *Deployer) Deploy(ctx context.Context, project *state.Project, opts DeployOptions) (*DeployResult, error) {
    logger := logging.FromContext(ctx).With(
        logging.String("project", project.Name),
        logging.String("project_id", project.ID),
    )

    logger.Info("starting deployment",
        logging.String("ref", opts.GitRef),
        logging.Duration("timeout", opts.Timeout),
    )

    // Acquire lock
    logger.Debug("acquiring deployment lock")
    deploymentLock, err := lock.AcquireDeploymentLock(opts.DataDir, project.Name)
    if err != nil {
        logger.Error("failed to acquire deployment lock", logging.Error(err))
        return nil, fmt.Errorf("failed to acquire deployment lock: %w", err)
    }
    defer deploymentLock.Release()

    // Fetch changes
    if project.RepoType == "remote" {
        logger.Info("fetching latest changes")
        if err := d.gitMgr.Fetch(ctx); err != nil {
            logger.Error("fetch failed", logging.Error(err))
            return nil, fmt.Errorf("failed to fetch: %w", err)
        }
    }

    // ... rest of deployment logic with logging

    logger.Info("deployment successful",
        logging.String("sha", shortSHA),
        logging.String("compose_project", composeProjectName),
    )

    return result, nil
}
```

### 2. Compose Manager (compose/orchestrator.go)

```go
func (m *Manager) Up(ctx context.Context, envFilePath string) error {
    logger := logging.FromContext(ctx).With(
        logging.String("project_name", m.projectName),
        logging.String("compose_file", m.composeFile),
    )

    logger.Debug("starting compose up",
        logging.String("env_file", envFilePath),
    )

    start := time.Now()
    err := cmd.Run()
    duration := time.Since(start)

    if err != nil {
        logger.Error("compose up failed",
            logging.Error(err),
            logging.Duration("duration", duration),
        )
        return handleError(ctx, err)
    }

    logger.Info("compose up successful",
        logging.Duration("duration", duration),
    )
    return nil
}
```

### 3. Git Operations (git/worktree.go)

```go
func (m *Manager) CreateWorktree(ctx context.Context, sha, path string) error {
    logger := logging.FromContext(ctx).With(
        logging.String("sha", sha),
        logging.String("path", path),
    )

    logger.Debug("creating worktree")

    err := cmd.Run()
    if err != nil {
        logger.Error("worktree creation failed", logging.Error(err))
        return fmt.Errorf("%w: %w", errors.ErrWorktreeCreateFailed, err)
    }

    logger.Info("worktree created successfully")
    return nil
}
```

### 4. Health Checks (traefik/health.go)

```go
func WaitForHealthy(ctx context.Context, composeMgr compose.ComposeOperations, projectName string, timeout time.Duration) error {
    logger := logging.FromContext(ctx).With(
        logging.String("project_name", projectName),
        logging.Duration("timeout", timeout),
    )

    logger.Info("starting health check")

    ticker := time.NewTicker(checkInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            logger.Warn("health check cancelled")
            return ctx.Err()
        case <-timeoutCtx.Done():
            logger.Error("health check timed out",
                logging.Duration("elapsed", time.Since(start)),
            )
            return ErrHealthCheckTimeout
        case <-ticker.C:
            logger.Debug("checking container health",
                logging.Int("attempt", attempt),
            )
            // ... check logic
        }
    }
}
```

### 5. State Store (state/sqlite.go)

```go
func (s *SQLiteStore) CreateDeployment(ctx context.Context, deployment *state.Deployment) error {
    logger := logging.FromContext(ctx).With(
        logging.String("project_id", deployment.ProjectID),
        logging.String("git_sha", deployment.GitSHA),
    )

    logger.Debug("creating deployment record")

    result, err := s.db.ExecContext(ctx, query, args...)
    if err != nil {
        logger.Error("failed to create deployment", logging.Error(err))
        return fmt.Errorf("failed to create deployment: %w", err)
    }

    logger.Info("deployment record created",
        logging.String("deployment_id", deployment.ID),
    )
    return nil
}
```

## Standard Log Fields

Use consistent field names across the codebase:

| Field Name | Type | Description | Example |
|------------|------|-------------|---------|
| `project` | string | Project name | "myapp" |
| `project_id` | string | Project UUID | "123e4567-e89b" |
| `deployment_id` | string | Deployment UUID | "987f6543-d21c" |
| `git_sha` | string | Full Git SHA | "abc123..." |
| `git_ref` | string | Git reference | "main", "v1.0.0" |
| `compose_project` | string | Docker Compose project name | "myapp-abc123" |
| `service` | string | Docker service name | "web", "db" |
| `worktree_path` | string | Worktree path | "/path/to/worktree" |
| `duration` | duration | Operation duration | "5.2s" |
| `error` | error | Error object | error instance |
| `timeout` | duration | Timeout value | "10m" |
| `attempt` | int | Retry attempt number | 3 |
| `health_status` | string | Container health | "healthy", "unhealthy" |

## Usage Pattern

```go
// In main.go or CLI commands
func main() {
    // User optionally configures logger
    var logger logging.Logger
    if os.Getenv("OTTERSTACK_STRUCTURED_LOGS") == "true" {
        // User integrates their preferred logger
        zapLogger, _ := zap.NewProduction()
        logger = NewZapLogger(zapLogger)
    } else {
        // Default: no logging
        logger = logging.NewNoOpLogger()
    }

    // Add logger to context
    ctx := logging.WithLogger(context.Background(), logger)

    // All operations use context
    deployer.Deploy(ctx, project, opts)
}
```

## Implementation Checklist

When implementing this feature (only if requested by users):

- [ ] Create `internal/logging/logger.go` with interface and field helpers
- [ ] Create `internal/logging/noop.go` with no-op logger implementation
- [ ] Create `internal/logging/context.go` for context integration
- [ ] Add logging calls to `internal/orchestrator/deployer.go`
- [ ] Add logging calls to `internal/compose/orchestrator.go`
- [ ] Add logging calls to `internal/git/worktree.go`
- [ ] Add logging calls to `internal/traefik/health.go`
- [ ] Add logging calls to `internal/state/sqlite.go`
- [ ] Create example integration in `examples/logging/` for zap, logrus, slog
- [ ] Update documentation with logging integration guide
- [ ] Add tests for logger implementations

## Testing Strategy

### Unit Tests

Test that logger calls are made with correct fields:

```go
type TestLogger struct {
    calls []LogCall
}

type LogCall struct {
    Level  string
    Msg    string
    Fields map[string]interface{}
}

func (l *TestLogger) Info(msg string, fields ...logging.Field) {
    call := LogCall{
        Level:  "info",
        Msg:    msg,
        Fields: make(map[string]interface{}),
    }
    for _, f := range fields {
        call.Fields[f.Key] = f.Value
    }
    l.calls = append(l.calls, call)
}

// In tests
func TestDeployerLogging(t *testing.T) {
    testLogger := &TestLogger{}
    ctx := logging.WithLogger(context.Background(), testLogger)

    deployer.Deploy(ctx, project, opts)

    // Assert expected log calls
    assert.Len(t, testLogger.calls, 5)
    assert.Equal(t, "starting deployment", testLogger.calls[0].Msg)
    assert.Equal(t, "myapp", testLogger.calls[0].Fields["project"])
}
```

### Integration Tests

Test actual logger integrations in `examples/`:

```go
func TestZapIntegration(t *testing.T) {
    buf := &bytes.Buffer{}
    encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
    core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.DebugLevel)
    zapLogger := zap.New(core)

    logger := NewZapLogger(zapLogger)
    ctx := logging.WithLogger(context.Background(), logger)

    // Run operation
    deployer.Deploy(ctx, project, opts)

    // Parse and verify JSON logs
    logs := buf.String()
    assert.Contains(t, logs, "starting deployment")
    assert.Contains(t, logs, "\"project\":\"myapp\"")
}
```

## Performance Considerations

- No-op logger has zero allocation overhead
- Field creation is lazy (only when logger is configured)
- Context lookup is O(1) map access
- Logger.With() creates new logger instances (immutable pattern)

## Migration Path

1. **Phase 1**: Add logging package with interface and no-op implementation
2. **Phase 2**: Add logging calls to high-value areas (deployer, compose)
3. **Phase 3**: Add logging to remaining components
4. **Phase 4**: Create example integrations and documentation

## Alternatives Considered

### Alternative 1: Direct Dependency on a Logging Library

Rejected because:
- Imposes dependency on all users
- Limits user choice of logging library
- Harder to change logging implementation later

### Alternative 2: Standard Library log Package

Rejected because:
- No structured logging support
- No log levels
- Limited context support
- Not production-ready for modern observability

### Alternative 3: Printf-style Debug Output

Current approach (fmt.Println). Rejected for structured logging because:
- No machine-readable format
- No log levels
- No field context
- Hard to filter and search

## Open Questions

- Should we provide a simple JSON logger implementation?
  - **Decision**: No. Keep it simple. Users bring their own.
- Should logger be injectable via dependency injection instead of context?
  - **Decision**: Use context. More idiomatic for Go, easier to thread through call chains.
- Should we support log sampling/rate limiting?
  - **Decision**: No. User's logger can handle this.

## References

- [Uber's Zap Logger](https://github.com/uber-go/zap)
- [Logrus](https://github.com/sirupsen/logrus)
- [Go stdlib slog](https://pkg.go.dev/log/slog)
- [Go context patterns](https://go.dev/blog/context)

## Conclusion

This design provides a minimal, optional structured logging interface that:
- Has zero dependencies
- Supports popular Go logging libraries
- Integrates cleanly via context
- Maintains backward compatibility
- Enables production observability when needed

**Remember**: Only implement if users request it. This is a design document, not a requirement.
