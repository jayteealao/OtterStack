---
name: generate-api-tests
description: Generate comprehensive API and integration tests from endpoints
argument-hint: "[endpoint path or controller]"
---

# Generate API Tests Command

Generate comprehensive API and integration tests from endpoint definitions.

## Usage

```
/generate-api-tests [target]
```

**Arguments:**
- `target` - Endpoint path, controller, or API module

## Examples

```bash
# Generate tests for specific endpoint
/generate-api-tests /api/users

# Generate tests for a controller
/generate-api-tests UsersController

# Generate tests for all endpoints in a module
/generate-api-tests src/api/

# Generate from OpenAPI spec
/generate-api-tests openapi.yaml
```

## Workflow

### 1. Discover Endpoints

Analyze the target to find all API endpoints:

```markdown
## API Endpoint Discovery

**Source:** [controller/routes/openapi]

### Endpoints Found
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /api/users | JWT | List users |
| POST | /api/users | JWT | Create user |
| GET | /api/users/:id | JWT | Get user |
| PUT | /api/users/:id | JWT | Update user |
| DELETE | /api/users/:id | Admin | Delete user |

### Request/Response Schemas
- Identified from types/interfaces
- Extracted from OpenAPI if available
- Inferred from code if necessary
```

### 2. Test Strategy

Plan comprehensive API tests:

```markdown
## API Test Strategy

### For Each Endpoint
1. **Happy Path Tests**
   - Valid request returns expected response
   - Correct status codes
   - Response schema validation

2. **Authentication Tests**
   - Missing token → 401
   - Invalid token → 401
   - Expired token → 401
   - Wrong role → 403

3. **Validation Tests**
   - Missing required fields → 400
   - Invalid field types → 400
   - Field length limits → 400

4. **Edge Cases**
   - Empty collections
   - Maximum payload sizes
   - Special characters in inputs

5. **Error Scenarios**
   - Resource not found → 404
   - Conflict (duplicate) → 409
   - Rate limiting → 429
   - Server errors → 500
```

### 3. Generate Tests

Use the `api-test-generator` agent:

```markdown
## Test Generation

### GET /api/users
```python
def test_list_users_returns_paginated_results():
    """List users endpoint returns paginated user list."""
    response = client.get("/api/users", headers=auth_headers)

    assert response.status_code == 200
    assert "data" in response.json()
    assert "pagination" in response.json()

def test_list_users_without_auth_returns_401():
    """List users requires authentication."""
    response = client.get("/api/users")

    assert response.status_code == 401

def test_list_users_with_invalid_token_returns_401():
    """List users rejects invalid tokens."""
    response = client.get("/api/users", headers={"Authorization": "Bearer invalid"})

    assert response.status_code == 401
```

### POST /api/users
```python
def test_create_user_with_valid_data_returns_201():
    """Create user with valid data succeeds."""
    user_data = UserFactory.build_dict()
    response = client.post("/api/users", json=user_data, headers=auth_headers)

    assert response.status_code == 201
    assert response.json()["email"] == user_data["email"]

def test_create_user_without_email_returns_400():
    """Create user requires email field."""
    user_data = {"name": "Test User"}
    response = client.post("/api/users", json=user_data, headers=auth_headers)

    assert response.status_code == 400
    assert "email" in response.json()["errors"]
```
```

### 4. Create Test Fixtures

```markdown
## Test Fixtures

### Factory Classes
```python
class UserFactory:
    @staticmethod
    def build(**kwargs):
        defaults = {
            "name": fake.name(),
            "email": fake.email(),
            "role": "user"
        }
        return User(**{**defaults, **kwargs})

    @staticmethod
    def build_dict(**kwargs):
        user = UserFactory.build(**kwargs)
        return user.to_dict()
```

### Auth Helpers
```python
@pytest.fixture
def auth_headers():
    token = create_test_token(user_id=1, role="user")
    return {"Authorization": f"Bearer {token}"}

@pytest.fixture
def admin_headers():
    token = create_test_token(user_id=1, role="admin")
    return {"Authorization": f"Bearer {token}"}
```
```

### 5. Run and Verify

```bash
# Run API tests
pytest tests/api/ -v

# Run with coverage
pytest tests/api/ --cov=src/api

# Run specific endpoint tests
pytest tests/api/test_users.py -v
```

## Test Organization

```
tests/
├── api/
│   ├── conftest.py          # Shared fixtures
│   ├── test_users.py        # /api/users tests
│   ├── test_products.py     # /api/products tests
│   └── test_orders.py       # /api/orders tests
├── fixtures/
│   ├── factories.py         # Data factories
│   └── auth.py              # Auth helpers
└── integration/
    └── test_workflows.py    # Multi-endpoint flows
```

## Output

Generated tests follow project structure with:
- Individual test files per endpoint/controller
- Shared fixtures in conftest.py
- Factory classes for test data
- Auth helper fixtures

## Related

- `/generate-tests` - Generate unit tests
- `/analyze-coverage` - Identify coverage gaps
- `/document-api` - Generate API documentation
- `test-patterns` skill - Test pattern reference
- `api-documentation` skill - API documentation standards
