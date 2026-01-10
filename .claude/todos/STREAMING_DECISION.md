# Streaming vs Buffering Decision for Compose Methods

## Summary

Updated `internal/compose/orchestrator.go` to implement consistent streaming strategy for Docker Compose operations.

## Implementation Decision: Hybrid Approach

### Methods That Now Stream (Updated)

1. **Validate()** - Streams validation errors in real-time
   - **Why**: Users need to see validation errors immediately
   - **How**: Uses `cmd.Stdout/Stderr` with `Run()`
   - **Rationale**: The `--quiet` flag only suppresses success messages; errors still appear

2. **ValidateWithEnv()** - Streams validation errors with env file support
   - **Why**: Same as Validate(), but with environment variable substitution
   - **How**: Uses `cmd.Stdout/Stderr` with `Run()`
   - **Rationale**: Consistent with Validate()

### Methods That Already Streamed (No Changes)

3. **Up()** - Streams docker compose up output
4. **Down()** - Streams docker compose down output
5. **Pull()** - Streams image pull progress
6. **Restart()** - Streams restart output

### Methods That Use Buffering (Documented Why)

7. **Status()** - Uses buffered output
   - **Why**: Needs to parse output into `[]ServiceStatus` struct
   - **Rationale**: Returns structured data, not raw output
   - **Documentation Added**: Explains why buffering is necessary

8. **Logs()** - Uses buffered output
   - **Why**: Interface signature returns `string`
   - **Rationale**: Callers expect complete log output as return value
   - **Documentation Added**: Explains interface contract requires buffering

## Changes Made

### Files Modified
- `C:\Users\jayte\Documents\dev\OtterStack\internal\compose\orchestrator.go`

### Code Changes

1. **Validate()**: Changed from `CombinedOutput()` to streaming with `cmd.Stdout/Stderr`
2. **ValidateWithEnv()**: Changed from `CombinedOutput()` to streaming with `cmd.Stdout/Stderr`
3. **Status()**: Added documentation explaining why buffering is used
4. **Logs()**: Added documentation explaining why buffering is used

### Tests
All existing tests pass:
- ✓ TestManager_Validate
- ✓ TestManager_Validate/valid_compose_file
- ✓ TestManager_Validate/invalid_compose_file
- ✓ All other compose tests

## Benefits

1. **Consistency**: Most methods now stream (Up, Down, Pull, Restart, Validate, ValidateWithEnv)
2. **User Experience**: Users see validation errors in real-time
3. **Clear Documentation**: Methods that buffer explain why they differ
4. **No Breaking Changes**: Interface signatures unchanged

## Streaming Pattern Used

```go
cmd := exec.CommandContext(ctx, "docker", args...)
cmd.Dir = m.workingDir
cmd.Stdout = m.getStdout()  // Streams to configured output
cmd.Stderr = m.getStderr()  // Streams to configured output

err := cmd.Run()  // Run without capturing output
```

## Buffering Pattern Used (Where Necessary)

```go
cmd := exec.CommandContext(ctx, "docker", args...)
cmd.Dir = m.workingDir

output, err := cmd.Output()  // Capture output for parsing
// Parse output into structured data
return parsedData, nil
```

## Rationale Summary

The hybrid approach balances:
- **Real-time feedback** for operations where users want to see progress
- **Structured data** for methods that return parsed information
- **Interface compatibility** for methods with string return signatures

This resolves the inconsistency identified in TODO #006 while maintaining backward compatibility and providing a better user experience.
