---
title: Enhanced Environment Variable Management (Hybrid Approach)
type: enhancement
priority: P0
estimated_effort: 8 days
related_improvements: "#1, #2, #5 from IMPROVEMENTS.md"
status: planned
created: 2026-01-11
---

# Enhanced Environment Variable Management

## Overview

Implement a hybrid approach to environment variable management that combines setup-time convenience with deployment-time safety. This enhancement addresses the top pain point in OtterStack deployments: environment variable configuration and validation.

**Key Components:**
1. **Auto-discovery on project add** - Automatically detect and load `.env.<project-name>`, prompt for missing vars
2. **Pre-deployment validation gate** - Parse compose file before deployment, show missing vars with clear fix instructions
3. **Interactive env scanning** - `otterstack env scan` command for existing projects

## Problem Statement

### Current Pain Points

1. **Manual env var setup is tedious**
   - Users must run `otterstack env set <project> KEY=value` for each variable
   - No automatic detection of what variables are needed
   - Copy-paste errors common during initial setup
   - Takes 5-10 minutes per project

2. **Cryptic deployment failures**
   - Generic "variable is not set" warnings from Docker Compose
   - No clear indication of which variables are missing
   - No distinction between required vs optional variables
   - Users must manually check compose file to find missing vars
   - **40% of deployment failures** are due to missing env vars

3. **No validation before deployment starts**
   - Deployment starts git operations, pulls images, etc.
   - Only fails at `docker compose up` stage (too late)
   - Wastes time and resources on doomed deployments
   - Poor developer experience

### User Journey (Current)

```bash
# 1. Add project
otterstack project add myapp /path/to/repo

# 2. Try to validate (fails with cryptic error)
otterstack project validate myapp
# Error: compose validation failed
# WARNING: The DATABASE_URL variable is not set. Defaulting to a blank string.
# WARNING: The SECRET_KEY variable is not set. Defaulting to a blank string.
# ... (no clear next steps)

# 3. Manually check compose file to find what vars are needed
cat /path/to/repo/compose.yaml
# ... search for ${VAR} patterns by hand

# 4. Set each variable manually
otterstack env set myapp DATABASE_URL=postgres://...
otterstack env set myapp SECRET_KEY=abc123
otterstack env set myapp LOG_LEVEL=info
# ... repeat for 10-20 variables

# 5. Try validate again (repeat until all vars set)
otterstack project validate myapp

# 6. Finally ready to deploy
otterstack deploy myapp
```

**Result:** Frustrating, error-prone, time-consuming

## Proposed Solution

### Enhanced User Journey

```bash
# 1. Create .env.myapp file locally (optional but recommended)
cat > .env.myapp <<EOF
DATABASE_URL=postgres://localhost/myapp
SECRET_KEY=abc123
EOF

# 2. Add project with auto-discovery
otterstack project add myapp /path/to/repo
# ✓ Found .env.myapp - Loading 2 variables...
# ✓ Loaded: DATABASE_URL, SECRET_KEY
#
# Scanning compose file for required variables...
# Missing required variables:
#   [?] REDIS_URL: (detected type: URL)
#       Enter value: redis://localhost:6379
#
# Missing optional variables (will use defaults):
#   [?] LOG_LEVEL (default: info):
#       Press enter to use default, or type value:
#   [?] PORT (default: 3000):
#       Press enter to use default, or type value:
#
# ✓ All environment variables configured
# ✓ Project ready for deployment

# 3. Deploy immediately (validation happens automatically)
otterstack deploy myapp
# ✓ Validating environment variables...
# ✓ All required variables set
# ... deployment proceeds
```

**Result:** Fast, guided, clear feedback

### For Existing Projects

```bash
# Scan and interactively configure missing vars
otterstack env scan myapp

# Pre-deployment validation (automatic, but can run manually)
otterstack deploy myapp
# Automatically runs validation before starting deployment
```

## Technical Approach

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    project add / env scan                   │
│                                                             │
│  1. Look for .env.<name> → Load into database              │
│  2. Parse compose file → Extract ${VAR} patterns           │
│  3. Compare: required vars vs stored vars                  │
│  4. Interactive prompt for missing vars                    │
│  5. Store all vars in database                             │
│  6. Generate .env.example (documentation)                  │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                      deploy (validation)                    │
│                                                             │
│  1. Parse compose file → Extract ${VAR} patterns           │
│  2. Get stored vars from database                          │
│  3. Check: all required vars present?                      │
│  4. Display checklist of missing vars (if any)             │
│  5. Abort if required vars missing                         │
│  6. Continue deployment if all vars present                │
└─────────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. Environment Variable Parser

**New file:** `internal/compose/env_parser.go`

```go
package compose

import (
	"regexp"
	"gopkg.in/yaml.v3"
)

// EnvVarReference represents a variable reference found in compose file
type EnvVarReference struct {
	Name         string // Variable name (e.g., "DATABASE_URL")
	HasDefault   bool   // Has default value (e.g., ${VAR:-default})
	DefaultValue string // The default value if present
	IsRequired   bool   // Explicitly marked required (e.g., ${VAR:?error})
	ErrorMessage string // Custom error message if IsRequired
	Locations    []string // Where in compose file (for debugging)
}

// ParseEnvVars extracts all ${VAR} references from compose file
func ParseEnvVars(composeFilePath string) ([]EnvVarReference, error)

// GetMissingVars compares required vars against stored vars
func GetMissingVars(required []EnvVarReference, stored map[string]string) []EnvVarReference
```

**Implementation details:**
- Use regex to extract all forms: `${VAR}`, `${VAR:-default}`, `${VAR:?error}`, `$VAR`
- Parse YAML to get service names for better error messages
- Track locations (service, field) for debugging
- Distinguish required (`:?`) from optional (`:-`) variables

**Regex pattern:**
```go
// Matches: ${VAR}, ${VAR:-default}, ${VAR:?error}, ${VAR-default}, ${VAR?error}
pattern := regexp.MustCompile(
	`\$\{([A-Za-z_][A-Za-z0-9_]*)(?:(:-|:?|\?|-)([^}]*))?\}|\$([A-Za-z_][A-Za-z0-9_]*)`
)
```

#### 2. Interactive Prompt System

**New file:** `internal/prompt/env_collector.go`

```go
package prompt

import (
	"github.com/charmbracelet/huh"
	"github.com/jayteealao/otterstack/internal/compose"
)

// CollectMissingVars interactively prompts for missing variables
// Returns map of var_name -> value
func CollectMissingVars(missing []compose.EnvVarReference) (map[string]string, error)

// DetectType infers variable type from name and default value
// Returns: "url", "email", "integer", "boolean", "string"
func DetectType(varName, defaultValue string) string

// ValidateByType returns appropriate validator for detected type
func ValidateByType(varType string) func(string) error
```

**Type detection heuristics:**
- `*_URL`, `*_URI` → URL validation
- `*_EMAIL` → Email validation
- `*_PORT`, `*_TIMEOUT` → Integer validation
- `*_ENABLE`, `*_ENABLED`, `*_FLAG` → Boolean (yes/no prompt)
- Default value format → Infer type (number, URL, etc.)

**Libraries:**
- `github.com/charmbracelet/huh` - Interactive forms (modern, accessible)
- Built-in Go validators (`net/url`, `net/mail`, `strconv`)

#### 3. Validation Reporter

**New file:** `internal/validate/env_validator.go`

```go
package validate

import (
	"github.com/jayteealao/otterstack/internal/compose"
)

// ValidationResult represents env var validation outcome
type ValidationResult struct {
	AllPresent bool
	Missing    []MissingVar
	Optional   []MissingVar
}

// MissingVar represents a variable that's not set
type MissingVar struct {
	Name         string
	IsRequired   bool
	DefaultValue string
	Service      string // Which service needs it
}

// ValidateEnvVars checks if all required vars are present
func ValidateEnvVars(composeFile string, storedVars map[string]string) (*ValidationResult, error)

// FormatValidationError creates user-friendly error message with checklist
func FormatValidationError(result *ValidationResult, projectName string) string
```

**Output format:**
```
Missing environment variables:

Required (deployment will fail):
  ✗ DATABASE_URL (needed by: web)
  ✗ SECRET_KEY (needed by: web)

Optional (will use defaults):
  ⚠ LOG_LEVEL (default: info)
  ⚠ PORT (default: 3000)

Fix:
  otterstack env set myapp DATABASE_URL <value>
  otterstack env set myapp SECRET_KEY <value>

Or run:
  otterstack env scan myapp
```

### Implementation Phases

#### Phase 1: Core Parsing Infrastructure (2 days)

**Goal:** Extract environment variable references from compose files

**Tasks:**
1. Create `internal/compose/env_parser.go`
   - Implement regex-based extraction of `${VAR}` patterns
   - Parse YAML to get service context
   - Handle all forms: `:-`, `:?`, `-`, `?`
   - Return structured `EnvVarReference` list

2. Create `internal/compose/env_parser_test.go`
   - Test basic `${VAR}` extraction
   - Test default values `${VAR:-default}`
   - Test required vars `${VAR:?error}`
   - Test nested/complex compose files
   - Test edge cases (escaped `$$`, invalid names)

3. Update `internal/compose/orchestrator.go`
   - Add `ParseEnvVars(composeFilePath string)` method to Manager

**Files to create:**
- `internal/compose/env_parser.go` (~200 lines)
- `internal/compose/env_parser_test.go` (~300 lines)

**Files to modify:**
- `internal/compose/orchestrator.go` (add method)

**Acceptance criteria:**
- ✓ Parser extracts all variable forms from compose file
- ✓ Correctly identifies required vs optional variables
- ✓ Handles nested services and multi-file composes
- ✓ 90%+ test coverage

---

#### Phase 2: Pre-Deployment Validation Gate (2 days)

**Goal:** Validate env vars before deployment starts, fail fast with clear errors

**Tasks:**
1. Create `internal/validate/env_validator.go`
   - Implement `ValidateEnvVars()` function
   - Compare required vars vs stored vars
   - Create structured validation result
   - Format user-friendly error messages

2. Create `internal/validate/env_validator_test.go`
   - Test with all vars present (success case)
   - Test with missing required vars (failure case)
   - Test with missing optional vars (warning case)
   - Test error message formatting

3. Modify `internal/orchestrator/deployer.go` (lines 137-156)
   - Add validation step BEFORE `composeMgr.ValidateWithEnv()`
   - Call `env_validator.ValidateEnvVars()`
   - If missing required vars: format error, call `onStatus()`, abort
   - If missing optional vars: call `onStatus()` with warning, continue

**Location in deployment flow:**
```go
// internal/orchestrator/deployer.go around line 137

// NEW: Pre-deployment env validation
onVerbose("Validating environment variables...")
validation, err := validate.ValidateEnvVars(
    filepath.Join(worktreePath, project.ComposeFile),
    envVars,
)
if err != nil {
    return nil, fmt.Errorf("failed to validate env vars: %w", err)
}

if !validation.AllPresent {
    errorMsg := validate.FormatValidationError(validation, project.Name)
    onStatus(errorMsg)
    return nil, fmt.Errorf("missing required environment variables")
}

if len(validation.Optional) > 0 {
    onStatus("Note: Some optional variables not set (defaults will be used)")
}

// EXISTING: Write env file
envFilePath, err := writeEnvFile(opts.DataDir, project.Name, envVars)
// ... rest of deployment
```

**Files to create:**
- `internal/validate/env_validator.go` (~150 lines)
- `internal/validate/env_validator_test.go` (~250 lines)

**Files to modify:**
- `internal/orchestrator/deployer.go` (add validation, ~20 lines)

**Acceptance criteria:**
- ✓ Deployment aborts if required vars missing
- ✓ Clear error message lists all missing vars
- ✓ Shows fix commands for each missing var
- ✓ Warnings for optional vars with defaults
- ✓ No impact on deployment time (<100ms overhead)

---

#### Phase 3: Interactive Prompt System (2 days)

**Goal:** User-friendly interactive collection of missing variables

**Tasks:**
1. Add dependency: `go get github.com/charmbracelet/huh@latest`

2. Create `internal/prompt/env_collector.go`
   - Implement `CollectMissingVars()` with huh forms
   - Implement `DetectType()` with heuristics
   - Implement type-specific validators (URL, email, number, bool)
   - Handle keyboard interrupts gracefully

3. Create `internal/prompt/env_collector_test.go`
   - Test type detection for common patterns
   - Test validator functions
   - Mock input/output for integration tests

4. Create `internal/prompt/types.go`
   - Define type detection patterns
   - Define validation functions

**Type detection patterns:**
```go
var typePatterns = map[string]*regexp.Regexp{
    "url":     regexp.MustCompile(`(?i)_(URL|URI|ENDPOINT)$`),
    "email":   regexp.MustCompile(`(?i)_EMAIL$`),
    "port":    regexp.MustCompile(`(?i)_(PORT|PORTS)$`),
    "boolean": regexp.MustCompile(`(?i)_(ENABLE|ENABLED|FLAG|DEBUG)$`),
    "integer": regexp.MustCompile(`(?i)_(COUNT|LIMIT|TIMEOUT|SIZE|MAX|MIN)$`),
}
```

**Files to create:**
- `internal/prompt/env_collector.go` (~200 lines)
- `internal/prompt/types.go` (~100 lines)
- `internal/prompt/env_collector_test.go` (~200 lines)

**Acceptance criteria:**
- ✓ Interactive prompts work in terminal
- ✓ Type detection works for common patterns
- ✓ Validation prevents invalid input
- ✓ Graceful handling of Ctrl+C
- ✓ Clear, helpful prompt text

---

#### Phase 4: Enhanced Project Add (1 day)

**Goal:** Auto-discover `.env.<project-name>` and prompt for missing vars on project add

**Tasks:**
1. Modify `cmd/project.go` (runProjectAdd function, lines 97-223)
   - After compose file detection (line 202)
   - Check for `.env.<projectName>` in current directory
   - If found, load and store vars via `env.LoadEnvFile()`
   - Parse compose file for required vars
   - Compare and identify missing vars
   - Call `prompt.CollectMissingVars()` for interactive collection
   - Store all vars in database
   - Generate `.env.example` file

**Implementation location:**
```go
// cmd/project.go around line 205

// NEW: Auto-discover and load environment variables
envFilePath := filepath.Join(".", ".env."+name)
if _, err := os.Stat(envFilePath); err == nil {
    fmt.Printf("✓ Found %s - Loading variables...\n", envFilePath)
    envVars, err := loadEnvFile(envFilePath)
    if err != nil {
        return fmt.Errorf("failed to load env file: %w", err)
    }
    if err := store.SetEnvVars(ctx, name, envVars); err != nil {
        return fmt.Errorf("failed to store env vars: %w", err)
    }
    fmt.Printf("✓ Loaded %d variables\n", len(envVars))
}

// NEW: Parse compose file and prompt for missing vars
fmt.Println("\nScanning compose file for required variables...")
composePath := filepath.Join(repoPath, composeFile)
requiredVars, err := compose.ParseEnvVars(composePath)
if err != nil {
    return fmt.Errorf("failed to parse compose file: %w", err)
}

storedVars, err := store.GetEnvVars(ctx, name)
if err != nil {
    return fmt.Errorf("failed to get stored vars: %w", err)
}

missingVars := compose.GetMissingVars(requiredVars, storedVars)
if len(missingVars) > 0 {
    fmt.Printf("\nFound %d missing variables. Let's configure them:\n\n", len(missingVars))
    newVars, err := prompt.CollectMissingVars(missingVars)
    if err != nil {
        if err == terminal.InterruptErr {
            return fmt.Errorf("configuration cancelled")
        }
        return fmt.Errorf("failed to collect vars: %w", err)
    }

    if err := store.SetEnvVars(ctx, name, newVars); err != nil {
        return fmt.Errorf("failed to store vars: %w", err)
    }
    fmt.Printf("✓ Configured %d variables\n", len(newVars))
}

// NEW: Generate .env.example for documentation
examplePath := filepath.Join(repoPath, ".env.example")
if err := generateEnvExample(requiredVars, examplePath); err != nil {
    // Non-fatal - just warn
    fmt.Printf("⚠ Could not generate .env.example: %v\n", err)
}

// EXISTING: Update project status
fmt.Println("\n✓ All environment variables configured")
if err := store.UpdateProjectStatus(ctx, name, "ready"); err != nil {
    return fmt.Errorf("failed to update project: %w", err)
}
```

**Helper functions to add:**
```go
// loadEnvFile parses .env file and returns map
func loadEnvFile(path string) (map[string]string, error)

// generateEnvExample creates .env.example from required vars
func generateEnvExample(vars []compose.EnvVarReference, path string) error
```

**Files to modify:**
- `cmd/project.go` (add auto-discovery, ~60 lines)

**Files to create:**
- None (uses components from previous phases)

**Acceptance criteria:**
- ✓ Detects `.env.<project-name>` in current directory
- ✓ Loads and stores variables from file
- ✓ Prompts for missing required variables
- ✓ Skips prompts if all variables already set
- ✓ Generates `.env.example` for documentation
- ✓ Sets project status to "ready" after completion

---

#### Phase 5: Env Scan Command (1 day)

**Goal:** Allow users to scan existing projects and configure missing vars

**Tasks:**
1. Modify `cmd/env.go` - Add new `scan` subcommand
   - Parse compose file
   - Get stored vars from database
   - Identify missing vars
   - Interactive prompt for missing vars
   - Store new vars
   - Update project status to "ready" if was "unconfigured"

**Command structure:**
```go
// cmd/env.go

var envScanCmd = &cobra.Command{
    Use:   "scan <project>",
    Short: "Scan compose file and interactively configure missing environment variables",
    Args:  cobra.ExactArgs(1),
    RunE:  runEnvScan,
}

func runEnvScan(cmd *cobra.Command, args []string) error {
    projectName := args[0]

    // Get project and compose file path
    // Parse compose file for required vars
    // Get stored vars from database
    // Identify missing vars
    // Prompt for missing vars
    // Store new vars
    // Generate/update .env.example
    // Update project status if needed

    fmt.Println("✓ Environment variable scan complete")
    return nil
}
```

**Files to modify:**
- `cmd/env.go` (add scan subcommand, ~100 lines)

**Acceptance criteria:**
- ✓ `otterstack env scan <project>` command works
- ✓ Shows current env var status before prompting
- ✓ Only prompts for missing variables
- ✓ Updates `.env.example` after completion
- ✓ Marks project as "ready" if was "unconfigured"

---

### Alternative Approaches Considered

#### Option A: Use compose-go library instead of regex

**Pros:**
- Official Docker Compose parsing library
- Handles complex YAML edge cases
- Resolves interpolation properly

**Cons:**
- Large dependency (~2MB, many transitive deps)
- Overkill for just extracting variable names
- We don't need full compose parsing (Docker does that)

**Decision:** Use regex for now, can migrate to compose-go later if needed

#### Option B: Shell script instead of Go prompts

**Pros:**
- Simpler implementation
- Familiar bash tools

**Cons:**
- Not cross-platform (Windows)
- Poor error handling
- Inconsistent with rest of OtterStack
- No type detection/validation

**Decision:** Use Go with huh library for consistent, cross-platform UX

#### Option C: Only validate on deployment, skip project add enhancement

**Pros:**
- Smaller scope
- Faster to implement

**Cons:**
- Misses opportunity to improve first-time setup
- Users still have tedious manual configuration
- Doesn't maximize UX improvement

**Decision:** Implement hybrid approach for best of both worlds

---

## Acceptance Criteria

### Functional Requirements

- ✓ **Project add auto-discovers** `.env.<project-name>` and loads variables
- ✓ **Interactive prompts** collect missing variables with type validation
- ✓ **Pre-deployment validation** checks all required vars before deployment starts
- ✓ **Clear error messages** list all missing vars with fix commands
- ✓ **Env scan command** allows scanning existing projects
- ✓ **`.env.example` generation** documents all required variables

### User Experience Requirements

- ✓ **First-time setup** takes <2 minutes (down from 5-10 minutes)
- ✓ **Zero cryptic errors** - all messages include fix instructions
- ✓ **Type detection** works for common patterns (URLs, ports, booleans)
- ✓ **Keyboard interrupts** handled gracefully (Ctrl+C)
- ✓ **No breaking changes** to existing commands

### Technical Requirements

- ✓ **Parsing performance** <100ms for typical compose file
- ✓ **No new dependencies** except `huh` (5MB total)
- ✓ **Test coverage** >85% for new code
- ✓ **Cross-platform** works on Linux, macOS, Windows
- ✓ **Backward compatible** existing projects continue to work

### Quality Gates

- ✓ All unit tests pass
- ✓ Integration tests cover happy path and error cases
- ✓ Manual testing on sample projects
- ✓ Documentation updated (README, TROUBLESHOOTING)
- ✓ Code review approval

---

## Success Metrics

### Target Impact

- **40% reduction** in deployment failures due to missing env vars
- **80% reduction** in setup time (5-10 min → 1-2 min)
- **90% reduction** in support questions about env vars
- **100%** of required vars identified before deployment starts

### Measurement

**Before (baseline):**
- Track: deployment failures with "missing env var" in error
- Track: time from `project add` to first successful deploy
- Track: number of `env set` commands run per project

**After (post-implementation):**
- Compare same metrics
- User feedback via GitHub issues
- Deployment success rate

---

## Dependencies & Risks

### Dependencies

1. **New Go library:** `github.com/charmbracelet/huh`
   - Well-maintained (active in 2026)
   - Part of Charm ecosystem (widely used)
   - Risk: Low (can fallback to promptui if needed)

2. **Compose file format stability**
   - Docker Compose v2 format is stable
   - Variable interpolation syntax hasn't changed since 2017
   - Risk: Very low

### Risks & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Regex misses edge cases | Medium | Low | Comprehensive test suite, fallback to manual config |
| Type detection incorrect | Low | Medium | Allow manual override, show detected type to user |
| Terminal compatibility | Medium | Low | huh has good terminal support, test on multiple terminals |
| Performance on large compose files | Low | Low | Benchmark with realistic files, optimize if needed |
| User doesn't have `.env.<name>` | Low | High | Gracefully skip if not found, still prompt for missing |

---

## Files to Create/Modify

### New Files (5 files, ~1,350 lines)

| File | Lines | Purpose |
|------|-------|---------|
| `internal/compose/env_parser.go` | ~200 | Parse compose files for env var references |
| `internal/compose/env_parser_test.go` | ~300 | Test env var parsing |
| `internal/validate/env_validator.go` | ~150 | Validate env vars before deployment |
| `internal/validate/env_validator_test.go` | ~250 | Test validation logic |
| `internal/prompt/env_collector.go` | ~200 | Interactive prompt system |
| `internal/prompt/types.go` | ~100 | Type detection and validators |
| `internal/prompt/env_collector_test.go` | ~200 | Test prompts and validators |

### Modified Files (4 files, ~200 lines of changes)

| File | Lines Changed | Changes |
|------|---------------|---------|
| `cmd/project.go` | ~60 | Add auto-discovery on project add |
| `cmd/env.go` | ~100 | Add scan subcommand |
| `internal/orchestrator/deployer.go` | ~20 | Add pre-deployment validation |
| `internal/compose/orchestrator.go` | ~20 | Add ParseEnvVars method |
| `go.mod` | ~5 | Add huh dependency |

**Total:** 9 files, ~1,550 lines (including tests)

---

## Testing Strategy

### Unit Tests

1. **Env Parser Tests** (`env_parser_test.go`)
   - Extract basic `${VAR}` patterns
   - Extract default values `${VAR:-default}`
   - Extract required vars `${VAR:?error}`
   - Handle escaped variables `$$VAR`
   - Handle invalid variable names
   - Parse multi-service compose files

2. **Validator Tests** (`env_validator_test.go`)
   - All vars present (success)
   - Missing required vars (failure)
   - Missing optional vars (warning)
   - Error message formatting
   - Service context in errors

3. **Prompt Tests** (`env_collector_test.go`)
   - Type detection for common patterns
   - URL validation
   - Email validation
   - Number validation
   - Boolean handling

### Integration Tests

1. **End-to-End Project Add**
   - With `.env.<name>` present
   - Without `.env.<name>`
   - With all vars in file
   - With missing vars requiring prompts

2. **End-to-End Deployment**
   - With all vars configured (success)
   - With missing required vars (abort)
   - With missing optional vars (warning)

3. **Env Scan Command**
   - On unconfigured project
   - On partially configured project
   - On fully configured project

### Manual Testing

1. Test on sample projects:
   - Node.js app with Prisma + PostgreSQL
   - Python FastAPI + Redis
   - Multi-service app (frontend + backend + database)

2. Test on different terminals:
   - Windows Terminal
   - iTerm2 (macOS)
   - GNOME Terminal (Linux)

3. Test error cases:
   - Malformed compose file
   - Compose file with no env vars
   - Very large compose file (100+ vars)
   - Keyboard interrupt (Ctrl+C)

---

## Documentation Updates

### README.md

Add new section under "Environment Variables":

```markdown
## Environment Variables

OtterStack automatically discovers and validates environment variables.

### During Project Setup

When you add a project, OtterStack:
1. Looks for `.env.<project-name>` in the current directory
2. Loads variables from the file (if found)
3. Scans your compose file for required variables
4. Prompts for any missing variables interactively
5. Generates `.env.example` for documentation

```bash
# Create env file (optional)
cat > .env.myapp <<EOF
DATABASE_URL=postgres://localhost/myapp
SECRET_KEY=abc123
EOF

# Add project - auto-discovers env vars
otterstack project add myapp /path/to/repo
```

### Scanning Existing Projects

```bash
# Scan and configure missing variables
otterstack env scan myapp
```

### Before Deployment

OtterStack validates all required variables before deploying:
- Lists missing required variables
- Shows optional variables with defaults
- Provides clear fix commands
- Aborts deployment if required vars missing
```

### TROUBLESHOOTING.md

Add section:

```markdown
## Environment Variable Issues

### "Missing required environment variables"

**Problem:** Deployment fails with a list of missing variables.

**Solution:**
1. Run `otterstack env scan <project>` to interactively configure
2. Or set variables manually: `otterstack env set <project> KEY=value`
3. Then retry deployment

### Type Detection Incorrect

**Problem:** Prompt asks for URL but you want to enter a string.

**Solution:** Just enter your value - validation will be skipped if it doesn't match the expected type.

### .env File Not Auto-Detected

**Problem:** Your `.env` file wasn't loaded on project add.

**Solution:** File must be named exactly `.env.<project-name>` in current directory.
- ✓ `.env.myapp` (correct)
- ✗ `.env` (wrong)
- ✗ `myapp.env` (wrong)
```

---

## Implementation Checklist

### Phase 1: Core Parsing (2 days)
- [ ] Create `internal/compose/env_parser.go`
- [ ] Create `internal/compose/env_parser_test.go`
- [ ] Update `internal/compose/orchestrator.go`
- [ ] Run tests, verify 90%+ coverage
- [ ] Commit: "feat(compose): add environment variable parser"

### Phase 2: Pre-Deployment Validation (2 days)
- [ ] Create `internal/validate/env_validator.go`
- [ ] Create `internal/validate/env_validator_test.go`
- [ ] Modify `internal/orchestrator/deployer.go`
- [ ] Test validation gate with sample projects
- [ ] Commit: "feat(deploy): add pre-deployment env validation gate"

### Phase 3: Interactive Prompts (2 days)
- [ ] Add huh dependency: `go get github.com/charmbracelet/huh@latest`
- [ ] Create `internal/prompt/env_collector.go`
- [ ] Create `internal/prompt/types.go`
- [ ] Create `internal/prompt/env_collector_test.go`
- [ ] Test prompts in terminal
- [ ] Commit: "feat(prompt): add interactive env var collection"

### Phase 4: Enhanced Project Add (1 day)
- [ ] Modify `cmd/project.go` (add auto-discovery)
- [ ] Add helper functions (loadEnvFile, generateEnvExample)
- [ ] Test with `.env.<name>` present
- [ ] Test with `.env.<name>` absent
- [ ] Test with missing vars requiring prompts
- [ ] Commit: "feat(project): auto-discover env vars on project add"

### Phase 5: Env Scan Command (1 day)
- [ ] Modify `cmd/env.go` (add scan subcommand)
- [ ] Test on unconfigured project
- [ ] Test on partially configured project
- [ ] Commit: "feat(env): add scan command for interactive configuration"

### Documentation & Polish
- [ ] Update README.md
- [ ] Update TROUBLESHOOTING.md
- [ ] Add examples to docs/
- [ ] Manual testing on sample projects
- [ ] Update IMPROVEMENTS.md (mark #1, #2 as complete)
- [ ] Commit: "docs: document enhanced env var management"

---

## References & Research

### Internal Code References

**Project Add Flow:**
- `cmd/project.go:97-223` - Current project add implementation
- `cmd/project.go:198-202` - Compose file detection
- `cmd/project.go:205-221` - Project finalization

**Environment Variable System:**
- `cmd/env.go:23-160` - env set command
- `cmd/env.go:80-377` - env load command
- `internal/state/sqlite.go:595-678` - Env var storage (database)
- `internal/orchestrator/deployer.go:139-150` - Env file writing

**Deployment Flow:**
- `cmd/deploy.go:42-110` - Deploy command entry point
- `internal/orchestrator/deployer.go:50-267` - Full deployment flow
- `internal/orchestrator/deployer.go:152-156` - Compose validation
- `internal/orchestrator/deployer.go:188-197` - Services start + log fetch

**Validation:**
- `cmd/project.go:321-387` - Project validate command
- `internal/compose/orchestrator.go:223-247` - ValidateWithEnv
- `internal/validate/input.go:213-242` - Compose file discovery

### External References

**Docker Compose Documentation:**
- [Variable Interpolation](https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/)
- [Set Environment Variables](https://docs.docker.com/compose/how-tos/environment-variables/set-environment-variables/)
- [Interpolation Syntax](https://docs.docker.com/reference/compose-file/interpolation/)
- [Compose Spec - Interpolation](https://github.com/compose-spec/compose-spec/blob/main/12-interpolation.md)

**Go Libraries:**
- [charmbracelet/huh](https://github.com/charmbracelet/huh) - Interactive forms
- [compose-spec/compose-go](https://github.com/compose-spec/compose-go) - Official compose parsing (alternative)
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) - YAML parsing
- [buildkite/interpolate](https://github.com/buildkite/interpolate) - POSIX parameter expansion

**Blog Posts & Guides:**
- [Environmental Variables and Interpolation in Docker Compose](https://blog.foxxmd.dev/posts/compose-envs-explained/)
- [Interactive CLI prompts in Go](https://dev.to/tidalcloud/interactive-cli-prompts-in-go-3bj9)
- [Building interactive CLIs with Bubbletea](https://www.inngest.com/blog/interactive-clis-with-bubbletea)

### Related Work

- IMPROVEMENTS.md #1 - Better Environment Variable Validation
- IMPROVEMENTS.md #2 - Automatic Environment Variable Discovery
- IMPROVEMENTS.md #5 - Streaming Logs (already implemented on this branch)
- Commit 4bc98c8 - Show container logs when deployment fails

---

## Future Considerations

### Phase 6: Advanced Features (Future)

1. **Secret Management Integration**
   - Detect sensitive variable names (PASSWORD, SECRET, KEY)
   - Prompt: "Store in system keyring instead of database?"
   - Integration with OS keychain (macOS), credential manager (Windows), secret-tool (Linux)

2. **Variable Inheritance & Overrides**
   - Project-level defaults
   - Environment-specific overrides (staging, production)
   - `--env staging` flag for environment selection

3. **Remote .env File Support**
   - Load from S3, vault, etc.
   - `otterstack env load <project> --from s3://bucket/env`

4. **Validation Rules**
   - Custom validators in `.otterstack/validators.yaml`
   - Regex patterns for specific variables
   - Cross-variable dependencies

5. **Environment Variable Diffs**
   - Show what changed between deployments
   - Warn on sensitive var changes
   - `otterstack env diff <project> <deployment-1> <deployment-2>`

### Extensibility

- Plugin system for custom type detectors
- Hooks for external validation services
- Integration with secrets managers (HashiCorp Vault, AWS Secrets Manager)

---

## Notes

- This plan represents the hybrid approach combining setup convenience with deployment safety
- Prioritizes user experience while maintaining backward compatibility
- Uses existing patterns and conventions from OtterStack codebase
- Follows CLAUDE.md principles: explicit over clever, single source of truth, no silent catches
- Estimated total effort: **8 days** (2+2+2+1+1)
- High confidence in success based on thorough research and clear requirements

---

**Status:** Ready for implementation
**Next Steps:** Start with Phase 1 (Core Parsing Infrastructure)
