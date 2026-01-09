---
name: dependency-auditor
description: Audit project dependencies for updates, security vulnerabilities, and compatibility issues with safe update recommendations
---

# Dependency Auditor Agent

Audit and manage project dependencies safely.

## Purpose

This agent analyzes project dependencies to:
- Identify outdated packages
- Detect security vulnerabilities
- Check compatibility constraints
- Generate safe update plans

## Skills Used

- `dependency-management` - Update strategies and security audit patterns

## Activation

This agent is triggered by:
- `/update-deps` command
- `/workflows:maintain` workflow
- `/health-report` command

## Workflow

### 1. Scan Dependencies

Detect package manager and scan:

```markdown
## Dependency Scan

**Package Manager:** npm
**Lock File:** package-lock.json
**Total Dependencies:** 145 (42 direct, 103 transitive)

### Outdated Packages
| Package | Current | Latest | Type |
|---------|---------|--------|------|
| lodash | 4.17.15 | 4.17.21 | patch |
| axios | 0.21.0 | 1.6.0 | major |
| jest | 29.5.0 | 29.7.0 | minor |
| typescript | 5.0.4 | 5.3.0 | minor |

### Summary
- Patch updates: 8
- Minor updates: 12
- Major updates: 3
```

### 2. Security Audit

Check for vulnerabilities:

```markdown
## Security Audit

### Vulnerabilities Found

#### Critical (0)
None

#### High (2)
1. **lodash < 4.17.21**
   - CVE-2021-23337
   - Prototype pollution
   - Fix: Update to 4.17.21

2. **axios < 0.27.0**
   - CVE-2023-45857
   - SSRF vulnerability
   - Fix: Update to 1.6.0

#### Medium (3)
[List medium vulnerabilities]

#### Low (5)
[List low vulnerabilities]
```

### 3. Compatibility Analysis

Check for compatibility issues:

```markdown
## Compatibility Check

### Engine Requirements
- Node.js: >=18.0.0 ✅
- npm: >=9.0.0 ✅

### Peer Dependencies
| Package | Requires | Installed | Status |
|---------|----------|-----------|--------|
| @testing-library/react | react >=18 | 18.2.0 | ✅ |
| eslint-plugin-react | react >=16.8 | 18.2.0 | ✅ |

### Breaking Changes
- **axios 1.x**: Changed default behavior for redirects
- **typescript 5.x**: New decorator syntax
```

### 4. Generate Update Plan

Create prioritized update plan:

```markdown
## Update Plan

### Immediate (Security)
Priority: Critical
```bash
npm install lodash@4.17.21
npm install axios@1.6.0
```

**Testing:**
- Run full test suite
- Check API integrations

### This Sprint (Minor/Patch)
Priority: High
```bash
npm install jest@29.7.0 typescript@5.3.0
npm update # All patch updates
```

**Testing:**
- Run tests
- Check build

### Planned (Major)
Priority: Medium

#### axios 0.x → 1.x
**Breaking changes:**
- Redirect behavior changed
- Response type changes

**Migration:**
1. Update axios
2. Update redirect handling
3. Update response type handling
4. Run integration tests

**Estimated effort:** 2-4 hours
```

### 5. Execute Updates (Optional)

If requested, execute the update plan:

```bash
# Create update branch
git checkout -b deps/security-updates-2024-01-15

# Install security updates
npm install lodash@4.17.21 axios@1.6.0

# Run tests
npm test

# If passing, commit
git add package.json package-lock.json
git commit -m "Security: Update lodash and axios"
```

## Output

### Report Location

```
.claude/
└── maintenance/
    └── deps-report-YYYY-MM-DD.md
```

### Report Format

```markdown
# Dependency Audit Report

**Generated:** 2024-01-15
**Package Manager:** npm

## Summary
| Category | Count |
|----------|-------|
| Total Dependencies | 145 |
| Outdated | 23 |
| Vulnerabilities | 10 |
| Breaking Updates | 3 |

## Security Issues
[Detailed vulnerability list]

## Update Plan
[Prioritized update steps]

## Recommendations
[Specific actions to take]
```

## Update Strategies

### Conservative
```markdown
1. Security updates only
2. Patch updates monthly
3. Minor updates quarterly
4. Major updates planned
```

### Moderate
```markdown
1. Security updates immediately
2. Patch/minor updates weekly
3. Major updates monthly
```

### Aggressive
```markdown
1. All updates weekly
2. Automated PR merging for passing tests
3. Major updates as available
```

## Integration

This agent integrates with:
- `/update-deps` command for manual audits
- `/workflows:maintain` for scheduled maintenance
- `/health-report` for dependency health metrics
- `codebase-health` agent for health scoring
