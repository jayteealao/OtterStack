# OtterStack: Git-Driven Docker Compose Deployment Tool

**Type:** Enhancement (New Project)
**Priority:** High
**Created:** 2026-01-08
**Status:** Planning

---

## Overview

OtterStack is a multi-project, Git-driven Docker Compose deployment tool for single Linux VPS environments. It enables zero-friction deployments from GitHub URLs (or any Git remote) with deterministic releases, safe rollbacks, strong observability, and a read-only TUI dashboard.

**Core value proposition:**
> "Here's the GitHub repo" → "It's running"

With no ceremony, no surprises, and no magic.

---

## Problem Statement

Running Docker Compose apps on a VPS involves manual, error-prone steps:

1. **Manual cloning**: SSH into server, clone repos by hand, manage directory layout
2. **Ad-hoc deploys**: No standardized deployment process, inconsistent across projects
3. **Unclear history**: No audit trail of what was deployed when
4. **Fragile rollbacks**: Rollback requires manual intervention, often fails under pressure
5. **URL friction**: Requiring pre-cloned repos adds setup overhead

Existing tools (Kamal, Coolify, CapRover) either:
- Focus on container registries rather than Git-native workflows
- Add too much abstraction over Docker Compose
- Require additional infrastructure (proxies, databases, control planes)

OtterStack fills the gap: **pure Git + pure Compose, nothing more**.

---

## Proposed Solution

A single Go binary that provides:

1. **CLI** (`otterstack`): Project management, deployment, rollback commands
2. **Repo Manager**: Clones Git URLs, manages fetches, provides base for worktrees
3. **Worktree Manager**: Creates per-deployment worktrees for atomic releases
4. **Compose Orchestrator**: Validates and executes `docker compose up --wait`
5. **State Store**: SQLite database tracking projects, deployments, history
6. **TUI Dashboard**: Read-only monitoring of all projects and their health

---

## Technical Approach

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        OtterStack CLI                           │
│  (Cobra commands: project, deploy, rollback, status, monitor)   │
└─────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐      ┌───────────────┐      ┌───────────────┐
│  Repo Manager │      │   Worktree    │      │    Compose    │
│               │      │   Manager     │      │  Orchestrator │
│  - Clone      │      │               │      │               │
│  - Fetch      │      │  - Create     │      │  - Validate   │
│  - Auth check │      │  - List       │      │  - Up --wait  │
│               │      │  - Prune      │      │  - Down       │
└───────────────┘      └───────────────┘      └───────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                ▼
                    ┌───────────────────────┐
                    │     State Store       │
                    │      (SQLite)         │
                    │                       │
                    │  - Projects           │
                    │  - Deployments        │
                    │  - Deploy history     │
                    │  - Operation logs     │
                    └───────────────────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │    TUI Dashboard      │
                    │    (Bubble Tea)       │
                    │                       │
                    │  - Project list       │
                    │  - Health status      │
                    │  - Deploy history     │
                    └───────────────────────┘
```

### Data Flow: Deploy from URL

```
1. User: otterstack project add --name myapp --repo https://github.com/user/myapp.git
   │
   ▼
2. Repo Manager: Clone to /var/lib/otterstack/repos/myapp/
   │
   ▼
3. Validate: Check compose.yaml exists at root
   │
   ▼
4. State: Record project (type=remote, url, path)

───────────────────────────────────────────────────────────────────

5. User: otterstack deploy myapp --ref v1.0.0
   │
   ▼
6. Lock: Acquire /var/lib/otterstack/locks/myapp.lock
   │
   ▼
7. Repo Manager: git fetch --prune --tags
   │
   ▼
8. Resolve: git rev-parse v1.0.0 → abc123def456
   │
   ▼
9. Worktree Manager: git worktree add /var/lib/otterstack/worktrees/myapp/abc123d (7-char SHA)
   │
   ▼
10. Compose: docker compose -f compose.yaml config (validate)
    │
    ▼
11. Compose: docker compose --project-name otterstack-myapp-abc123d up -d --wait --wait-timeout 300
    NOTE: Always use --project-name to prevent container/network name collisions
    │
    ▼
12. State: Record deployment (sha=abc123, status=active, deployed_at=now)
    │
    ▼
13. Prune: Remove old worktrees (keep last 3)
    │
    ▼
14. Lock: Release
```

### Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | **Go 1.22+** | Single binary, Docker ecosystem alignment, mature CLI tooling |
| CLI Framework | **Cobra** | Used by Docker, Kubernetes, GitHub CLI; excellent subcommand support |
| Configuration | **Viper** | Pairs with Cobra, supports YAML/ENV/flags |
| TUI | **Bubble Tea** | Elm architecture, rich ecosystem (Bubbles, Lip Gloss) |
| Git (basic ops) | **go-git** | Clone, fetch, status without external dependency |
| Git (worktrees) | **git CLI** | go-git doesn't support worktrees; shell out required |
| Docker | **docker compose CLI** | Shell out to CLI - simpler than SDK, same behavior |
| State | **SQLite** | ACID transactions, single file, no server |
| Locking | **File locks** | Cross-platform via `github.com/gofrs/flock` with stale lock detection |

### Directory Layout

```
/var/lib/otterstack/
├── otterstack.db           # SQLite state database
├── repos/                  # Cloned repositories (URL-backed projects)
│   ├── myapp/
│   │   └── .git/           # Bare-ish clone
│   └── another-app/
├── worktrees/              # Deployment worktrees (7-char short SHA)
│   ├── myapp/
│   │   ├── abc123d/        # v1.0.0
│   │   ├── def456g/        # v1.0.1 (current)
│   │   └── ghi789j/        # v1.0.2 (next)
│   └── another-app/
├── locks/                  # Per-project operation locks
│   └── myapp.lock
└── logs/                   # Operation logs
    └── myapp/
        └── 2026-01-08T10-30-00-deploy.log
```

### State Machine

```
Project Lifecycle:
  UNCONFIGURED ──add──► CLONING ──success──► READY
                            │
                         failure
                            ▼
                       CLONE_FAILED ──retry──► CLONING

Deployment Lifecycle:
  READY ──deploy──► FETCHING ──► RESOLVING ──► CREATING_WORKTREE
                        │             │               │
                     failure       failure         failure
                        ▼             ▼               ▼
                   FETCH_FAILED  RESOLVE_FAILED  WORKTREE_FAILED

  CREATING_WORKTREE ──► VALIDATING ──► STARTING ──► WAITING_HEALTHY
                              │            │              │
                           failure      failure        timeout
                              ▼            ▼              ▼
                        INVALID_COMPOSE  START_FAILED  UNHEALTHY

  WAITING_HEALTHY ──healthy──► DEPLOYED
                       │
                    rollback
                       ▼
                   ROLLING_BACK ──► ROLLED_BACK
                       │
                    failure
                       ▼
                   ROLLBACK_FAILED
```

---

## Implementation Phases

### Phase 1: Foundation (Core CLI + Single Project)

**Goal:** Deploy a single project from local Git repo via CLI

**Deliverables:**
- [ ] `cmd/root.go` - Cobra root command with global flags + signal handling (SIGINT/SIGTERM)
- [ ] `cmd/project/add.go` - Add project from local path
- [ ] `cmd/project/list.go` - List registered projects
- [ ] `cmd/project/remove.go` - Remove project (with confirmation)
- [ ] `cmd/deploy.go` - Deploy project to specific ref
- [ ] `cmd/status.go` - Show project status
- [ ] `cmd/cleanup.go` - Reconcile state: fix interrupted deploys, orphaned worktrees, stale locks
- [ ] `internal/git/worktree.go` - Worktree create/list/remove (shells to git)
- [ ] `internal/compose/orchestrator.go` - Validate and up/down (shells to docker compose CLI)
- [ ] `internal/state/sqlite.go` - SQLite schema with WAL mode and basic CRUD
- [ ] `internal/state/migrations.go` - Embedded SQL migrations with version tracking
- [ ] `internal/lock/file.go` - File-based locking with PID tracking for stale detection
- [ ] `internal/validate/input.go` - Input validation for project names, URLs

**Files to create:**
```
otterstack/
├── main.go
├── cmd/
│   ├── root.go           # Includes signal handling
│   ├── project/
│   │   ├── project.go    # Parent command
│   │   ├── add.go
│   │   ├── list.go
│   │   └── remove.go
│   ├── deploy.go
│   ├── status.go
│   └── cleanup.go        # Reconcile state
├── internal/
│   ├── git/
│   │   └── worktree.go
│   ├── compose/
│   │   └── orchestrator.go
│   ├── state/
│   │   ├── sqlite.go
│   │   ├── migrations.go
│   │   └── migrations/
│   │       └── 001_initial.sql
│   ├── lock/
│   │   └── file.go       # With PID-based stale detection
│   ├── validate/
│   │   └── input.go
│   └── errors/
│       └── errors.go     # Sentinel errors
└── go.mod
```

**Acceptance criteria:**
- [ ] `otterstack project add --name myapp --repo /srv/myapp` registers project
- [ ] `otterstack project list` shows registered projects
- [ ] `otterstack deploy myapp --ref v1.0.0` creates worktree, runs compose up with `--project-name`
- [ ] Deployment fails cleanly if compose.yaml missing
- [ ] Deployment fails cleanly if ref doesn't exist
- [ ] Concurrent deploys to same project are blocked by lock
- [ ] Stale locks (dead PID) are automatically cleaned up on acquire
- [ ] SIGINT/SIGTERM during deploy marks deployment as `interrupted` and releases lock
- [ ] `otterstack cleanup` reconciles orphaned worktrees and interrupted deployments
- [ ] Project names validated: `^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]?$` (no path traversal)

---

### Phase 2: Git URL Cloning

**Goal:** Support GitHub/GitLab URLs as repo source

**Deliverables:**
- [ ] `internal/git/repo.go` - Clone, fetch, detect URL vs path
- [ ] `internal/git/auth.go` - SSH agent detection, HTTPS credential helper passthrough
- [ ] Update `cmd/project/add.go` to handle URLs
- [ ] Atomic clone with temp directory and rename
- [ ] Pre-flight auth check before clone

**New/updated files:**
```
internal/git/
├── repo.go       # Clone, fetch operations
├── auth.go       # Auth detection and validation
└── worktree.go   # (from Phase 1)
```

**Acceptance criteria:**
- [ ] `otterstack project add --name app --repo https://github.com/user/app.git` clones to managed storage
- [ ] `otterstack project add --name app --repo git@github.com:user/app.git` works with SSH
- [ ] Clone failure leaves no orphaned directories
- [ ] Clear error message if SSH key missing or HTTPS auth fails
- [ ] `git fetch --prune --tags` runs before each deploy for URL-backed projects

---

### Phase 3: Rollback & History

**Goal:** Track deployment history and enable instant rollback

**Deliverables:**
- [ ] `cmd/rollback.go` - Rollback to previous deployment
- [ ] `cmd/history.go` - Show deployment history
- [ ] Worktree retention policy (keep last N, configurable)
- [ ] Auto-prune old worktrees after successful deploy

**New files:**
```
cmd/
├── rollback.go
└── history.go
```

**State schema additions:**
```sql
CREATE TABLE deployments (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    git_sha TEXT NOT NULL,
    git_ref TEXT,  -- original ref (tag/branch) if any
    status TEXT CHECK(status IN ('deploying', 'active', 'inactive', 'failed', 'rolled_back', 'interrupted')),
    started_at DATETIME,
    finished_at DATETIME,
    error_message TEXT,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

-- CRITICAL: Indexes for query performance at scale
CREATE INDEX idx_deployments_project_status ON deployments(project_id, status);
CREATE INDEX idx_deployments_project_started ON deployments(project_id, started_at DESC);
CREATE INDEX idx_deployments_git_sha ON deployments(project_id, git_sha);
```

**Acceptance criteria:**
- [ ] `otterstack history myapp` shows last 20 deployments with SHA, ref, status, timestamp
- [ ] `otterstack rollback myapp` redeploys the previous active deployment
- [ ] `otterstack rollback myapp --to abc123` redeploys specific SHA
- [ ] Rollback reuses existing worktree if available
- [ ] Old worktrees pruned to configured limit (default: 3)

---

### Phase 4: TUI Dashboard

**Goal:** Read-only terminal UI for monitoring all projects

**Deliverables:**
- [ ] `cmd/monitor.go` - Launch TUI
- [ ] `internal/tui/model.go` - Bubble Tea model
- [ ] `internal/tui/views/` - Project list, detail, logs views
- [ ] `internal/tui/styles.go` - Lip Gloss styling
- [ ] Real-time health check via Docker API

**New files:**
```
cmd/monitor.go
internal/tui/
├── model.go
├── update.go
├── views/
│   ├── project_list.go
│   ├── project_detail.go
│   └── help.go
└── styles.go
```

**TUI Layout:**
```
┌─ OtterStack Monitor ─────────────────────────────────────────────┐
│                                                                  │
│  PROJECT          REPO                    REF        STATUS      │
│  ─────────────────────────────────────────────────────────────   │
│▸ myapp            github.com/user/myapp   v1.0.2     ● HEALTHY   │
│  api-service      local:/srv/api          main       ● HEALTHY   │
│  background-jobs  github.com/co/jobs      abc123     ○ STARTING  │
│  legacy-app       local:/opt/legacy       v2.3.1     ✗ UNHEALTHY │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│ [↑↓] Navigate  [Enter] Details  [r] Refresh  [q] Quit           │
└──────────────────────────────────────────────────────────────────┘
```

**Acceptance criteria:**
- [ ] `otterstack monitor` launches TUI
- [ ] Projects displayed with name, repo origin, current ref, health status
- [ ] Health status updates every 5 seconds (configurable)
- [ ] Enter on project shows detail view with containers, logs, history
- [ ] `q` exits cleanly
- [ ] Works over SSH

---

### Phase 5: Notifications & Watch Mode

**Goal:** Proactive monitoring and alerting

**Deliverables:**
- [ ] `cmd/watch.go` - Continuous health monitoring
- [ ] `internal/notify/` - Notification backends (webhook, Discord, Slack, email)
- [ ] Configuration for notification channels per project
- [ ] Unhealthy container detection and alerting

**New files:**
```
cmd/watch.go
internal/notify/
├── notifier.go    # Interface
├── webhook.go
├── discord.go
├── slack.go
└── email.go
```

**Acceptance criteria:**
- [ ] `otterstack watch` runs as daemon, monitoring all projects
- [ ] Notification sent when container becomes unhealthy
- [ ] Notification sent on deploy success/failure
- [ ] Configurable notification channels per project
- [ ] Rate limiting to prevent notification spam

---

## Alternative Approaches Considered

### Language: Rust vs Go

**Rust (Clap + Ratatui):**
- Pros: Better performance, memory safety, smaller binary
- Cons: Slower development, steeper learning curve, Docker SDK is Go-native

**Decision:** Go - aligns with Docker ecosystem, faster iteration, Bubble Tea is more productive than Ratatui for this use case.

### Git: go-git vs shell out

**Pure go-git:**
- Pros: No external dependency, cross-platform
- Cons: Doesn't support worktrees (critical feature)

**Pure shell:**
- Pros: Full feature support
- Cons: Parsing text output, external dependency

**Decision:** Hybrid - go-git for clone/fetch, shell for worktrees.

### State: YAML vs SQLite

**YAML files:**
- Pros: Human-readable, easy debugging
- Cons: No ACID, concurrent access issues, scales poorly

**SQLite:**
- Pros: ACID transactions, concurrent reads, queryable
- Cons: Binary format, requires migration tooling

**Decision:** SQLite - critical for safe concurrent operations and history queries.

### Proxy integration

**Include proxy (like Kamal):**
- Pros: Zero-downtime deployments, auto-HTTPS
- Cons: Scope creep, complexity, opinionated

**No proxy:**
- Pros: Pure Compose, user brings their own proxy
- Cons: No automatic zero-downtime

**Decision:** No proxy - keep scope minimal. Users configure Traefik/Caddy in their Compose files.

---

## Acceptance Criteria

### Functional Requirements

- [ ] Register project from local Git path
- [ ] Register project from GitHub/GitLab HTTPS URL
- [ ] Register project from Git SSH URL
- [ ] Clone URL-backed repos to managed storage
- [ ] Fetch updates before deploy for URL-backed repos
- [ ] Deploy to specific Git ref (tag, branch, SHA)
- [ ] Resolve ref to SHA deterministically
- [ ] Create worktree per deployment
- [ ] Validate compose.yaml before deployment
- [ ] Run `docker compose up -d --wait` with configurable timeout
- [ ] Record deployment in state database
- [ ] Rollback to previous deployment
- [ ] Rollback to specific SHA
- [ ] Show deployment history
- [ ] Show project status (current deployment, health)
- [ ] List all projects
- [ ] Remove project (with optional container cleanup)
- [ ] TUI dashboard showing all projects
- [ ] Health status per project in TUI
- [ ] Notification on deploy success/failure
- [ ] Notification on container unhealthy

### Non-Functional Requirements

- [ ] **Timeouts:**
  - Clone: 10 minutes (configurable)
  - Fetch: 2 minutes (configurable)
  - Deploy (total): 15 minutes (configurable)
  - Health check wait: 5 minutes (configurable)
- [ ] **Concurrency:**
  - File lock prevents concurrent deploys to same project
  - Global semaphore limits parallel deploys (default: 2)
- [ ] **Retention:**
  - Worktrees: keep last 3 per project (configurable)
  - History: keep last 50 deployments per project
  - Logs: keep last 30 days
- [ ] **Security:**
  - No credentials stored on disk
  - URLs sanitized in logs (strip userinfo)
  - Env vars matching `*_KEY`, `*_SECRET`, `*_PASSWORD`, `*_TOKEN` masked in logs
  - Repo directory permissions: 0750
  - State file permissions: 0600
  - Git hooks disabled during clone/fetch
- [ ] **Performance:**
  - Full clone (NOT shallow - shallow clones break worktree operations on older refs)
  - TUI refresh: 5 seconds default, configurable

### Quality Gates

- [ ] Unit test coverage > 70% for internal packages
- [ ] Integration tests for Git operations (local test repo)
- [ ] Integration tests for Compose operations (Docker-in-Docker)
- [ ] E2E test: full deploy cycle from URL
- [ ] Linting: golangci-lint with standard config
- [ ] No data races (`go test -race`)
- [ ] Documentation: README, CLI help

### Testing Conventions

- [ ] **Table-driven tests** for all logic functions
- [ ] **testify/assert and testify/require** for assertions
- [ ] **Interfaces for external dependencies** (Git, Docker) to enable mocking
- [ ] **`testdata/` directories** for fixture repos and compose files
- [ ] **`context.Context` as first parameter** on all internal functions for cancellation
- [ ] **No sleeps in tests** - use channels/waitgroups for synchronization

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Time from URL to running | < 5 minutes (excluding image pulls) |
| Rollback time | < 30 seconds |
| TUI startup time | < 500ms |
| Binary size | < 30MB |
| Memory usage (idle) | < 50MB |
| Memory usage (TUI) | < 100MB |

---

## Dependencies & Prerequisites

### System Requirements

- Linux (primary target), macOS (secondary)
- Docker Engine 24.0+ with Compose plugin
- Git 2.30+ (for worktree improvements)
- SQLite 3.35+ (for RETURNING clause)

### Go Dependencies

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/bubbles v0.18.0
    github.com/charmbracelet/lipgloss v0.9.0
    github.com/go-git/go-git/v5 v5.11.0
    github.com/mattn/go-sqlite3 v1.14.22
    github.com/gofrs/flock v0.8.1           // Cross-platform file locking with stale detection
    github.com/stretchr/testify v1.9.0      // Testing assertions
    golang.org/x/sync v0.6.0                // Semaphore for concurrency control
)
```

**Note:** Docker Compose operations shell out to `docker compose` CLI rather than embedding the SDK. This is simpler, more maintainable, and matches common deployment tool patterns.

### External Dependencies

- `git` CLI (for worktree operations)
- `docker compose` CLI (for compose operations - shell out, not SDK)

---

## Risk Analysis & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| go-git limitations discovered late | High | Medium | Spike worktree ops early in Phase 1; have shell fallback ready |
| SQLite concurrent write issues | Medium | Medium | Use WAL mode (`PRAGMA journal_mode=WAL`); single writer pattern |
| Git auth complexity | Medium | High | Clear docs; test SSH agent, HTTPS creds; helpful error messages |
| Compose file variations | Medium | High | Support compose.yaml/yml, docker-compose.yaml/yml; configurable per project |
| Disk space exhaustion | High | Medium | Pre-flight check; configurable worktree retention; alert threshold |
| User edits managed repo | Low | Medium | Detect dirty state; warn; `--force` flag to override |
| Port conflicts on deploy | Medium | Medium | Pre-check ports; clear error with conflicting service |
| Interrupted deploy leaves bad state | High | Medium | Signal handling marks as `interrupted`; `cleanup` command reconciles |
| Stale locks block operations | Medium | Medium | Lock files include PID; stale lock detection on acquire |
| Container name collisions | High | High | Always use `--project-name otterstack-{project}-{sha_short}` |

---

## Future Considerations

**Potential v2.0 features (not in scope for v1.0):**

1. **Remote API**: HTTP/gRPC API for integrations (CI/CD, custom dashboards)
2. **Multi-host**: Deploy same project to multiple servers
3. **Secrets management**: Integration with Vault, SOPS, or doppler
4. **Proxy integration**: Optional Caddy/Traefik sidecar with auto-HTTPS
5. **Build support**: Build images from Dockerfile before deploy
6. **Registry push**: Push built images to registry for multi-host
7. **Scheduled deployments**: Cron-like deploy on schedule
8. **Canary deployments**: Gradual traffic shift between versions
9. **Resource quotas**: Limit CPU/memory per project
10. **Audit log**: Immutable log of all operations for compliance

---

## Documentation Plan

| Document | Purpose | When |
|----------|---------|------|
| README.md | Project overview, quick start | Phase 1 |
| INSTALL.md | Installation methods (binary, go install, Docker) | Phase 1 |
| docs/cli-reference.md | Generated from Cobra help | Phase 1 |
| docs/configuration.md | Config file format, environment variables | Phase 2 |
| docs/security.md | Auth setup, permissions, best practices | Phase 2 |
| docs/troubleshooting.md | Common errors and solutions | Phase 3 |
| man pages | Generated via cobra-doc | Phase 4 |
| CHANGELOG.md | Version history | Ongoing |

---

## References & Research

### Internal References

- PRD: User-provided in this conversation
- CLAUDE.md: `C:\Users\jayte\Documents\dev\OtterStack\CLAUDE.md` (lines 1-79)

### External References

**Similar Tools:**
- [Kamal Deploy Documentation](https://kamal-deploy.org/)
- [Kamal Proxy GitHub](https://github.com/basecamp/kamal-proxy)
- [Coolify Docker Compose Documentation](https://coolify.io/docs/knowledge-base/docker/compose)
- [Dokku Documentation](https://dokku.com/docs/)

**Git Worktree:**
- [Git Worktree Best Practices](https://gist.github.com/ChristopherA/4643b2f5e024578606b9cd5d2e6815cc)
- [Git Worktree Documentation](https://git-scm.com/docs/git-worktree)

**Docker Compose:**
- [Docker Compose Deploy Specification](https://docs.docker.com/reference/compose-file/deploy/)
- [Docker Compose Watch](https://docs.docker.com/compose/how-tos/file-watch/)
- [Health Checks Best Practices](https://www.tvaidyan.com/2025/02/13/health-checks-in-docker-compose-a-practical-guide/)

**CLI & TUI:**
- [Cobra Documentation](https://cobra.dev/)
- [Bubble Tea GitHub](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss Styling](https://github.com/charmbracelet/lipgloss)

**State Management:**
- [SQLite in Production](https://fractaledmind.github.io/2023/12/23/rubyconftw/)
- [Docker Compose SDK](https://docs.docker.com/compose/compose-sdk/)

**Go Libraries:**
- [go-git Package](https://pkg.go.dev/github.com/go-git/go-git/v5)
- [go-git-cmd-wrapper](https://pkg.go.dev/github.com/ldez/go-git-cmd-wrapper/worktree)

---

## ERD: Data Model

```mermaid
erDiagram
    PROJECT ||--o{ DEPLOYMENT : has
    PROJECT {
        text id PK
        text name UK
        text repo_type "local|remote"
        text repo_url "nullable, for remote"
        text repo_path "local path or managed clone path"
        text compose_file "default: compose.yaml"
        int worktree_retention "default: 3"
        text status "unconfigured|cloning|ready|clone_failed"
        datetime created_at
        datetime updated_at
    }

    DEPLOYMENT {
        text id PK
        text project_id FK
        text git_sha
        text git_ref "nullable, original tag/branch"
        text worktree_path
        text status "deploying|active|inactive|failed|rolled_back"
        text error_message "nullable"
        datetime started_at
        datetime finished_at
    }

    OPERATION_LOG {
        text id PK
        text project_id FK
        text deployment_id FK "nullable"
        text operation "clone|fetch|deploy|rollback|remove"
        text status "running|success|failed"
        text log_path
        datetime started_at
        datetime finished_at
    }

    NOTIFICATION_CONFIG ||--|| PROJECT : configures
    NOTIFICATION_CONFIG {
        text id PK
        text project_id FK UK
        boolean on_deploy_success
        boolean on_deploy_failure
        boolean on_unhealthy
        text webhook_url "nullable"
        text slack_webhook "nullable"
        text discord_webhook "nullable"
        text email "nullable"
    }
```

---

## CLI Command Reference (Planned)

```bash
# Project management
otterstack project add --name <name> --repo <path|url> [--compose-file <file>]
otterstack project list [--json]
otterstack project show <name>
otterstack project remove <name> [--force] [--cleanup=containers|all|none]

# Deployment
otterstack deploy <name> --ref <ref> [--dry-run] [--no-wait] [--timeout <duration>]
otterstack rollback <name> [--to <sha>]
otterstack history <name> [--limit <n>] [--json]
otterstack status [<name>] [--json]

# Monitoring
otterstack monitor              # Launch TUI
otterstack watch [--daemon]     # Watch mode with notifications

# Maintenance
otterstack cleanup [<name>]     # Reconcile state: fix interrupted deploys, orphaned worktrees, stale locks
otterstack doctor               # Check system prerequisites (Docker, Git, disk space, permissions)
otterstack prune <name>         # Remove old worktrees
otterstack logs <name> [--deployment <id>]
otterstack validate <name>      # Validate compose file without deploying
```

## Input Validation Rules

```go
// Project name: alphanumeric + hyphen, 1-64 chars, no path traversal
var validProjectName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]?$`)

// Compose file precedence (highest to lowest):
// 1. --compose-file flag
// 2. compose.yaml
// 3. compose.yml
// 4. docker-compose.yaml
// 5. docker-compose.yml
```

---

## Review Feedback Addressed

This plan incorporates critical feedback from the review process:

| Issue | Resolution |
|-------|------------|
| Shallow clone breaks worktrees | Removed shallow clone - full clone required for worktree ops |
| Missing SQLite indexes | Added indexes for project_status, project_started, git_sha |
| Container name collisions | Added mandatory `--project-name otterstack-{project}-{sha}` |
| No graceful shutdown | Added signal handling, `interrupted` status, `cleanup` command |
| Stale lock recovery | Lock files include PID; stale detection on acquire |
| No migration strategy | Added `internal/state/migrations.go` with embedded SQL |
| Missing input validation | Added `internal/validate/input.go` with project name rules |
| Compose SDK complexity | Switched to shell-out to `docker compose` CLI |
| Full SHA paths verbose | Changed to 7-char short SHA in worktree paths |
| Cross-platform locking | Switched from `flock` to `github.com/gofrs/flock` |

---

**Plan ready for implementation.**
