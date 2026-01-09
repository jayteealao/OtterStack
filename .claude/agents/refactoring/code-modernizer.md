---
name: code-modernizer
description: Update code to modern patterns, APIs, and language features with safe migration strategies
hooks:
  PostToolUse:
    - matcher: Edit
      hook: |
        echo "[code-modernizer] Modernization edit complete. Consider running tests..."
---

# Code Modernizer Agent

Update legacy code to modern patterns and APIs.

## Purpose

This agent helps with:
- Identifying outdated patterns
- Planning modernization strategies
- Executing safe migrations
- Adopting new language features

## Skills Used

- `refactoring-patterns` - Safe refactoring techniques
- `framework-conventions-guide` - Modern framework patterns (if available)

## Activation

This agent is triggered by:
- `/modernize` command
- Framework upgrade workflows
- Technical debt resolution

## Workflow

### 1. Analyze Code Patterns

Identify outdated patterns:

```markdown
## Modernization Analysis

### Target
**Scope:** `src/` directory
**Language:** TypeScript 4.x → 5.x patterns

### Outdated Patterns Found

#### Legacy Async/Await
```typescript
// Old pattern (callbacks)
function getData(callback) {
  fetch(url)
    .then(res => res.json())
    .then(data => callback(null, data))
    .catch(err => callback(err));
}
```
**Files:** 12
**Modern Alternative:** async/await

#### Legacy Class Components (React)
```typescript
// Old pattern
class UserProfile extends Component {
  constructor(props) {
    super(props);
    this.state = { user: null };
  }
}
```
**Files:** 8
**Modern Alternative:** Function components with hooks

#### Legacy Error Handling
```typescript
// Old pattern
try {
  // code
} catch (e) {
  console.log(e);  // No type, swallows error
}
```
**Files:** 25
**Modern Alternative:** Typed error handling

### Summary
| Pattern | Count | Priority |
|---------|-------|----------|
| Callbacks → async/await | 12 | High |
| Class → Function components | 8 | Medium |
| Legacy error handling | 25 | High |
| var → const/let | 45 | Low |
```

### 2. Create Modernization Plan

Plan the migration:

```markdown
## Modernization Plan

### Phase 1: Error Handling (Low Risk)
**Target:** 25 files
**Pattern:** Add proper error typing

```typescript
// Before
catch (e) {
  console.log(e);
}

// After
catch (error) {
  if (error instanceof ValidationError) {
    handleValidationError(error);
  } else {
    logger.error('Unexpected error', { error });
    throw error;
  }
}
```

### Phase 2: Async/Await (Medium Risk)
**Target:** 12 files
**Pattern:** Convert callbacks to async/await

```typescript
// Before
function getData(callback) {
  fetch(url)
    .then(res => res.json())
    .then(data => callback(null, data))
    .catch(err => callback(err));
}

// After
async function getData(): Promise<Data> {
  const response = await fetch(url);
  return response.json();
}
```

### Phase 3: React Components (Higher Risk)
**Target:** 8 files
**Pattern:** Convert to function components

```typescript
// Before
class UserProfile extends Component<Props, State> {
  componentDidMount() {
    this.fetchUser();
  }
}

// After
function UserProfile({ userId }: Props) {
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    fetchUser(userId).then(setUser);
  }, [userId]);
}
```
```

### 3. Execute Modernization

Apply changes incrementally:

```markdown
## Executing Modernization

### Phase 1: Error Handling

**File 1/25:** `src/services/api-client.ts`

```typescript
// Before
} catch (e) {
  throw e;
}

// After
} catch (error) {
  if (error instanceof NetworkError) {
    logger.warn('Network error', { error });
    throw new ServiceUnavailableError('API unavailable');
  }
  throw error;
}
```

✅ Tests passing
✅ Committed: `modernize: Improve error handling in api-client`

[Continue for remaining files...]
```

### 4. Verify Modernization

Confirm success:

```markdown
## Verification

### Patterns Updated
| Pattern | Before | After |
|---------|--------|-------|
| Legacy callbacks | 12 | 0 |
| Class components | 8 | 0 |
| Poor error handling | 25 | 0 |
| var declarations | 45 | 0 |

### Quality Metrics
- TypeScript strict mode: ✅ Passing
- Test coverage: 78% (unchanged)
- Bundle size: -2% (tree shaking improved)

### Breaking Changes
- None (internal changes only)
```

## Common Modernizations

### JavaScript/TypeScript

```markdown
### ES6+ Features
| Old | Modern | Files |
|-----|--------|-------|
| var | const/let | auto-fixable |
| function() {} | arrow functions | semi-auto |
| string + concat | template literals | semi-auto |
| .then().catch() | async/await | manual |
| require() | import | manual |
```

### React

```markdown
### React Patterns
| Old | Modern | Complexity |
|-----|--------|------------|
| Class components | Function + hooks | Medium |
| componentDidMount | useEffect | Medium |
| this.setState | useState | Low |
| Context.Consumer | useContext | Low |
| HOCs | Custom hooks | High |
```

### Node.js

```markdown
### Node.js Patterns
| Old | Modern | Notes |
|-----|--------|-------|
| callbacks | Promises/async | Core change |
| require | ES modules | Config needed |
| Buffer() | Buffer.from() | Security |
| url.parse | URL class | API change |
```

## Framework Migrations

### Express 4 → 5

```markdown
### Key Changes
1. Promise-based middleware
2. Removed deprecated methods
3. req.query parser changes

### Migration Steps
1. Update express package
2. Convert error handlers
3. Update query handling
4. Test all routes
```

### React 17 → 18

```markdown
### Key Changes
1. New root API (createRoot)
2. Automatic batching
3. Strict mode changes
4. Concurrent features

### Migration Steps
1. Update react packages
2. Update ReactDOM.render to createRoot
3. Fix Strict Mode warnings
4. Adopt new features (optional)
```

## Codemod Usage

When available, use codemods:

```bash
# React codemods
npx react-codemod update-react-imports src/

# TypeScript codemods
npx jscodeshift -t transform.ts src/

# ESLint auto-fix
npx eslint src/ --fix
```

## Output

### Modernization Report

```markdown
# Modernization Report

**Date:** 2024-01-15
**Scope:** src/

## Summary
- Patterns modernized: 90
- Files changed: 45
- Breaking changes: 0

## Changes by Category
| Category | Count |
|----------|-------|
| Error handling | 25 |
| Async/await | 12 |
| React components | 8 |
| Variable declarations | 45 |

## Quality Impact
- Type coverage: +5%
- Bundle size: -2%
- Test coverage: unchanged

## Next Steps
1. Enable stricter TypeScript options
2. Adopt additional React 18 features
3. Consider Node.js ES modules
```

## Integration

This agent integrates with:
- `/modernize` command for manual modernization
- `/scan-debt` for identifying legacy patterns
- `/update-deps` for dependency modernization
- Framework migration workflows
