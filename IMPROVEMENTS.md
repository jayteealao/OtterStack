# OtterStack Improvements

Based on extensive deployment experience with the Aperture project, this document outlines prioritized improvements for OtterStack organized by impact and complexity.

## Table of Contents

- [Top Priority Improvements](#top-priority-improvements)
- [Medium Priority Improvements](#medium-priority-improvements)
- [Lower Priority Improvements](#lower-priority-improvements)
- [Documentation Improvements](#documentation-improvements)
- [Implementation Roadmap](#implementation-roadmap)

---

## Top Priority Improvements

These improvements have high impact and low-medium complexity, making them ideal candidates for implementation.

### 1. Better Environment Variable Validation

**Problem**: Generic "variable is not set" warnings from Docker Compose are cryptic and don't help users fix the issue quickly.

**Solution**: Parse compose file before deployment and show an interactive checklist:
```
Missing environment variables:
[ ] DATABASE_URL (required by service: web)
[ ] SECRET_KEY (required by service: web)
[✓] LOG_LEVEL (optional, defaults to: INFO)

Run: otterstack env set <project> <key> <value>
```

**Impact**: Reduces deployment failures by ~40% by catching missing env vars early with clear fix instructions.

**Files to Modify**:
- `internal/compose/orchestrator.go` (line 152 - add env var parser before ValidateWithEnv)
- `internal/compose/env_parser.go` (new file - parse ${VAR} patterns from compose YAML)

**Implementation Steps**:
1. Create `EnvVarParser` that extracts all `${VAR}` references from compose file
2. Compare extracted vars against stored env vars in database
3. Categorize vars as required (no default) vs optional (has default like `${VAR:-default}`)
4. Display formatted checklist before validation
5. Abort deployment if required vars are missing

**Complexity**: Medium (requires YAML parsing and formatting logic)

---

### 2. Automatic Environment Variable Discovery

**Problem**: Manually running `otterstack env set` for each variable is tedious for new projects.

**Solution**: Add `otterstack env scan <project>` command that:
- Parses compose file for `${VAR}` patterns
- Prompts interactively for each missing variable
- Detects variable types (URL, boolean, number, path)
- Generates `.env.example` file automatically

**Impact**: Saves 5-10 minutes per new project setup, reduces human error.

**Files to Create/Modify**:
- `cmd/env.go` (add `scan` subcommand)
- `internal/env/scanner.go` (new file - variable discovery and interactive prompts)
- `internal/env/detector.go` (new file - detect variable types from names/patterns)

**Implementation Steps**:
1. Add `scan` subcommand to `cmd/env.go`
2. Parse compose file to extract all `${VAR}` references
3. Check which vars are already set in database
4. For each unset var:
   - Detect type from variable name (e.g., `*_URL` → URL, `*_PORT` → number)
   - Prompt user with appropriate validation
   - Save to database
5. Generate `.env.example` file with all variables and detected types

**Complexity**: Medium (interactive prompts + type detection logic)

---

### 3. Health Check Debugging Assistant

**Problem**: "health check timeout" error doesn't explain WHY the container failed health checks.

**Solution**: When health check fails, automatically:
- Show last 20 lines of container logs
- Show container status (restarting, exited, running but unhealthy)
- Test health check command manually and display result
- Suggest fixes based on common error patterns

**Impact**: Reduces debugging time by ~50%, speeds up time-to-resolution for deployment failures.

**Files to Modify**:
- `internal/traefik/health.go` (lines 50-80 - WaitForHealthy function)
- `internal/compose/orchestrator.go` (add health check testing method)

**Implementation Steps**:
1. Modify `WaitForHealthy()` to capture failure details
2. On timeout/failure:
   - Get container status via `docker inspect`
   - Fetch last 20 lines of logs via `docker logs`
   - Extract health check command from container config
   - Execute health check manually via `docker exec`
   - Display all results in structured format
3. Add pattern matching for common errors:
   - Connection refused → check port/endpoint
   - 404 → wrong health check path
   - Timeout → increase start_period

**Complexity**: Medium (requires container inspection and error pattern matching)

---

### 4. Pre-Deployment Validation Hooks

**Problem**: No way to run custom validation or tests before deployment starts.

**Solution**: Support `.otterstack/hooks/pre-deploy.sh` script:
```bash
#!/bin/bash
set -e  # Exit on first error

# Run tests
npm run test

# Validate compose file
docker compose config --quiet

# Check environment
if [ -z "$REQUIRED_VAR" ]; then
  echo "ERROR: REQUIRED_VAR not set"
  exit 1
fi
```

If hook exits with non-zero code, abort deployment.

**Impact**: Catches errors earlier in CI/CD pipeline, prevents broken deploys.

**Files to Modify**:
- `internal/orchestrator/deployer.go` (line 50 - before git operations)
- `internal/hooks/runner.go` (new file - execute scripts safely)

**Implementation Steps**:
1. Check for `.otterstack/hooks/pre-deploy.sh` in worktree
2. If exists and executable, run it with timeout (e.g., 5 minutes)
3. Capture stdout/stderr
4. If exit code non-zero:
   - Display script output
   - Abort deployment with clear error
5. If exit code zero, continue deployment

**Complexity**: Low (straightforward script execution)

---

### 5. Streaming Container Logs During Deployment

**Problem**: Logs only shown on failure (last 50 lines). During health check wait, no visibility into what's happening.

**Solution**: Stream container logs in real-time during the health check phase.

**Impact**: Better visibility, easier debugging, users can see application startup in real-time.

**Files to Modify**:
- `internal/orchestrator/deployer.go` (lines 199-220 - health check section)
- `internal/compose/orchestrator.go` (add log streaming method)

**Implementation Steps**:
1. Add `StreamLogs(ctx context.Context, projectName string, follow bool)` method to compose.Manager
2. In `deployer.Deploy()`, spawn goroutine before `WaitForHealthy()`:
   ```go
   go func() {
       composeMgr.StreamLogs(ctx, composeProjectName, true)
   }()
   ```
3. Stream logs to verbose output callback
4. Cancel goroutine when health check completes (success or failure)

**Complexity**: Low (goroutine + docker logs --follow)

---

## Medium Priority Improvements

These improvements have medium impact and low-medium complexity.

### 6. Compose Compatibility Validator

**Problem**: Deployment fails with OtterStack-incompatible compose files (container names, env_file usage, etc).

**Solution**: Add `otterstack project validate <name>` command that checks:
- ❌ No `container_name:` directives
- ✅ Uses `environment:` section (not `env_file:`)
- ❌ No static Traefik priority labels
- ✅ Has health checks defined
- ✅ Syntax validation passes

Output pass/fail with specific line numbers and fixes.

**Impact**: Prevents ~30% of deployment failures by catching incompatibilities early.

**Files to Create**:
- `cmd/validate.go` (new command)
- `internal/validate/compose.go` (new file - compose file validation)

**Complexity**: Low (YAML parsing + pattern matching)

---

### 7. Rollback Command

**Problem**: Manual `deploy --ref <old-sha>` to rollback is slow (rebuilds containers).

**Solution**: Add `otterstack rollback <project>` that:
- Gets previous active deployment from database
- Swaps Traefik priorities instantly (new → 0, old → current timestamp)
- No rebuild/restart needed (old containers still running)

**Impact**: Faster incident recovery (30 seconds vs 5 minutes).

**Files to Create/Modify**:
- `cmd/rollback.go` (new command)
- Use existing Traefik priority logic from `internal/traefik/override.go`

**Complexity**: Low (reuses existing logic)

---

### 8. Interactive Deployment Mode

**Problem**: All-or-nothing deployment can cause accidental bad deploys.

**Solution**: Add `--interactive` flag that:
- Shows deployment diff before starting (git diff, image changes, env changes)
- Pauses before traffic switch
- Asks "Continue? [y/n]"

**Impact**: Prevents accidental bad deploys, gives users a safety net.

**Files to Modify**:
- `cmd/deploy.go` (add --interactive flag)
- `internal/orchestrator/deployer.go` (line 224 - pause before Traefik labels)
- `internal/diff/generator.go` (new file - generate deployment diff)

**Complexity**: Medium (diff generation + TUI confirmation prompt)

---

### 9. Deployment Diff Preview

**Problem**: No visibility into what changed between current and new deployment.

**Solution**: Before deploying, automatically show:
- Git commit diff summary (files changed, additions/deletions)
- Container image changes (old tag → new tag)
- Environment variable changes (added/removed/modified)

**Impact**: Better change awareness, easier troubleshooting if something breaks.

**Files to Modify**:
- `internal/orchestrator/deployer.go` (before line 100)
- `internal/diff/generator.go` (new file - generate diffs)

**Complexity**: Medium (git diff + image tag comparison)

---

### 10. Post-Deployment Smoke Tests

**Problem**: Deployment "succeeds" but application is broken (returns 500, database not connected, etc).

**Solution**: Support `.otterstack/hooks/post-deploy.sh`:
```bash
#!/bin/bash
curl -f https://myapp.com/health || exit 1
curl -f https://myapp.com/api/version || exit 1
```

If tests fail, automatically rollback to previous deployment.

**Impact**: Catch runtime issues immediately, automatic rollback prevents broken production.

**Files to Modify**:
- `internal/orchestrator/deployer.go` (after line 260)
- `internal/hooks/runner.go` (reuse from pre-deploy hooks)
- Add rollback trigger on test failure

**Complexity**: Low (reuses hook runner + rollback logic)

---

## Lower Priority Improvements

Good-to-have features with lower immediate impact.

### 11. Deployment Progress Bar
**Description**: Visual progress indicator using bubbletea TUI
**Impact**: Better UX
**Complexity**: Low

### 12. Container Resource Metrics
**Description**: Show CPU/memory/network usage via `docker stats`
**Impact**: Operational visibility
**Complexity**: Low

### 13. Deployment Webhooks
**Description**: Rich Slack/Discord notifications with logs, commit info
**Impact**: Team awareness
**Complexity**: Medium

### 14. .env.example Generator
**Description**: Auto-generate from compose file scan
**Impact**: Documentation
**Complexity**: Low (reuses env scanner from #2)

### 15. Automatic Compose Scaffolding
**Description**: `otterstack scaffold <stack-type>` generates compose from templates
**Impact**: Faster project setup
**Complexity**: Medium

### 16. Multi-Environment Support
**Description**: Separate staging/production with `--env` flag
**Impact**: Enterprise feature
**Complexity**: Medium

### 17. Deployment Tags
**Description**: Custom annotations (version, release notes, deployer)
**Impact**: Audit trail
**Complexity**: Easy

### 18. Color-Coded Status
**Description**: Green/yellow/red status indicators with emoji
**Impact**: Visual clarity
**Complexity**: Easy

### 19. Estimated Deployment Time
**Description**: Show estimate based on historical data
**Impact**: User expectations
**Complexity**: Easy

---

## Documentation Improvements

Critical missing documentation that should be added.

### 1. Compose File Requirements Reference

**File**: `docs/compose-requirements.md`

**Content**:
- Complete list of OtterStack compose file requirements
- Why each requirement exists
- Examples of correct and incorrect configurations
- Common pitfalls and fixes

**Why Needed**: Requirements currently scattered across README and TROUBLESHOOTING.md

---

### 2. Environment Variable Management Guide

**File**: `docs/environment-variables.md`

**Content**:
- How environment variables flow through OtterStack
- `--env-file` vs `environment:` section
- Secrets management best practices
- Variable precedence rules
- Debugging missing variables

**Why Needed**: Common source of confusion for new users

---

### 3. Zero-Downtime Deployment Deep Dive

**File**: `docs/zero-downtime.md`

**Content**:
- Technical explanation of Traefik priority mechanism
- Health check process and timing
- Rollback behavior on failures
- What happens during each deployment stage
- Diagram of traffic flow during deployment

**Why Needed**: Users don't understand how traffic switching works

---

### 4. Common Patterns Cookbook

**Directory**: `docs/cookbooks/`

**Files**:
- `node-prisma-postgres.md` - Node.js + Prisma + PostgreSQL
- `fastapi-redis.md` - Python FastAPI + Redis
- `multi-service.md` - Multiple interdependent services
- `frontend-backend.md` - React SPA + API backend

**Why Needed**: Best practices not documented, users don't know how to structure projects

---

## Implementation Roadmap

### Phase 1: Quick Wins (1-2 weeks)
Implement improvements with lowest complexity and highest impact:

1. ✅ Pre-Deployment Hooks (#4) - 2 days
2. ✅ Streaming Logs (#5) - 1 day
3. ✅ Compose Validator (#6) - 3 days
4. ✅ Rollback Command (#7) - 2 days

**Impact**: Catches errors early, better visibility, faster recovery

### Phase 2: User Experience (2-3 weeks)
Improve deployment workflow and debugging:

1. ✅ Better Env Var Validation (#1) - 4 days
2. ✅ Env Var Discovery (#2) - 4 days
3. ✅ Health Check Debugging (#3) - 5 days

**Impact**: Reduces deployment failures and debugging time significantly

### Phase 3: Safety & Confidence (2 weeks)
Add safety features for production:

1. ✅ Post-Deployment Smoke Tests (#10) - 3 days
2. ✅ Interactive Deployment Mode (#8) - 4 days
3. ✅ Deployment Diff Preview (#9) - 4 days

**Impact**: Prevents bad deploys, automatic rollback on failures

### Phase 4: Documentation (1 week)
Complete documentation gaps:

1. ✅ Compose Requirements Reference - 1 day
2. ✅ Environment Variable Guide - 1 day
3. ✅ Zero-Downtime Deep Dive - 2 days
4. ✅ Cookbooks - 2 days

**Impact**: Reduces support burden, faster onboarding

### Phase 5: Polish (1-2 weeks)
Lower priority improvements as time permits:

- Progress bars
- Resource metrics
- Webhooks
- Scaffolding

---

## Priority Matrix

| Feature | Impact | Complexity | Priority | Estimated Effort |
|---------|--------|------------|----------|------------------|
| Better Env Var Validation (#1) | High | Medium | P0 | 4 days |
| Env Var Discovery (#2) | High | Medium | P0 | 4 days |
| Health Check Debugging (#3) | High | Medium | P0 | 5 days |
| Pre-Deployment Hooks (#4) | High | Low | P0 | 2 days |
| Streaming Logs (#5) | High | Low | P0 | 1 day |
| Compose Validator (#6) | Medium | Low | P1 | 3 days |
| Rollback Command (#7) | Medium | Low | P1 | 2 days |
| Interactive Mode (#8) | Medium | Medium | P1 | 4 days |
| Deployment Diff (#9) | Medium | Medium | P1 | 4 days |
| Post-Deploy Tests (#10) | Medium | Low | P1 | 3 days |
| Documentation | High | Low | P0 | 5 days |

**Total estimated effort for P0 items**: ~25 days
**Total estimated effort for P1 items**: ~16 days

---

## Success Metrics

After implementing top priority improvements, we expect to see:

- **40% reduction** in deployment failures due to missing env vars
- **50% reduction** in debugging time for health check failures
- **30% reduction** in deployment failures due to compose compatibility issues
- **80% faster** rollback time (30s vs 5min)
- **90% reduction** in support questions about environment variables (with better docs)

---

## Contributing

To implement these improvements:

1. Create a GitHub issue for the improvement
2. Reference this document in the issue description
3. Include implementation steps from the relevant section
4. Link to specific files and line numbers
5. Add "good first issue" label for low-complexity items

Example issue title: "[Feature] Better Environment Variable Validation (#1)"

---

## Feedback

This improvements list is based on real deployment experience with Aperture. If you have additional suggestions or want to prioritize differently, please open an issue or discussion.

**Lessons Learned from Aperture Deployment:**
- Environment variables are the #1 source of deployment failures
- Health check failures are cryptic and hard to debug
- Compose file incompatibilities cause frustration
- Real-time visibility would have saved hours of debugging
- Documentation would have prevented many issues upfront

Let's make OtterStack even better! 🦦
