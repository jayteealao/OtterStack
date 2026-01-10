# feat: Environment Variable Management for Deployments

## Overview

Add environment variable management to OtterStack, allowing users to configure and pass environment variables to Docker Compose deployments.

## Problem Statement

OtterStack has no way to manage environment variables for deployed applications. Users must commit `.env` files to repos (security risk) or manually create them on servers after each deployment.

## Proposed Solution (Simplified)

### CLI Interface

```bash
# Set environment variables
otterstack env set <project> KEY=VALUE [KEY=VALUE...]
otterstack env set <project> --file .env.production

# Get/list environment variables
otterstack env get <project> KEY
otterstack env list <project>
otterstack env list <project> --show-values

# Remove environment variables
otterstack env unset <project> KEY [KEY...]
```

### Storage Design (Simplified)

Use a JSON blob column on the existing `projects` table instead of a normalized table:

```sql
-- internal/state/migrations/002_add_env_vars.sql
ALTER TABLE projects ADD COLUMN env_vars TEXT DEFAULT '{}';

INSERT OR IGNORE INTO schema_migrations (version) VALUES (2);
```

**Rationale**: The access patterns (set/get/list/delete by project) don't require normalized tables. JSON blob is simpler and sufficient.

### Env File Strategy (Simplified)

Use a single env file per project (not SHA-based):

```
~/.otterstack/
├── otterstack.db
├── envfiles/
│   ├── myapp.env        # Single file per project
│   └── otherapp.env
└── worktrees/
```

**Rationale**: Env vars are per-project, not per-deployment. No need for SHA-based snapshots or cleanup logic.

### Deployment Flow

1. User sets env vars: `otterstack env set myapp DATABASE_URL=...`
2. Vars stored as JSON in `projects.env_vars` column
3. During deploy:
   - Read env vars from project record
   - Write to `<dataDir>/envfiles/<project>.env`
   - Pass `--env-file` to `docker compose up`

## Technical Approach

### Files to Modify

| File | Changes |
|------|---------|
| `internal/state/migrations/002_add_env_vars.sql` | New migration (ALTER TABLE) |
| `internal/state/sqlite.go` | Add `SetEnvVars`, `GetEnvVars` methods |
| `internal/state/interfaces.go` | Add interface methods |
| `internal/compose/orchestrator.go` | Modify `Up()` signature |
| `internal/compose/interfaces.go` | Update `ComposeOperations` interface |
| `internal/orchestrator/deployer.go` | Write env file before compose up |
| `internal/validate/input.go` | Add `ValidateEnvKey` function |
| `internal/errors/errors.go` | Add `ErrInvalidEnvKey` |
| `cmd/env.go` | New CLI commands |

### Migration Strategy

Update `migrate()` in `sqlite.go` to handle version 2:

```go
//go:embed migrations/002_add_env_vars.sql
var envVarsMigration string

// In migrate():
if version < 2 {
    if _, err := s.db.Exec(envVarsMigration); err != nil {
        return fmt.Errorf("failed to run env vars migration: %w", err)
    }
}
```

### Interface Changes

**StateStore** (add methods):
```go
SetEnvVars(ctx context.Context, projectID string, vars map[string]string) error
GetEnvVars(ctx context.Context, projectID string) (map[string]string, error)
```

**ComposeOperations** (update signature):
```go
Up(ctx context.Context, envFilePath string) error
```

### Security

- Env files written with `0600` permissions
- Values masked in `env list` by default (use `--show-values` to reveal)
- Input validation: keys must match `^[A-Za-z_][A-Za-z0-9_]*$`
- Values properly quoted in env file output

## Acceptance Criteria

- [ ] `env set <project> KEY=VALUE` stores variable
- [ ] `env set <project> KEY=VALUE` with `=` in value parses correctly
- [ ] `env set <project> --file .env` imports from file
- [ ] `env get <project> KEY` returns the value
- [ ] `env list <project>` shows keys (values masked)
- [ ] `env list <project> --show-values` shows actual values
- [ ] `env unset <project> KEY` removes the variable
- [ ] Deploy passes env vars to containers
- [ ] Env file has `0600` permissions
- [ ] Invalid key names rejected with clear error

## Implementation Plan

### Phase 1: Storage Layer

1. Create `internal/state/migrations/002_add_env_vars.sql`
2. Update `sqlite.go` to embed and run migration 2
3. Add `SetEnvVars` and `GetEnvVars` methods to Store
4. Update `StateStore` interface
5. Add `ValidateEnvKey` to `internal/validate/input.go`
6. Add `ErrInvalidEnvKey` to `internal/errors/errors.go`

### Phase 2: CLI Commands

1. Create `cmd/env.go` with command structure
2. Implement `env set` (including `--file` flag)
3. Implement `env get`
4. Implement `env list` (with `--show-values` flag)
5. Implement `env unset`

### Phase 3: Deployment Integration

1. Update `ComposeOperations.Up()` signature to accept `envFilePath`
2. Update `compose.Manager.Up()` implementation
3. Add `writeEnvFile()` helper to deployer
4. Modify `Deployer.Deploy()` to write env file and pass to compose

### Phase 4: Tests

1. Unit tests for env var storage
2. Unit tests for CLI commands
3. Integration test: deploy with env vars

## What Was Removed (Simplifications)

| Removed | Reason |
|---------|--------|
| Normalized `project_env_vars` table | JSON blob is simpler, same functionality |
| `is_secret` column | Unused - all values masked by default anyway |
| `env import` command | Redundant with `env set --file` |
| `env export` command | Defer - users can use `env list --show-values` |
| SHA-based env file naming | Overcomplicated - single file per project is sufficient |
| Env file cleanup/retention | Not needed with single file approach |
| Timestamps on env vars | Not surfaced in CLI |

## MVP Code Snippets

### cmd/env.go

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables for a project",
	Long:  `Set, get, list, and remove environment variables for OtterStack projects.`,
}

var envSetCmd = &cobra.Command{
	Use:     "set <project> KEY=VALUE [KEY=VALUE...]",
	Short:   "Set environment variables",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runEnvSet,
	Aliases: []string{"add"},
}

var envGetCmd = &cobra.Command{
	Use:   "get <project> <key>",
	Short: "Get an environment variable value",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvGet,
}

var envListCmd = &cobra.Command{
	Use:     "list <project>",
	Short:   "List environment variables",
	Args:    cobra.ExactArgs(1),
	RunE:    runEnvList,
	Aliases: []string{"ls"},
}

var envUnsetCmd = &cobra.Command{
	Use:     "unset <project> <key> [key...]",
	Short:   "Remove environment variables",
	Args:    cobra.MinimumNArgs(2),
	RunE:    runEnvUnset,
	Aliases: []string{"rm"},
}

var envFileFlag string
var showValuesFlag bool

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(envSetCmd, envGetCmd, envListCmd, envUnsetCmd)

	envSetCmd.Flags().StringVarP(&envFileFlag, "file", "f", "", "Import from env file")
	envListCmd.Flags().BoolVar(&showValuesFlag, "show-values", false, "Show actual values")
}
```

### internal/state/sqlite.go (additions)

```go
// SetEnvVars sets environment variables for a project (merge with existing).
func (s *Store) SetEnvVars(ctx context.Context, projectID string, vars map[string]string) error {
	// Get existing vars
	existing, err := s.GetEnvVars(ctx, projectID)
	if err != nil {
		return err
	}

	// Merge
	for k, v := range vars {
		existing[k] = v
	}

	// Serialize and update
	data, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal env vars: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `UPDATE projects SET env_vars = ? WHERE id = ?`, string(data), projectID)
	if err != nil {
		return fmt.Errorf("failed to update env vars: %w", err)
	}
	return nil
}

// GetEnvVars returns environment variables for a project.
func (s *Store) GetEnvVars(ctx context.Context, projectID string) (map[string]string, error) {
	var envJSON string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(env_vars, '{}') FROM projects WHERE id = ?`, projectID).Scan(&envJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get env vars: %w", err)
	}

	var vars map[string]string
	if err := json.Unmarshal([]byte(envJSON), &vars); err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %w", err)
	}
	return vars, nil
}

// DeleteEnvVar removes an environment variable from a project.
func (s *Store) DeleteEnvVar(ctx context.Context, projectID, key string) error {
	vars, err := s.GetEnvVars(ctx, projectID)
	if err != nil {
		return err
	}

	delete(vars, key)

	data, err := json.Marshal(vars)
	if err != nil {
		return fmt.Errorf("failed to marshal env vars: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `UPDATE projects SET env_vars = ? WHERE id = ?`, string(data), projectID)
	return err
}
```

### internal/compose/orchestrator.go (modification)

```go
// Up starts the compose services.
func (m *Manager) Up(ctx context.Context, envFilePath string) error {
	args := m.baseArgs()
	if envFilePath != "" {
		args = append(args, "--env-file", envFilePath)
	}
	args = append(args, "up", "-d", "--wait", "--remove-orphans")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = m.workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
```

## References

- [Docker Compose Environment Variables](https://docs.docker.com/compose/how-tos/environment-variables/)
- [Dokku config:set](https://dokku.com/docs/configuration/environment-variables/)
- `internal/state/sqlite.go:127` - Existing ID generation pattern
- `cmd/project.go:79-184` - CLI command pattern
