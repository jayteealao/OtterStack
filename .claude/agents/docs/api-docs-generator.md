---
name: api-docs-generator
description: Generate comprehensive API documentation including OpenAPI specs, endpoint references, and usage examples
---

# API Documentation Generator Agent

Generate and maintain API documentation from code.

## Purpose

This agent analyzes API code to generate comprehensive documentation including:
- OpenAPI/Swagger specifications
- Endpoint reference documentation
- Request/response examples
- Authentication documentation

## Skills Used

- `api-documentation` - OpenAPI patterns and documentation standards

## Activation

This agent is triggered by:
- `/document-api` command
- Changes to API endpoints during `/workflows:work`

## Workflow

### 1. Discover API Endpoints

First, analyze the codebase to find all API endpoints:

```markdown
## Endpoint Discovery

Search for:
- Route definitions (Express, FastAPI, Rails, etc.)
- Controller classes
- OpenAPI decorators/annotations
- Existing OpenAPI specs

### Found Endpoints
| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/users | UsersController#index | JWT |
| POST | /api/users | UsersController#create | JWT |
```

### 2. Extract Endpoint Details

For each endpoint, extract:

```markdown
## Endpoint: GET /api/users

### Request
- **Parameters:**
  - `page` (query, optional): Page number, default 1
  - `limit` (query, optional): Items per page, default 20
  - `search` (query, optional): Search term

- **Headers:**
  - `Authorization`: Bearer token (required)

### Response
- **200 OK:**
  ```json
  {
    "data": [{ "id": 1, "email": "user@example.com" }],
    "pagination": { "page": 1, "total": 100 }
  }
  ```

- **401 Unauthorized:**
  ```json
  { "error": { "message": "Invalid token" } }
  ```
```

### 3. Generate OpenAPI Spec

Create or update OpenAPI specification:

```yaml
openapi: 3.0.3
info:
  title: API Name
  version: 1.0.0
  description: API description

servers:
  - url: https://api.example.com/v1
    description: Production
  - url: http://localhost:3000/v1
    description: Development

paths:
  /users:
    get:
      summary: List users
      operationId: listUsers
      tags:
        - Users
      parameters:
        - name: page
          in: query
          schema:
            type: integer
            default: 1
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserList'

components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        email:
          type: string
          format: email
```

### 4. Generate Examples

Create realistic request/response examples:

```markdown
## Example: Create User

### Request
```bash
curl -X POST https://api.example.com/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "name": "New User",
    "role": "member"
  }'
```

### Response
```json
{
  "data": {
    "id": 123,
    "email": "newuser@example.com",
    "name": "New User",
    "role": "member",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```
```

### 5. Document Authentication

```markdown
## Authentication

### Bearer Token

All API requests require authentication via Bearer token in the Authorization header:

```
Authorization: Bearer <your-token>
```

### Obtaining a Token

```bash
curl -X POST https://api.example.com/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-16T10:30:00Z"
}
```

### Token Expiration

Tokens expire after 24 hours. Use the refresh endpoint to obtain a new token:

```bash
curl -X POST https://api.example.com/v1/auth/refresh \
  -H "Authorization: Bearer <your-token>"
```
```

## Output

### File Locations

```
docs/
└── api/
    ├── openapi.yaml          # OpenAPI specification
    ├── README.md             # API overview
    ├── authentication.md     # Auth documentation
    └── endpoints/
        ├── users.md          # Users endpoints
        ├── products.md       # Products endpoints
        └── orders.md         # Orders endpoints
```

### Validation

After generating documentation:

1. Validate OpenAPI spec:
   ```bash
   npx @redocly/cli lint docs/api/openapi.yaml
   ```

2. Generate preview:
   ```bash
   npx @redocly/cli preview-docs docs/api/openapi.yaml
   ```

3. Test examples work:
   ```bash
   # Run example requests against local server
   ```

## Integration

This agent integrates with:
- `/document-api` command for manual triggers
- `/workflows:work` for automatic updates when endpoints change
- `api-test-generator` agent for generating tests from documentation
