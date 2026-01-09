---
name: health-report
description: Generate comprehensive codebase health report
argument-hint: "[optional: output format]"
---

# Health Report Command

Generate a comprehensive health report for your codebase.

## Usage

```
/health-report [format]
```

**Arguments:**
- `format` (optional) - Output format: `markdown` (default), `json`, `html`

## Examples

```bash
# Generate default markdown report
/health-report

# Generate JSON report
/health-report json

# Generate HTML report
/health-report html
```

## Workflow

### 1. Data Collection

The command gathers metrics from multiple sources:

```markdown
## Collecting Health Metrics

### Test Coverage
- Running coverage analysis...
- Line Coverage: 78%
- Branch Coverage: 65%
- Function Coverage: 82%

### Code Quality
- Running linters...
- Warnings: 45
- Errors: 3
- Style Issues: 120

### Complexity Analysis
- Analyzing complexity...
- Average: 4.2
- High Complexity Files: 8

### Dependency Health
- Checking dependencies...
- Outdated: 12 packages
- Vulnerabilities: 7 (2 high, 5 moderate)

### Technical Debt
- Scanning for debt...
- Total Items: 30
- P0/P1 Items: 9
```

### 2. Calculate Health Score

```markdown
## Health Score Calculation

| Component | Score | Weight | Weighted |
|-----------|-------|--------|----------|
| Test Coverage | 78 | 25% | 19.5 |
| Code Quality | 72 | 20% | 14.4 |
| Complexity | 68 | 15% | 10.2 |
| Dependencies | 65 | 15% | 9.75 |
| Technical Debt | 60 | 15% | 9.0 |
| Documentation | 55 | 10% | 5.5 |

### Overall Health Score: 68/100

### Grade: Fair (Needs Attention)
```

### 3. Compare Trends

```markdown
## Trend Analysis

### 30-Day Comparison
| Metric | 30d Ago | Now | Change |
|--------|---------|-----|--------|
| Health Score | 65 | 68 | ↑ +3 |
| Test Coverage | 72% | 78% | ↑ +6% |
| Tech Debt Items | 25 | 30 | ↑ +5 |
| Vulnerabilities | 3 | 7 | ↑ +4 |

### Trend Direction: Mixed
- Positive: Coverage improving
- Negative: Debt and vulnerabilities increasing
```

### 4. Generate Recommendations

```markdown
## Recommendations

### Immediate (This Week)
1. **Fix security vulnerabilities**
   - 2 high-severity issues
   - Estimated: 30 minutes
   - Priority: Critical

2. **Address P0 technical debt**
   - Circular dependency in core
   - Estimated: 2-4 hours
   - Priority: High

### Short-term (This Sprint)
3. **Improve payment module coverage**
   - Current: 45% → Target: 80%
   - Estimated: 4-6 hours

4. **Reduce top complexity files**
   - 2 files above threshold
   - Estimated: 4-6 hours

### Medium-term (This Quarter)
5. **Documentation sprint**
   - API docs, architecture diagrams
   - Estimated: 2-3 days

6. **Dependency update**
   - 12 outdated packages
   - Estimated: 1-2 days
```

## Report Output

### Location

```
.claude/
└── reports/
    └── health-YYYY-MM-DD.md
```

### Report Structure

```markdown
# Codebase Health Report

**Generated:** 2024-01-15 14:30 UTC
**Scope:** Full codebase
**Previous:** 2024-01-01

---

## Executive Summary

**Health Score: 68/100 (Fair)**

### Key Findings
- ✅ Test coverage improved 6%
- ⚠️ 7 security vulnerabilities detected
- ❌ Technical debt increased by 5 items

### Critical Actions
1. Fix 2 high-severity vulnerabilities
2. Address circular dependency (P0)
3. Improve payment module testing

---

## Detailed Analysis

### Test Coverage
[Breakdown by module, trends, gaps]

### Code Quality
[Linting results, patterns, issues]

### Complexity
[High-complexity files, recommendations]

### Dependencies
[Outdated packages, vulnerabilities]

### Technical Debt
[Summary, top items, trends]

### Documentation
[Coverage, gaps, recommendations]

---

## Recommendations
[Prioritized action items]

---

## Appendix
[Raw metrics, data sources, methodology]
```

## Health Score Components

### Test Coverage (25%)
| Score | Coverage |
|-------|----------|
| 100 | 95%+ |
| 80 | 85-94% |
| 60 | 70-84% |
| 40 | 50-69% |
| 20 | <50% |

### Code Quality (20%)
Based on linting errors, warnings, and style issues normalized per 1000 lines.

### Complexity (15%)
Based on average cyclomatic complexity and percentage of files above threshold.

### Dependencies (15%)
| Score | Status |
|-------|--------|
| 100 | All current, no vulnerabilities |
| 75 | Minor updates available |
| 50 | Major updates needed |
| 25 | Known vulnerabilities |
| 0 | Critical vulnerabilities |

### Technical Debt (15%)
Based on total debt items weighted by priority level.

### Documentation (10%)
Based on README presence, API documentation, and inline documentation coverage.

## Related

- `/scan-debt` - Detailed technical debt scan
- `/analyze-coverage` - Detailed coverage analysis
- `/update-deps` - Update dependencies
- `codebase-health` agent - Health analysis agent
