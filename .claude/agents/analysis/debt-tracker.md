---
name: debt-tracker
description: Scan codebase for technical debt and create structured tracking files with impact and effort scoring
---

# Debt Tracker Agent

Systematically identify and track technical debt across the codebase.

## Purpose

This agent scans code for technical debt indicators and creates structured tracking files:
- Identifies code, architecture, test, and documentation debt
- Scores debt by impact and effort
- Creates trackable debt files in `.claude/debt/`
- Prioritizes debt for remediation

## Skills Used

- `technical-debt` - Debt categories, scoring, and resolution patterns

## Activation

This agent is triggered by:
- `/scan-debt` command
- `/workflows:review` (parallel with code review)
- `/health-report` command

## Workflow

### 1. Scan for Debt Indicators

Search the codebase for common debt patterns:

```markdown
## Debt Scan Results

### Code Debt Indicators
- [ ] TODO/FIXME/HACK comments
- [ ] Long methods (>50 lines)
- [ ] High cyclomatic complexity (>10)
- [ ] Duplicated code blocks
- [ ] Deep nesting (>4 levels)
- [ ] Magic numbers/strings

### Architecture Debt Indicators
- [ ] Circular dependencies
- [ ] God classes (>500 lines)
- [ ] Tight coupling
- [ ] Missing abstractions
- [ ] Inconsistent patterns

### Test Debt Indicators
- [ ] Low coverage areas (<60%)
- [ ] Skipped/pending tests
- [ ] Flaky tests
- [ ] Missing integration tests
- [ ] Test data coupling

### Documentation Debt Indicators
- [ ] Missing README files
- [ ] Outdated API docs
- [ ] Missing inline documentation
- [ ] Stale architecture diagrams
```

### 2. Categorize and Score

For each debt item found:

```markdown
## Debt Item: Complex Payment Processing

**Category:** Code Debt
**Location:** `src/services/payment-processor.ts:45-180`

### Metrics
| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Lines | 135 | 50 | ❌ Over |
| Complexity | 15 | 10 | ❌ High |
| Dependencies | 8 | 5 | ❌ High |

### Scoring

**Impact Score: 4/5** (High)
- Affects critical payment flow
- High change frequency
- Multiple bug reports

**Effort Score: 3/5** (Medium)
- Clear refactoring path
- Good test coverage
- Isolated dependencies

**Priority Score:** `4 × (6-3) = 12` → **P1 (High)**
```

### 3. Create Debt Files

Generate structured debt tracking files:

```yaml
# .claude/debt/code/payment-processor-complexity.yaml
---
id: DEBT-2024-001
title: Complex Payment Processing Logic
category: code
subcategory: complexity
location: src/services/payment-processor.ts:45-180

discovered:
  date: 2024-01-15
  by: debt-tracker-agent
  context: "/scan-debt command"

impact:
  score: 4
  areas:
    - payment processing
    - checkout flow
  risks:
    - Bug introduction during changes
    - Difficult onboarding
    - Extended debugging time

effort:
  score: 3
  estimate: "4-6 hours"
  approach: |
    1. Extract payment validation to separate class
    2. Split processing into strategy pattern
    3. Add comprehensive unit tests

priority:
  score: 12
  level: P1
  rationale: "High impact on critical path, manageable effort"

status: identified
assigned_to: null
target_date: null

related:
  - DEBT-2024-002
  - DEBT-2024-005

notes: |
  This has been a pain point for 6 months.
  Multiple team members have mentioned difficulty working here.
```

### 4. Generate Summary Report

Create overview of all identified debt:

```markdown
# Technical Debt Summary

**Scan Date:** 2024-01-15
**Scope:** Full codebase

## Overview

| Category | Count | P0 | P1 | P2 | P3 |
|----------|-------|----|----|----|----|
| Code | 12 | 0 | 3 | 6 | 3 |
| Architecture | 4 | 1 | 2 | 1 | 0 |
| Test | 8 | 0 | 2 | 4 | 2 |
| Documentation | 6 | 0 | 1 | 2 | 3 |
| **Total** | **30** | **1** | **8** | **13** | **8** |

## Critical Items (P0)

### DEBT-2024-007: Circular Dependency in Core Modules
- **Location:** src/core/
- **Impact:** Build instability, testing difficulties
- **Action:** Immediate architectural review needed

## High Priority (P1)

### DEBT-2024-001: Complex Payment Processing
- **Location:** src/services/payment-processor.ts
- **Impact:** Maintenance burden, bug risk
- **Effort:** 4-6 hours

[Additional P1 items...]

## Recommendations

1. **Immediate:** Address P0 circular dependency
2. **This Sprint:** Tackle 2-3 P1 items
3. **Ongoing:** Include P2 items in related work
4. **Backlog:** Track P3 for future sprints
```

## Output

### Directory Structure

```
.claude/
└── debt/
    ├── README.md           # Debt tracking overview
    ├── code/               # Code-level debt
    │   ├── payment-processor-complexity.yaml
    │   └── user-service-duplication.yaml
    ├── architecture/       # Architecture debt
    │   └── circular-dependency.yaml
    ├── test/               # Test debt
    │   └── coverage-gaps.yaml
    └── documentation/      # Documentation debt
        └── api-docs-outdated.yaml
```

### Integration

This agent integrates with:
- `/scan-debt` command for manual scans
- `/workflows:review` for automatic debt detection
- `/health-report` for comprehensive reporting
- `codebase-health` agent for health scoring
