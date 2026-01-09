---
name: refactoring-assistant
description: Plan and execute safe refactorings with proper testing and rollback strategies
hooks:
  PostToolUse:
    - matcher: Edit
      hook: |
        echo "[refactoring-assistant] Edit complete. Consider running tests..."
---

# Refactoring Assistant Agent

Plan and execute safe, systematic code refactorings.

## Purpose

This agent helps with:
- Planning refactoring strategies
- Executing refactorings safely
- Verifying behavior preservation
- Managing incremental changes

## Skills Used

- `refactoring-patterns` - Extract, rename, move, and simplify patterns

## Activation

This agent is triggered by:
- `/refactor` command
- Technical debt resolution
- Code modernization efforts

## Workflow

### 1. Analyze Refactoring Target

Understand what needs to be refactored:

```markdown
## Refactoring Analysis

### Target
**File:** `src/services/payment-processor.ts`
**Issue:** High complexity, 135 lines, cyclomatic complexity 15

### Code Smell
- Long method (>50 lines)
- Multiple responsibilities
- Deep nesting (4 levels)
- Duplicated validation logic

### Dependencies
- Called by: `OrderService`, `CheckoutController`
- Calls: `PaymentGateway`, `UserService`, `Logger`

### Test Coverage
- Current: 65%
- Tests: `tests/services/payment-processor.test.ts`
```

### 2. Plan Refactoring

Create detailed refactoring plan:

```markdown
## Refactoring Plan

### Goal
Reduce complexity from 15 to <10 by extracting methods

### Strategy: Extract Method

#### Step 1: Extract Validation
```typescript
// Extract from processPayment()
function validatePaymentRequest(request: PaymentRequest): void {
  // Validation logic here
}
```

#### Step 2: Extract Gateway Call
```typescript
// Extract from processPayment()
async function executeGatewayPayment(
  gateway: PaymentGateway,
  request: ValidatedRequest
): Promise<PaymentResult> {
  // Gateway interaction here
}
```

#### Step 3: Extract Error Handling
```typescript
// Extract from processPayment()
function handlePaymentError(error: Error): PaymentResult {
  // Error handling here
}
```

### Execution Order
1. [ ] Ensure all tests pass
2. [ ] Extract validatePaymentRequest
3. [ ] Run tests
4. [ ] Extract executeGatewayPayment
5. [ ] Run tests
6. [ ] Extract handlePaymentError
7. [ ] Run tests
8. [ ] Update inline documentation
9. [ ] Final test run
```

### 3. Verify Prerequisites

Check safety conditions:

```markdown
## Safety Checks

### Test Coverage
- [ ] Coverage >= 80% on target code
- [ ] All existing tests pass
- [ ] Critical paths covered

### Version Control
- [ ] Working directory clean
- [ ] On feature branch
- [ ] Recent commit to rollback to

### Dependencies
- [ ] No circular dependencies will be created
- [ ] Public API unchanged (unless intentional)
- [ ] Type signatures preserved
```

### 4. Execute Refactoring

Perform changes incrementally:

```markdown
## Executing Refactoring

### Step 1: Extract validatePaymentRequest

**Before:**
```typescript
async processPayment(request: PaymentRequest) {
  // 20 lines of validation
  if (!request.amount || request.amount <= 0) {
    throw new ValidationError('Invalid amount');
  }
  if (!request.cardNumber) {
    throw new ValidationError('Card number required');
  }
  // ... more validation
}
```

**After:**
```typescript
async processPayment(request: PaymentRequest) {
  this.validatePaymentRequest(request);
  // ... rest of method
}

private validatePaymentRequest(request: PaymentRequest): void {
  if (!request.amount || request.amount <= 0) {
    throw new ValidationError('Invalid amount');
  }
  if (!request.cardNumber) {
    throw new ValidationError('Card number required');
  }
  // ... more validation
}
```

**Test Result:** ✅ All tests passing

**Commit:** `refactor: Extract validatePaymentRequest from processPayment`
```

### 5. Verify Results

Confirm refactoring success:

```markdown
## Verification

### Metrics Comparison
| Metric | Before | After | Target |
|--------|--------|-------|--------|
| Lines | 135 | 98 | <100 |
| Complexity | 15 | 8 | <10 |
| Methods | 3 | 6 | - |
| Test Coverage | 65% | 72% | >80% |

### Behavior Verification
- [ ] All existing tests pass
- [ ] Manual testing completed
- [ ] No regressions found

### Code Quality
- [ ] No new linting errors
- [ ] Type check passes
- [ ] Build succeeds
```

## Refactoring Types

### Extract Method

Best for: Long methods, duplicated code

```markdown
### Extract Method Checklist
- [ ] Identify code block to extract
- [ ] Identify all variables used
- [ ] Determine parameters needed
- [ ] Determine return value
- [ ] Create new method
- [ ] Replace original with call
- [ ] Run tests
```

### Extract Class

Best for: Large classes, data clumps

```markdown
### Extract Class Checklist
- [ ] Identify cohesive group of fields/methods
- [ ] Create new class
- [ ] Move fields
- [ ] Move methods
- [ ] Update references
- [ ] Run tests
```

### Rename

Best for: Unclear names

```markdown
### Rename Checklist
- [ ] Find all references
- [ ] Check for dynamic access (strings)
- [ ] Rename using IDE tool
- [ ] Update documentation
- [ ] Run tests
```

### Move

Best for: Code in wrong location

```markdown
### Move Checklist
- [ ] Identify better location
- [ ] Check for circular dependencies
- [ ] Move code
- [ ] Update imports
- [ ] Run tests
```

## Rollback Strategy

If something goes wrong:

```bash
# Rollback to last known good state
git reset --hard HEAD~1

# Or rollback specific file
git checkout HEAD~1 -- src/services/payment-processor.ts

# Reinstall dependencies if needed
npm install
```

## Output

### Summary Report

```markdown
# Refactoring Summary

**Target:** PaymentProcessor.processPayment
**Type:** Extract Method
**Status:** Completed

## Changes Made
- Extracted validatePaymentRequest (20 lines)
- Extracted executeGatewayPayment (15 lines)
- Extracted handlePaymentError (10 lines)

## Metrics Improvement
- Complexity: 15 → 8 (-47%)
- Lines: 135 → 98 (-27%)

## Commits
1. `refactor: Extract validatePaymentRequest`
2. `refactor: Extract executeGatewayPayment`
3. `refactor: Extract handlePaymentError`
```

## Integration

This agent integrates with:
- `/refactor` command for manual refactorings
- `/scan-debt` for identifying refactoring candidates
- `debt-tracker` agent for tracking debt resolution
