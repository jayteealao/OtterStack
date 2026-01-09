---
name: codebase-health
description: Generate comprehensive codebase health reports combining test coverage, technical debt, dependencies, and code quality metrics
---

# Codebase Health Agent

Generate comprehensive health reports for codebases.

## Purpose

This agent synthesizes multiple analysis sources to create a complete health picture:
- Test coverage analysis
- Technical debt assessment
- Dependency health
- Code quality metrics
- Trend analysis over time

## Skills Used

- `technical-debt` - Debt analysis and scoring
- `test-patterns` - Coverage analysis patterns

## Activation

This agent is triggered by:
- `/health-report` command
- `/workflows:maintain` workflow
- Scheduled health checks

## Workflow

### 1. Gather Metrics

Collect data from multiple sources:

```markdown
## Data Collection

### Test Coverage
```bash
# Run coverage analysis
pytest --cov=src --cov-report=json
# or
npx jest --coverage --coverageReporters=json-summary
```

**Results:**
- Line Coverage: 78%
- Branch Coverage: 65%
- Function Coverage: 82%

### Code Quality
```bash
# Run linting
npx eslint src/ --format json
# or
pylint src/ --output-format=json
```

**Results:**
- Warnings: 45
- Errors: 3
- Style Issues: 120

### Complexity
```bash
# Python
radon cc src/ -j
# JavaScript
npx complexity-report src/ --format json
```

**Results:**
- Average Complexity: 4.2
- High Complexity Files: 8
- Very High (>15): 2

### Dependencies
```bash
# Check for outdated
npm outdated --json
# Check for vulnerabilities
npm audit --json
```

**Results:**
- Outdated: 12 packages
- Vulnerabilities: 2 high, 5 moderate
```

### 2. Analyze Technical Debt

Run debt analysis:

```markdown
## Technical Debt Summary

### By Category
| Category | Count | Trend |
|----------|-------|-------|
| Code | 12 | ↑ +2 |
| Architecture | 4 | → 0 |
| Test | 8 | ↓ -1 |
| Documentation | 6 | → 0 |

### Top Issues
1. **P0:** Circular dependency in core modules
2. **P1:** Payment processor complexity (135 lines)
3. **P1:** Missing API documentation
```

### 3. Calculate Health Score

Compute overall health score:

```markdown
## Health Score Calculation

### Component Scores (0-100)

| Component | Score | Weight | Weighted |
|-----------|-------|--------|----------|
| Test Coverage | 78 | 25% | 19.5 |
| Code Quality | 72 | 20% | 14.4 |
| Complexity | 68 | 15% | 10.2 |
| Dependencies | 65 | 15% | 9.75 |
| Technical Debt | 60 | 15% | 9.0 |
| Documentation | 55 | 10% | 5.5 |

### Overall Health Score: 68/100 (Fair)

### Grade Scale
- 90-100: Excellent
- 80-89: Good
- 70-79: Fair
- 60-69: Needs Attention
- Below 60: Critical
```

### 4. Generate Trends

Compare to previous reports:

```markdown
## Trends (Last 30 Days)

### Health Score Trend
```
Score
100│
 80│    ╭─╮    ╭───────╮
 60│╭───╯ ╰────╯       ╰──
 40│
 20│
  0└────────────────────────
    Jan 1   Jan 15   Jan 30
```

### Component Trends
| Component | 30d Ago | Now | Change |
|-----------|---------|-----|--------|
| Test Coverage | 72% | 78% | ↑ +6% |
| Code Quality | 68 | 72 | ↑ +4 |
| Technical Debt | 25 items | 30 items | ↑ +5 |
| Dependencies | 3 vulns | 7 vulns | ↑ +4 |
```

### 5. Generate Recommendations

Create actionable recommendations:

```markdown
## Recommendations

### Immediate Actions (This Week)
1. **Fix 2 high-severity vulnerabilities**
   - `lodash` → Update to 4.17.21
   - `axios` → Update to 1.6.0
   - Effort: 30 minutes

2. **Address P0 circular dependency**
   - Impact: Build stability
   - Effort: 2-4 hours

### Short-term (This Sprint)
3. **Improve test coverage for payment module**
   - Current: 45% → Target: 80%
   - Focus: `src/services/payment/`
   - Effort: 4-6 hours

4. **Reduce complexity in top 2 files**
   - `payment-processor.ts`: 15 → <10
   - `order-handler.ts`: 12 → <10
   - Effort: 4-6 hours

### Medium-term (This Quarter)
5. **Establish documentation standards**
   - Create API documentation
   - Add architecture diagrams
   - Effort: 2-3 days

6. **Dependency update sprint**
   - Update 12 outdated packages
   - Effort: 1-2 days
```

## Output

### Report Location

```
.claude/
└── reports/
    └── health-YYYY-MM-DD.md
```

### Report Structure

```markdown
# Codebase Health Report

**Generated:** YYYY-MM-DD HH:MM
**Scope:** Full codebase
**Previous Report:** YYYY-MM-DD

## Executive Summary

Overall Health Score: **68/100 (Fair)**

Key Findings:
- Test coverage improved by 6%
- 7 new security vulnerabilities detected
- Technical debt increased by 5 items

## Detailed Analysis

### Test Coverage
[Detailed coverage breakdown]

### Code Quality
[Linting and quality metrics]

### Technical Debt
[Debt summary and trends]

### Dependencies
[Dependency health analysis]

## Recommendations
[Prioritized action items]

## Appendix
[Raw metrics and data sources]
```

## Integration

This agent integrates with:
- `/health-report` command for manual reports
- `/workflows:maintain` for scheduled analysis
- `debt-tracker` agent for debt data
- `test-coverage-analyzer` agent for coverage data
- `dependency-auditor` agent for dependency data
