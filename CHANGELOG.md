# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.2.0-rc.4] - 2026-01-10

### Fixed
- Fixed critical double-locking bug where deployment command acquired locks at both command and orchestrator layers, causing immediate "deployment in progress (lock file exists, age: 0s)" errors
- Removed redundant lock acquisition from command layer (cmd/deploy.go), keeping only orchestrator-level locking for better separation of concerns

### Changed
- Consolidated two separate locking systems (DeploymentLock with O_EXCL and lock.Manager with flock) into single unified system using lock.Manager
- Deployer now uses lock.Manager for more robust locking with PID-based stale detection instead of time-based detection
- Better cross-platform file locking support via gofrs/flock library

### Deprecated
- Marked `DeploymentLock`, `AcquireDeploymentLock()`, and `AcquireDeploymentLockWithRetry()` as deprecated in favor of `lock.Manager`
- Deprecated functions remain for backward compatibility but are no longer used in production code

## [v0.2.0-rc.3] - 2026-01-10

### Fixed
- Fixed TOCTOU (time-of-check-time-of-use) race condition in deployment lock system that caused false "lock file exists" errors
- Added automatic retry logic with exponential backoff for transient lock acquisition failures
- Error messages now accurately reflect lock file state (no longer claim file exists when it doesn't)

### Added
- `AcquireDeploymentLockWithRetry()` function for custom retry behavior
- Comprehensive test coverage for lock race conditions
- Documentation in TROUBLESHOOTING.md for lock race condition issues

## [v0.2.0-rc.2] - 2026-01-10

### Fixed
- Fixed nil pointer panic in `DeploymentLock.Release()` when called on nil receiver
- Fixed database constraint error when adding projects with remote git URLs
- Fixed array-format priority label detection in Traefik override generation (now detects both map and array formats)

### Changed
- Docker output now streams in real-time during deployments instead of being buffered
- Users see image pull progress, container creation, and warnings as they happen
- No configuration needed - Docker automatically formats output for your environment

### Breaking Changes

- **Error output format changed**: Error messages no longer include full Docker output by default
  - **Old behavior**: `compose up failed: exit status 1\n[full docker output]`
  - **New behavior**: `compose up failed: exit status 1\n[last 64KB of stderr]`
  - **Migration**: Docker diagnostic output is still included but limited to last 64KB to prevent unbounded memory usage
  - **Impact**: Scripts parsing error messages may need updates if they relied on complete output in error strings

- **Output behavior changed**: Docker output now appears immediately during deployment
  - **Old behavior**: Silent during deployment, all output shown at end
  - **New behavior**: Real-time streaming output as Docker operations proceed
  - **Impact**: CI/CD logs now include real-time progress (may increase log size)
  - **Impact**: Scripts that parse deployment output may need updates for streaming behavior

### Fixed
- Deployments with large images no longer appear frozen during pull operations
- Docker warnings and deprecation notices are now visible during deployment
- Error messages are shown immediately as they occur instead of only at the end
- Race condition in context error checking (timeout vs cancellation) fixed
- Docker diagnostic output now captured in error messages (bounded to 64KB)

### Technical Notes
- Added `SetOutputStreams()` method for custom output redirection in tests
- Created `SafeBuffer` for thread-safe output capture
- Created `boundedBuffer` for memory-bounded error output capture (64KB limit)
- All Docker operations (up, down, pull, restart, validate) now stream output
- Standardized error wrapping using `%w` across all methods
- Added comprehensive GoDoc documentation for all public APIs
- Added CI pipeline with race detector (`go test -race`)
- Added integration tests for timeout vs cancellation handling
- Added edge case tests for nil stream handling
