---
name: refactor
description: Plan and execute safe refactorings with proper testing
argument-hint: "[refactoring type: extract|rename|move|simplify]"
hooks:
  PreToolUse:
    - matcher: Edit
      hook: |
        echo "[refactor] Verifying git status before edit..."
        git status --porcelain 2>/dev/null | head -5 || true
---

# Refactor Command

Plan and execute safe code refactorings.

## Usage

```
/refactor [type] [target]
```

**Types:**
- `extract` - Extract method, class, or module
- `rename` - Rename symbol across codebase
- `move` - Move code between files/modules
- `simplify` - Reduce code complexity

**Target:** File path, function name, or class name

## Examples

```bash
# Interactive refactoring
/refactor

# Extract method from long function
/refactor extract src/services/payment.ts:processPayment

# Rename function across codebase
/refactor rename getUserName to fetchUserName

# Move class to new file
/refactor move UserValidator to src/validators/

# Simplify complex function
/refactor simplify src/utils/parser.ts:parseConfig
```

## Workflow

### 1. Safety Check

Verify prerequisites:

```markdown
## Safety Check

### Prerequisites
- [ ] Tests exist: ✅ 45 tests found
- [ ] Tests passing: ✅ All green
- [ ] Coverage: 78% (target: 80%)
- [ ] Working directory clean: ✅
- [ ] On feature branch: ✅ refactor/payment-processor

### Warnings
⚠️ Coverage below target (78% < 80%)
Consider adding tests before refactoring.

### Proceed? [Y/n]
```

### 2. Analysis

Analyze the refactoring target:

```markdown
## Refactoring Analysis

### Target
**Type:** Extract Method
**Location:** `src/services/payment.ts:45-120`
**Function:** `processPayment`

### Current State
- Lines: 75
- Cyclomatic complexity: 15
- Parameters: 3
- Dependencies: 5

### Identified Extractions
1. **validateRequest** (lines 48-65)
   - Purpose: Input validation
   - Variables: request, config
   - Returns: void (throws on error)

2. **executeGatewayCall** (lines 68-95)
   - Purpose: Payment gateway interaction
   - Variables: gateway, validatedRequest
   - Returns: Promise<PaymentResult>

3. **handleResponse** (lines 98-118)
   - Purpose: Response processing
   - Variables: result, logger
   - Returns: ProcessedPayment
```

### 3. Plan

Create refactoring plan:

```markdown
## Refactoring Plan

### Goal
Reduce complexity from 15 to <10

### Steps
1. [ ] Create backup point (commit current state)
2. [ ] Extract validateRequest
3. [ ] Run tests → verify passing
4. [ ] Extract executeGatewayCall
5. [ ] Run tests → verify passing
6. [ ] Extract handleResponse
7. [ ] Run tests → verify passing
8. [ ] Verify metrics improved
9. [ ] Create PR

### Estimated Impact
| Metric | Before | After (est) |
|--------|--------|-------------|
| Lines | 75 | 25 |
| Complexity | 15 | 5 |
| Methods | 1 | 4 |

### Risk Assessment
- Risk Level: Low
- Rollback: `git reset --hard HEAD~3`
```

### 4. Execute

Perform refactoring with verification:

```markdown
## Executing Refactoring

### Step 1: Backup
```bash
git stash  # If needed
git commit -m "checkpoint: before refactoring processPayment"
```

### Step 2: Extract validateRequest
```typescript
// Extracted method
private validateRequest(request: PaymentRequest): void {
  if (!request.amount || request.amount <= 0) {
    throw new ValidationError('Invalid amount');
  }
  if (!request.cardNumber) {
    throw new ValidationError('Card number required');
  }
  // ... remaining validation
}

// Updated processPayment
async processPayment(request: PaymentRequest) {
  this.validateRequest(request);
  // ... rest
}
```

**Tests:** ✅ All passing
**Commit:** `refactor: Extract validateRequest from processPayment`

### Step 3: Extract executeGatewayCall
[Similar process...]

### Step 4: Extract handleResponse
[Similar process...]
```

### 5. Verify

Confirm refactoring success:

```markdown
## Verification

### Metrics
| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Lines | 75 | 28 | -63% |
| Complexity | 15 | 6 | -60% |
| Methods | 1 | 4 | +3 |
| Coverage | 78% | 82% | +4% |

### Tests
- All 45 tests passing
- No behavior changes detected
- New tests added: 3

### Code Quality
- Lint: ✅ No new issues
- Types: ✅ All type-checked
- Build: ✅ Successful
```

## Refactoring Types

### Extract

```bash
# Extract method
/refactor extract src/file.ts:functionName

# Extract class
/refactor extract class OrderValidator from src/models/order.ts

# Extract module
/refactor extract module validation from src/utils.ts
```

### Rename

```bash
# Rename function
/refactor rename oldName to newName

# Rename class
/refactor rename class OldClass to NewClass

# Rename file
/refactor rename file old-name.ts to new-name.ts
```

### Move

```bash
# Move function to file
/refactor move validateUser to src/validators/

# Move class to module
/refactor move UserService to src/services/user/

# Move methods to class
/refactor move validate, sanitize to InputProcessor
```

### Simplify

```bash
# Simplify conditionals
/refactor simplify src/utils/parser.ts

# Replace nested conditionals with guard clauses
/refactor simplify guards src/services/auth.ts

# Replace conditional with polymorphism
/refactor simplify polymorphism src/handlers/
```

## Configuration

Create `.refactor.yaml` for preferences:

```yaml
# .refactor.yaml
safety:
  require_tests: true
  min_coverage: 80
  require_clean_working_dir: true

commit:
  create_commits: true
  prefix: "refactor:"

verification:
  run_tests: true
  run_lint: true
  run_build: true
```

## Output

### Summary Report

```markdown
# Refactoring Summary

**Type:** Extract Method
**Target:** processPayment
**Status:** ✅ Completed

## Changes
- Extracted: validateRequest, executeGatewayCall, handleResponse
- Files modified: 1
- Lines changed: -47

## Metrics
- Complexity: 15 → 6
- Coverage: 78% → 82%

## Commits
1. `refactor: Extract validateRequest`
2. `refactor: Extract executeGatewayCall`
3. `refactor: Extract handleResponse`
```

## Related

- `/modernize` - Modernize code patterns
- `/scan-debt` - Find refactoring candidates
- `refactoring-patterns` skill - Pattern reference
- `refactoring-assistant` agent - Refactoring agent
