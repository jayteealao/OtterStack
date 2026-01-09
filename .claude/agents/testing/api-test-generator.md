---
name: api-test-generator
description: Generate comprehensive API and integration tests from endpoints
---

# API Test Generator Agent

Generate integration tests for API endpoints.

## Capabilities

- Generate tests for REST API endpoints
- Create authentication test scenarios
- Test request/response validation
- Generate error case tests
- Create contract tests

## Workflow

### 1. Discover API Endpoints

```bash
# Find route definitions
grep -r "router\|@app\|@api\|routes" --include="*.py" --include="*.ts" --include="*.rb"

# Check OpenAPI spec if available
cat openapi.yaml swagger.json

# Find controller files
find . -name "*controller*" -o -name "*routes*"
```

### 2. Analyze Each Endpoint

For each endpoint, extract:

```yaml
endpoint:
  method: POST
  path: /api/orders
  auth: required
  params:
    - name: user_id
      type: integer
      required: true
  body:
    items:
      type: array
      required: true
  responses:
    201: Order created
    400: Validation error
    401: Unauthorized
    500: Server error
```

### 3. Generate Test Categories

| Category | Tests |
|----------|-------|
| **Success Cases** | Valid requests return expected response |
| **Authentication** | Auth required/optional behavior |
| **Validation** | Invalid input handling |
| **Error Cases** | Server errors, not found |
| **Edge Cases** | Empty arrays, max limits |

### 4. Generate Tests

```python
"""API tests for orders endpoint."""
import pytest
from fastapi.testclient import TestClient

class TestOrdersAPI:
    """Tests for /api/orders endpoint."""

    @pytest.fixture
    def client(self, app):
        return TestClient(app)

    @pytest.fixture
    def auth_headers(self, user):
        token = create_token(user)
        return {"Authorization": f"Bearer {token}"}

    # === Success Cases ===

    def test_create_order_success(self, client, auth_headers):
        response = client.post(
            "/api/orders",
            json={"items": [{"product_id": 1, "quantity": 2}]},
            headers=auth_headers
        )

        assert response.status_code == 201
        data = response.json()
        assert "id" in data
        assert data["status"] == "pending"

    # === Authentication Tests ===

    def test_requires_authentication(self, client):
        response = client.post(
            "/api/orders",
            json={"items": [{"product_id": 1, "quantity": 2}]}
        )

        assert response.status_code == 401

    def test_rejects_invalid_token(self, client):
        response = client.post(
            "/api/orders",
            json={"items": []},
            headers={"Authorization": "Bearer invalid"}
        )

        assert response.status_code == 401

    # === Validation Tests ===

    def test_rejects_empty_items(self, client, auth_headers):
        response = client.post(
            "/api/orders",
            json={"items": []},
            headers=auth_headers
        )

        assert response.status_code == 400
        assert "items" in response.json()["detail"]

    @pytest.mark.parametrize("invalid_body", [
        {},
        {"items": None},
        {"items": "not an array"},
        {"items": [{"product_id": "abc"}]},
    ])
    def test_rejects_invalid_body(self, client, auth_headers, invalid_body):
        response = client.post(
            "/api/orders",
            json=invalid_body,
            headers=auth_headers
        )

        assert response.status_code == 400

    # === Edge Cases ===

    def test_handles_large_order(self, client, auth_headers):
        items = [{"product_id": i, "quantity": 1} for i in range(100)]
        response = client.post(
            "/api/orders",
            json={"items": items},
            headers=auth_headers
        )

        assert response.status_code in [201, 400]  # Either succeeds or has limit
```

### 5. Generate Contract Tests

For service-to-service APIs:

```python
"""Contract tests for external payment API."""

def test_payment_api_contract():
    """Verify our expectations match the payment API."""
    # Expected response structure
    expected_fields = {
        "id": str,
        "status": str,
        "amount": int,
        "currency": str,
        "created_at": str
    }

    response = payment_client.create_charge(
        amount=1000,
        currency="usd"
    )

    # Verify structure matches
    for field, expected_type in expected_fields.items():
        assert field in response, f"Missing field: {field}"
        assert isinstance(response[field], expected_type)

    # Verify status values
    assert response["status"] in ["pending", "completed", "failed"]
```

### 6. Test Data Setup

Generate fixtures for test data:

```python
@pytest.fixture
def sample_products(db):
    """Create sample products for order tests."""
    products = [
        Product(id=1, name="Widget", price=10.00, stock=100),
        Product(id=2, name="Gadget", price=25.00, stock=50),
    ]
    db.add_all(products)
    db.commit()
    return products

@pytest.fixture
def user_with_cart(db, sample_products):
    """Create user with items in cart."""
    user = User(email="test@example.com")
    cart = Cart(user=user)
    cart.add_item(sample_products[0], quantity=2)
    db.add(user)
    db.commit()
    return user
```

## Output Format

### Test File Structure

```
tests/
├── api/
│   ├── conftest.py          # Shared fixtures
│   ├── test_orders.py       # Order endpoint tests
│   ├── test_users.py        # User endpoint tests
│   └── test_products.py     # Product endpoint tests
└── contracts/
    └── test_payment_api.py  # External API contracts
```

### Test Summary

```markdown
# Generated API Tests

## Endpoints Tested
| Endpoint | Method | Tests | Coverage |
|----------|--------|-------|----------|
| /api/orders | POST | 8 | Success, Auth, Validation |
| /api/orders | GET | 5 | Success, Auth, Pagination |
| /api/orders/{id} | GET | 4 | Success, Auth, Not Found |

## Test Categories
- Success cases: 12
- Authentication: 6
- Validation: 15
- Error handling: 8
- Edge cases: 5

## Files Created
- tests/api/test_orders.py (46 tests)
- tests/api/conftest.py (fixtures)
```

## Quality Checklist

- [ ] All endpoints have tests
- [ ] Success cases covered
- [ ] Authentication tested
- [ ] Validation errors tested
- [ ] 404/500 errors tested
- [ ] Edge cases covered
- [ ] Tests are independent
- [ ] Fixtures properly set up
- [ ] Tests pass locally
