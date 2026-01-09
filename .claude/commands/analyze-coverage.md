---
name: analyze-coverage
description: Analyze test coverage and identify gaps in critical code paths
argument-hint: "[optional: specific module or file]"
---

# Analyze Coverage Command

Analyze test coverage and identify gaps that need tests.

## Usage

```
/analyze-coverage [target]
```

**Arguments:**
- `target` (optional) - Specific module, file, or directory to analyze

## Examples

```bash
# Analyze entire project
/analyze-coverage

# Analyze specific module
/analyze-coverage src/services/

# Analyze specific file
/analyze-coverage src/utils/validation.ts
```

## Workflow

### 1. Run Coverage Analysis

Detect and run the appropriate coverage tool:

```bash
# Python
pytest --cov=src --cov-report=term-missing --cov-report=json

# JavaScript/TypeScript
npx jest --coverage --coverageReporters=json-summary

# Ruby
bundle exec rspec --format json --out coverage/coverage.json

# Go
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### 2. Parse Coverage Report

```markdown
## Coverage Summary

**Overall Coverage:** X%

### By Module
| Module | Lines | Branches | Functions | Status |
|--------|-------|----------|-----------|--------|
| src/services | 85% | 72% | 90% | ⚠️ |
| src/utils | 95% | 88% | 100% | ✅ |
| src/models | 45% | 30% | 60% | ❌ |

### Legend
- ✅ Above threshold (80%+)
- ⚠️ Near threshold (60-79%)
- ❌ Below threshold (<60%)
```

### 3. Identify Critical Gaps

Use the `test-coverage-analyzer` agent:

```markdown
## Critical Uncovered Code

### High Priority (Business Logic)
1. **`src/services/payment.ts:45-67`**
   - `processRefund()` - No test coverage
   - Risk: Financial transaction handling

2. **`src/services/auth.ts:120-145`**
   - `validateToken()` - Only happy path tested
   - Risk: Security vulnerability

### Medium Priority (Error Handling)
1. **`src/utils/api-client.ts:89-102`**
   - Error retry logic untested
   - Risk: Silent failures

### Low Priority (Edge Cases)
1. **`src/utils/format.ts:34-40`**
   - Empty input handling
   - Risk: UI display issues
```

### 4. Generate Test Recommendations

```markdown
## Recommended Tests

### Immediate (P0)
- [ ] `test_process_refund_with_valid_amount`
- [ ] `test_process_refund_with_zero_amount_fails`
- [ ] `test_validate_token_with_expired_token`
- [ ] `test_validate_token_with_invalid_signature`

### Soon (P1)
- [ ] `test_api_client_retry_on_network_error`
- [ ] `test_api_client_retry_exhaustion`

### Later (P2)
- [ ] `test_format_with_empty_string`
- [ ] `test_format_with_null_input`
```

### 5. Create Todos (Optional)

If requested, create todo items for coverage gaps:

```markdown
## Coverage Todo Items

Created in `.claude/todos/coverage-gaps.yaml`:
- High priority items for immediate attention
- Linked to specific uncovered lines
- Estimated effort for each test
```

## Coverage Thresholds

| Category | Minimum | Target | Critical |
|----------|---------|--------|----------|
| Overall | 70% | 80% | 90% |
| Business Logic | 85% | 95% | 100% |
| Security Code | 90% | 100% | 100% |
| Utilities | 60% | 75% | 85% |
| UI Components | 50% | 70% | 80% |

## Output Files

```
.claude/
└── reports/
    └── coverage-analysis-YYYY-MM-DD.md
```

## Report Format

```markdown
# Coverage Analysis Report
Date: YYYY-MM-DD
Target: [module/file/all]

## Executive Summary
- Overall coverage: X%
- Critical gaps: N
- Recommended tests: M

## Detailed Findings
[...]

## Action Items
[...]
```

## Related

- `/generate-tests` - Generate tests for identified gaps
- `/generate-api-tests` - Generate API-specific tests
- `test-patterns` skill - Test pattern reference
