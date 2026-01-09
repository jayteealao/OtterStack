---
name: scan-debt
description: Scan codebase for technical debt and create tracking files
argument-hint: "[optional: specific area or type]"
---

# Scan Debt Command

Systematically identify and track technical debt in your codebase.

## Usage

```
/scan-debt [scope]
```

**Arguments:**
- `scope` (optional) - Specific area or type to scan:
  - Directory path: `src/services/`
  - Debt type: `code`, `architecture`, `test`, `documentation`

## Examples

```bash
# Scan entire codebase
/scan-debt

# Scan specific directory
/scan-debt src/services/

# Scan for specific debt type
/scan-debt code

# Scan for test debt only
/scan-debt test
```

## Workflow

### 1. Code Analysis

The command scans for debt indicators:

```markdown
## Scanning for Technical Debt

### Code Debt Indicators
- [x] TODO/FIXME/HACK comments: 23 found
- [x] Long methods (>50 lines): 8 found
- [x] High complexity (>10): 5 found
- [x] Duplicated code: 3 blocks found
- [x] Magic numbers: 12 found

### Architecture Debt Indicators
- [x] Circular dependencies: 2 found
- [x] God classes (>500 lines): 1 found
- [x] Inconsistent patterns: 4 found

### Test Debt Indicators
- [x] Low coverage (<60%): 6 modules
- [x] Skipped tests: 4 found
- [x] Missing integration tests: 3 areas

### Documentation Debt Indicators
- [x] Missing README: 2 modules
- [x] Outdated docs: 5 files
- [x] Missing API docs: 8 endpoints
```

### 2. Scoring and Prioritization

Each debt item is scored:

```markdown
## Debt Scoring

### Impact Levels
| Score | Level | Description |
|-------|-------|-------------|
| 5 | Critical | Blocks development, causes outages |
| 4 | High | Significantly slows work, frequent bugs |
| 3 | Medium | Noticeable friction, occasional issues |
| 2 | Low | Minor inconvenience |
| 1 | Minimal | Cosmetic, nice-to-have fixes |

### Effort Levels
| Score | Level | Estimate |
|-------|-------|----------|
| 5 | Very High | 1+ weeks |
| 4 | High | 2-5 days |
| 3 | Medium | 4-8 hours |
| 2 | Low | 1-4 hours |
| 1 | Minimal | <1 hour |

### Priority Formula
```
Priority Score = Impact × (6 - Effort)
```

Higher score = Higher priority
```

### 3. Create Debt Files

Generate tracking files in `.claude/debt/`:

```yaml
# Example: .claude/debt/code/payment-complexity.yaml
---
id: DEBT-2024-001
title: Complex Payment Processing Logic
category: code
subcategory: complexity
location: src/services/payment-processor.ts:45-180

discovered:
  date: 2024-01-15
  by: debt-tracker-agent

impact:
  score: 4
  areas: [payments, checkout]
  risks:
    - Bug introduction during changes
    - Difficult onboarding

effort:
  score: 3
  estimate: "4-6 hours"

priority:
  score: 12
  level: P1

status: identified
```

### 4. Generate Summary

Create summary report:

```markdown
## Technical Debt Summary

**Scan Date:** 2024-01-15
**Scope:** Full codebase

### Overview
| Category | Items | P0 | P1 | P2 | P3 |
|----------|-------|----|----|----|----|
| Code | 12 | 0 | 3 | 6 | 3 |
| Architecture | 4 | 1 | 2 | 1 | 0 |
| Test | 8 | 0 | 2 | 4 | 2 |
| Documentation | 6 | 0 | 1 | 2 | 3 |
| **Total** | **30** | **1** | **8** | **13** | **8** |

### Immediate Actions
1. Fix circular dependency (P0)
2. Refactor payment processor (P1)
3. Add missing API tests (P1)

### Created Files
- `.claude/debt/code/` - 12 files
- `.claude/debt/architecture/` - 4 files
- `.claude/debt/test/` - 8 files
- `.claude/debt/documentation/` - 6 files
```

## Output Structure

```
.claude/
└── debt/
    ├── README.md              # Debt tracking overview
    ├── summary.md             # Latest scan summary
    ├── code/                  # Code-level debt
    │   └── *.yaml
    ├── architecture/          # Architecture debt
    │   └── *.yaml
    ├── test/                  # Test debt
    │   └── *.yaml
    └── documentation/         # Documentation debt
        └── *.yaml
```

## Debt File Format

```yaml
---
id: DEBT-YYYY-NNN
title: Short descriptive title
category: code | architecture | test | documentation
subcategory: complexity | duplication | coverage | etc.
location: path/to/file.ext:line-range

discovered:
  date: YYYY-MM-DD
  by: agent-name | manual
  context: "How it was discovered"

impact:
  score: 1-5
  areas:
    - affected area 1
    - affected area 2
  risks:
    - risk description 1
    - risk description 2

effort:
  score: 1-5
  estimate: "time estimate"
  approach: |
    How to fix this debt

priority:
  score: calculated
  level: P0 | P1 | P2 | P3
  rationale: "Why this priority"

status: identified | planned | in_progress | resolved
assigned_to: null | username
target_date: null | YYYY-MM-DD

related:
  - DEBT-YYYY-NNN
```

## Related

- `/health-report` - Generate comprehensive health report
- `/refactor` - Address specific debt items
- `technical-debt` skill - Debt patterns and resolution
- `debt-tracker` agent - Debt scanning agent
