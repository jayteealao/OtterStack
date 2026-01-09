---
name: update-deps
description: Safely update dependencies with compatibility checks and testing
argument-hint: "[optional: specific package or 'all']"
hooks:
  PreToolUse:
    - matcher: Bash(npm install*)
      hook: |
        echo "[update-deps] Creating lockfile backup..."
        cp package-lock.json package-lock.json.bak 2>/dev/null || true
    - matcher: Bash(pip install*)
      hook: |
        echo "[update-deps] Creating requirements backup..."
        pip freeze > requirements.bak.txt 2>/dev/null || true
    - matcher: Bash(bundle install*)
      hook: |
        echo "[update-deps] Creating Gemfile.lock backup..."
        cp Gemfile.lock Gemfile.lock.bak 2>/dev/null || true
---

# Update Dependencies Command

Safely update project dependencies with comprehensive checks.

## Usage

```
/update-deps [target]
```

**Arguments:**
- `target` (optional):
  - Package name: Update specific package
  - `all`: Update all packages
  - `security`: Security updates only
  - `patch`: Patch updates only
  - `minor`: Minor updates only

## Examples

```bash
# Audit and show update plan
/update-deps

# Update specific package
/update-deps lodash

# Security updates only
/update-deps security

# All patch updates
/update-deps patch

# All updates
/update-deps all
```

## Workflow

### 1. Dependency Audit

First, audit current state:

```markdown
## Dependency Audit

**Package Manager:** npm
**Total Dependencies:** 145

### Outdated Summary
| Type | Count | Risk |
|------|-------|------|
| Patch | 8 | Low |
| Minor | 12 | Medium |
| Major | 3 | High |

### Security Vulnerabilities
| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 2 |
| Medium | 3 |
| Low | 5 |
```

### 2. Generate Update Plan

Create prioritized plan:

```markdown
## Update Plan

### Phase 1: Security (Immediate)
```bash
npm install lodash@4.17.21  # CVE-2021-23337
npm install axios@1.6.0     # CVE-2023-45857
```
- **Risk:** Low (patch updates)
- **Testing:** Run test suite

### Phase 2: Patch Updates
```bash
npm update  # Updates within semver range
```
- **Risk:** Low
- **Testing:** Run test suite

### Phase 3: Minor Updates
| Package | From | To | Breaking |
|---------|------|-----|----------|
| jest | 29.5.0 | 29.7.0 | No |
| typescript | 5.0.4 | 5.3.0 | No |

- **Risk:** Medium
- **Testing:** Full test suite + build

### Phase 4: Major Updates (Requires Planning)
| Package | From | To | Migration |
|---------|------|-----|-----------|
| axios | 0.21.0 | 1.6.0 | Required |
| react | 17.0.2 | 18.2.0 | Required |

- **Risk:** High
- **Testing:** Full regression
```

### 3. Execute Updates

With user confirmation, execute:

```bash
# Create update branch
git checkout -b deps/update-YYYY-MM-DD

# Phase 1: Security
npm install lodash@4.17.21 axios@1.6.0
npm test
git commit -am "Security: Update lodash and axios"

# Phase 2: Patch
npm update
npm test
git commit -am "deps: Apply patch updates"

# Phase 3: Minor
npm install jest@29.7.0 typescript@5.3.0
npm test && npm run build
git commit -am "deps: Apply minor updates"
```

### 4. Verify Updates

```markdown
## Verification

### Tests
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Build succeeds
- [ ] Type check passes

### Security
- [ ] npm audit shows no high/critical
- [ ] No new vulnerabilities introduced

### Compatibility
- [ ] No peer dependency warnings
- [ ] Engine requirements satisfied
```

## Update Modes

### Interactive (Default)

Shows plan and asks for confirmation before each phase.

### Auto-merge Safe

```bash
/update-deps --auto-safe
```
Automatically applies patch updates that pass tests.

### Dry Run

```bash
/update-deps --dry-run
```
Shows plan without making changes.

## Configuration

Create `.update-deps.yaml` for customization:

```yaml
# .update-deps.yaml
strategy: conservative  # conservative | moderate | aggressive

auto_merge:
  patch: true
  minor: false
  major: false

ignore:
  - legacy-package  # Don't update this

security:
  auto_fix: true
  severity_threshold: high  # Fix high and critical automatically

notifications:
  slack: "#dev-deps"
```

## Output

### Report Location

```
.claude/
└── maintenance/
    └── deps-update-YYYY-MM-DD.md
```

### Report Format

```markdown
# Dependency Update Report

**Date:** 2024-01-15
**Branch:** deps/update-2024-01-15

## Updates Applied

### Security
- lodash 4.17.15 → 4.17.21 (CVE-2021-23337)
- axios 0.21.0 → 1.6.0 (CVE-2023-45857)

### Patch
- 8 packages updated

### Minor
- jest 29.5.0 → 29.7.0
- typescript 5.0.4 → 5.3.0

## Verification
- All tests passing
- Build successful
- No new vulnerabilities

## Deferred
- react 17 → 18 (requires migration)
```

## Rollback

If issues occur:

```bash
# Revert to previous state
git checkout main -- package.json package-lock.json
npm install

# Or revert specific commit
git revert <commit-hash>
npm install
```

## Related

- `/health-report` - View dependency health
- `/scan-debt` - Identify dependency debt
- `/workflows:maintain` - Full maintenance workflow
- `dependency-management` skill - Update strategies
- `dependency-auditor` agent - Audit agent
