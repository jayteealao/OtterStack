# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.2.1] - 2026-01-12

### Added
- **Comprehensive test suite** for human-in-the-loop validation of OtterStack deployments:
  - 16 Docker Compose test templates across 7 categories
  - Basic functionality tests (nginx, redis, postgres)
  - Environment variable tests (validation gate, type detection, defaults)
  - Traefik routing tests (zero-downtime deployments, health checks)
  - Multi-service tests (dependencies, startup ordering, volume persistence)
  - Rollback scenario tests (health check failures, startup failures)
  - Edge case tests (concurrent deployments, resource limits)
  - Real application tests (Node.js Express, Go HTTP server with multi-stage builds)
  - Semi-automated test runner script with manual verification points
  - VPS setup script for one-time environment preparation
  - Helper libraries for verification, cleanup, and logging
  - Comprehensive documentation with verification steps for each template

## [v0.2.0] - 2026-01-11

### Added
- **Enhanced environment variable management system** with smart type detection and validation:
  - Automatic parsing of Docker Compose files to extract all environment variable references
  - Support for all Docker Compose variable formats: `${VAR}`, `${VAR:-default}`, `${VAR:?error}`, `$VAR`
  - Intelligent type detection from variable names (URL, EMAIL, PORT, INTEGER, BOOLEAN, STRING)
  - Type-specific validation (URLs require schemes, ports must be 1-65535, emails validated, etc.)
  - Interactive prompts with type-aware UI (Yes/No dialogs for booleans, validated text inputs for others)
  - Automatic `.env.example` generation for documentation

- **Enhanced `project add` command**:
  - Auto-discovers `.env.<project-name>` files in current directory
  - Parses compose file to identify required variables
  - Interactively prompts for missing variables with smart defaults
  - Stores collected variables in database
  - Sets project status to "ready" when all required vars are present

- **New `env scan` subcommand**:
  - Scans existing projects for missing environment variables
  - Identifies which variables are already configured
  - Interactively prompts for missing required and optional variables
  - Updates stored variables in database
  - Generates/updates `.env.example` file

- **Pre-deployment validation gate**:
  - Validates all required environment variables before deployment starts
  - Fails fast with clear, actionable error messages
  - Shows which variables are missing and how to set them
  - Prevents wasted deployment attempts with incomplete configuration

### Changed
- Project setup workflow now includes environment variable configuration
- Environment variables are validated before any Docker operations during deployment
- Clear user feedback throughout variable collection process with grouped prompts (required first, optional second)

### Technical
- Added `internal/compose/env_parser.go` - environment variable extraction from Docker Compose files
- Added `internal/prompt/types.go` - type detection and validation functions
- Added `internal/prompt/env_collector.go` - interactive variable collection with charmbracelet/huh
- Added `internal/validate/env_validator.go` - pre-deployment validation
- Added comprehensive test coverage (97.6%+ on all new packages)
- New dependency: `github.com/charmbracelet/huh@v0.8.0` for interactive TUI prompts

## [v0.2.0-rc.4] - 2026-01-10

### Fixed
- Fixed critical double-locking bug where deployment command acquired locks at both command and orchestrator layers, causing immediate "deployment in progress (lock file exists, age: 0s)" errors
- Removed redundant lock acquisition from command layer (cmd/deploy.go), keeping only orchestrator-level locking for better separation of concerns
- **Fixed environment variable loading during deployment**: Docker Compose now receives env vars during validation and pull operations, eliminating "variable is not set" warnings when env vars are correctly configured
- **Fixed rollback cleanup for failed deployments**: Containers from failed deployments are now properly stopped instead of being left in a restart loop
  - Added `--timeout 0` flag to force immediate SIGKILL for unhealthy containers
  - Increased rollback timeout from 30s to 60s
  - Improved error logging with manual cleanup instructions when automated cleanup fails

### Changed
- Consolidated two separate locking systems (DeploymentLock with O_EXCL and lock.Manager with flock) into single unified system using lock.Manager
- Deployer now uses lock.Manager for more robust locking with PID-based stale detection instead of time-based detection
- Better cross-platform file locking support via gofrs/flock library
- **Reordered deployment flow**: Environment file now written BEFORE validation and pull operations (was after)
- **Validation now uses env vars**: Changed from `Validate()` to `ValidateWithEnv()` to support env var substitution in compose files
- **Image pull now uses env vars**: Added `PullWithEnv()` method to support env vars in image names

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
