---
name: modernize
description: Update code to modern patterns and APIs
argument-hint: "[area or file to modernize]"
hooks:
  PreToolUse:
    - matcher: Edit
      hook: |
        echo "[modernize] Verifying git status before edit..."
        git status --porcelain 2>/dev/null | head -5 || true
---

# Modernize Command

Update legacy code to modern patterns and language features.

## Usage

```
/modernize [target]
```

**Target:**
- File path: Modernize specific file
- Directory: Modernize all files in directory
- Pattern name: Apply specific modernization

## Examples

```bash
# Analyze entire codebase
/modernize

# Modernize specific file
/modernize src/services/user-service.ts

# Modernize directory
/modernize src/components/

# Apply specific pattern
/modernize async-await
/modernize react-hooks
/modernize error-handling
```

## Workflow

### 1. Pattern Detection

Scan for outdated patterns:

```markdown
## Modernization Scan

### Outdated Patterns Found

| Pattern | Count | Priority | Auto-fixable |
|---------|-------|----------|--------------|
| var declarations | 45 | Low | Yes |
| Callbacks | 12 | High | Partial |
| Class components | 8 | Medium | No |
| Legacy error handling | 25 | High | Partial |
| CommonJS require | 18 | Medium | Yes |

### By File
| File | Patterns | Priority |
|------|----------|----------|
| src/services/api.ts | 5 | High |
| src/utils/helpers.ts | 12 | Medium |
| src/components/User.tsx | 3 | Medium |
```

### 2. Prioritization

Recommend modernization order:

```markdown
## Modernization Plan

### Phase 1: Quick Wins (Auto-fixable)
**Effort:** 10 minutes
**Risk:** Low

```bash
# var → const/let
npx eslint src/ --fix --rule 'no-var: error'

# CommonJS → ES modules
npx jscodeshift -t esm-transform src/
```

### Phase 2: Error Handling
**Effort:** 2-3 hours
**Risk:** Medium

Files: 25
Pattern: Add typed error handling

### Phase 3: Async/Await
**Effort:** 3-4 hours
**Risk:** Medium

Files: 12
Pattern: Convert callbacks to async/await

### Phase 4: React Components
**Effort:** 4-6 hours
**Risk:** Higher

Files: 8
Pattern: Convert class to function components
```

### 3. Execute Modernization

Apply changes with verification:

```markdown
## Executing Modernization

### Phase 1: Quick Wins
```bash
# Auto-fix var declarations
npx eslint src/ --fix
# Result: 45 files updated
```
✅ Tests passing
✅ Committed: `modernize: Convert var to const/let`

### Phase 2: Error Handling
**File:** `src/services/api.ts`

```typescript
// Before
try {
  const data = await fetch(url);
} catch (e) {
  console.log(e);
}

// After
try {
  const data = await fetch(url);
} catch (error) {
  if (error instanceof NetworkError) {
    logger.warn('Network error', { url, error });
    throw new ServiceError('Service unavailable', { cause: error });
  }
  throw error;
}
```
✅ Tests passing
✅ Committed: `modernize: Improve error handling in api service`

[Continue with remaining files...]
```

### 4. Verify Results

Confirm modernization success:

```markdown
## Verification

### Patterns Eliminated
| Pattern | Before | After |
|---------|--------|-------|
| var declarations | 45 | 0 |
| Legacy callbacks | 12 | 0 |
| Poor error handling | 25 | 0 |
| Class components | 8 | 0 |

### Quality Metrics
| Metric | Before | After |
|--------|--------|-------|
| TypeScript strict | ❌ | ✅ |
| Test coverage | 78% | 80% |
| Bundle size | 245kb | 238kb |
| Load time | 2.1s | 1.9s |

### No Breaking Changes
- All tests passing
- API unchanged
- Types preserved
```

## Modernization Patterns

### JavaScript/TypeScript

```markdown
### Available Patterns

#### var-to-const
Convert var to const/let based on reassignment

#### async-await
Convert Promise chains and callbacks to async/await

#### template-literals
Convert string concatenation to template literals

#### arrow-functions
Convert function expressions to arrow functions

#### destructuring
Use destructuring for object/array access

#### optional-chaining
Convert nested null checks to ?.

#### nullish-coalescing
Convert || to ?? where appropriate
```

### React

```markdown
### Available Patterns

#### react-hooks
Convert class components to function components with hooks

#### use-effect
Convert lifecycle methods to useEffect

#### use-state
Convert this.state to useState

#### use-context
Convert Context.Consumer to useContext

#### memo-components
Add React.memo to pure components
```

### Node.js

```markdown
### Available Patterns

#### esm-imports
Convert require() to ES module imports

#### async-fs
Convert callback fs to fs/promises

#### url-class
Convert url.parse to URL class

#### buffer-from
Convert new Buffer() to Buffer.from()
```

## Configuration

```yaml
# .modernize.yaml
patterns:
  enable:
    - var-to-const
    - async-await
    - react-hooks
  disable:
    - optional-chaining  # Node 12 support needed

options:
  auto_fix: true
  commit_each: true
  run_tests: true

typescript:
  strict: true
  target: ES2022
```

## Output

### Modernization Report

```markdown
# Modernization Report

**Date:** 2024-01-15
**Scope:** src/

## Summary
| Metric | Value |
|--------|-------|
| Files scanned | 156 |
| Files updated | 78 |
| Patterns fixed | 90 |
| Time taken | 45 min |

## Patterns Applied
| Pattern | Count |
|---------|-------|
| var → const/let | 45 |
| Async/await | 12 |
| Error handling | 25 |
| React hooks | 8 |

## Quality Impact
- TypeScript strict: Enabled
- Bundle size: -3%
- Test coverage: +2%

## Manual Review Needed
- src/legacy/old-module.ts (complex callbacks)
- src/components/DataGrid.tsx (complex state)
```

## Related

- `/refactor` - General refactoring
- `/scan-debt` - Find legacy patterns
- `/update-deps` - Update dependencies
- `refactoring-patterns` skill - Pattern reference
- `code-modernizer` agent - Modernization agent
