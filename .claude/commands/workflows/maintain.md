---
name: workflows:maintain
description: Run maintenance tasks to keep codebase healthy
argument-hint: "[full|quick|deps|debt|tests]"
---

# Maintenance Workflow

Systematic codebase maintenance for ongoing health.

## Usage

```
/workflows:maintain [mode]
```

**Modes:**
- `full` (default) - Complete maintenance cycle
- `quick` - Fast checks only
- `deps` - Dependency maintenance only
- `debt` - Technical debt scan only
- `tests` - Test coverage analysis only

## Examples

```bash
# Full maintenance cycle
/workflows:maintain

# Quick health check
/workflows:maintain quick

# Dependencies only
/workflows:maintain deps

# Technical debt scan
/workflows:maintain debt
```

## Full Workflow

### Phase 1: Parallel Analysis

Run these analyses concurrently:

```markdown
## Starting Maintenance Cycle

### Parallel Tasks
1. 🔄 Dependency audit (dependency-auditor agent)
2. 🔄 Technical debt scan (debt-tracker agent)
3. 🔄 Coverage analysis (test-coverage-analyzer agent)

Waiting for all tasks to complete...
```

### Phase 2: Dependency Health

```markdown
## Dependency Report

### Security
| Severity | Count | Action |
|----------|-------|--------|
| Critical | 0 | - |
| High | 2 | Immediate update |
| Medium | 3 | This sprint |
| Low | 5 | Backlog |

### Updates Available
| Type | Count |
|------|-------|
| Patch | 8 |
| Minor | 12 |
| Major | 3 |

### Recommendations
1. Update lodash (security)
2. Update axios (security)
3. Plan react migration
```

### Phase 3: Technical Debt

```markdown
## Technical Debt Report

### New Debt Identified
| Category | New | Total |
|----------|-----|-------|
| Code | +2 | 14 |
| Architecture | 0 | 4 |
| Test | +1 | 9 |
| Documentation | 0 | 6 |

### Priority Items
1. **P0:** Circular dependency in core (architecture)
2. **P1:** Payment processor complexity (code)
3. **P1:** Missing API tests (test)

### Trends
- Total debt: ↑ 3 items since last scan
- P0/P1 items: ↓ 1 (good progress!)
```

### Phase 4: Test Coverage

```markdown
## Coverage Report

### Overall Coverage
- Lines: 78% (target: 80%)
- Branches: 65% (target: 75%)
- Functions: 82% (target: 85%)

### Critical Gaps
1. `src/services/payment/` - 45% coverage
2. `src/api/auth/` - 60% coverage

### Recommendations
1. Add tests for payment processing
2. Add edge case tests for auth
```

### Phase 5: Health Score

```markdown
## Codebase Health Score

### Current Score: 68/100 (Fair)

| Component | Score | Trend |
|-----------|-------|-------|
| Test Coverage | 78 | ↑ +3 |
| Code Quality | 72 | → 0 |
| Dependencies | 65 | ↓ -5 |
| Technical Debt | 60 | ↓ -2 |
| Documentation | 55 | → 0 |

### Grade: C+ (Needs Attention)
```

### Phase 6: Triage

Present findings for decisions:

```markdown
## Maintenance Triage

### Immediate Actions (Recommended)
- [ ] Fix 2 high-severity vulnerabilities
- [ ] Address P0 circular dependency

### This Sprint
- [ ] Apply patch/minor dependency updates
- [ ] Add tests for payment module
- [ ] Address 2 P1 technical debt items

### Backlog
- [ ] Plan React 18 migration
- [ ] Improve documentation coverage
- [ ] Address remaining tech debt

### Quick Wins
- [ ] Update lodash (5 min)
- [ ] Fix 3 lint warnings (15 min)
- [ ] Add missing README (30 min)
```

### Phase 7: Resolution (Optional)

If user chooses to proceed:

```markdown
## Executing Maintenance Tasks

### Security Updates
```bash
npm install lodash@4.17.21 axios@1.6.0
npm test  # ✅ Passing
```

### Patch Updates
```bash
npm update
npm test  # ✅ Passing
npm run build  # ✅ Success
```

### Created PRs
- deps/security-updates-2024-01-15
- deps/patch-updates-2024-01-15

### Created Todos
- `.claude/todos/maintenance-2024-01-15.yaml`
```

## Output

### Report Locations

```
.claude/
├── reports/
│   └── health-YYYY-MM-DD.md
├── maintenance/
│   └── deps-report-YYYY-MM-DD.md
├── debt/
│   └── [new debt files]
└── todos/
    └── maintenance-YYYY-MM-DD.yaml
```

## Quick Mode

Fast health check without full analysis:

```markdown
## Quick Health Check

### Dependencies
- Security issues: 2 high
- Outdated: 23 packages

### Tests
- Coverage: 78%
- Failing: 0

### Lint
- Errors: 0
- Warnings: 45

### Build
- Status: ✅ Passing

### Quick Score: 70/100
```

## Schedule Recommendations

```markdown
## Recommended Maintenance Schedule

### Daily (Automated)
- Security vulnerability scan
- Test suite execution
- Build verification

### Weekly
- [ ] Run `/workflows:maintain quick`
- [ ] Review and merge Dependabot PRs
- [ ] Address any critical issues

### Monthly
- [ ] Run `/workflows:maintain full`
- [ ] Plan major dependency updates
- [ ] Review technical debt trends

### Quarterly
- [ ] Comprehensive health review
- [ ] Major version migrations
- [ ] Architecture review
```

## Related

- `/update-deps` - Update dependencies
- `/scan-debt` - Technical debt scan
- `/analyze-coverage` - Coverage analysis
- `/health-report` - Generate health report
- `/workflows:review` - Code review workflow
