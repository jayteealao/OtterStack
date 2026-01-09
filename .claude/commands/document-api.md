---
name: document-api
description: Generate or update API documentation from code
argument-hint: "[optional: specific endpoint or controller]"
---

# Document API Command

Generate comprehensive API documentation from your codebase.

## Usage

```
/document-api [target]
```

**Arguments:**
- `target` (optional) - Specific endpoint, controller, or module to document

## Examples

```bash
# Document entire API
/document-api

# Document specific controller
/document-api UsersController

# Document specific endpoint
/document-api /api/users

# Document from OpenAPI spec
/document-api openapi.yaml
```

## Workflow

### 1. Discovery

The command first discovers API endpoints:

```markdown
## API Discovery

**Detected Framework:** Express.js

### Endpoints Found
| Method | Path | Handler | Documented |
|--------|------|---------|------------|
| GET | /api/users | listUsers | ❌ |
| POST | /api/users | createUser | ❌ |
| GET | /api/users/:id | getUser | ✅ |
| PUT | /api/users/:id | updateUser | ❌ |
| DELETE | /api/users/:id | deleteUser | ❌ |

**Summary:** 5 endpoints, 1 documented, 4 need documentation
```

### 2. Generate Documentation

Spawn the `api-docs-generator` agent to create:

```markdown
## Documentation Generation

### OpenAPI Spec
Creating `docs/api/openapi.yaml`:
- Endpoint definitions
- Request/response schemas
- Authentication requirements
- Error responses

### Endpoint Reference
Creating `docs/api/endpoints/`:
- Individual endpoint documentation
- Usage examples
- Error handling

### Authentication Guide
Creating `docs/api/authentication.md`:
- Auth methods supported
- Token lifecycle
- Example flows
```

### 3. Generate Examples

Create working examples for each endpoint:

```markdown
## Generated Examples

### GET /api/users

**cURL:**
```bash
curl https://api.example.com/v1/users \
  -H "Authorization: Bearer $TOKEN"
```

**JavaScript:**
```javascript
const response = await fetch('/api/users', {
  headers: { 'Authorization': `Bearer ${token}` }
});
const users = await response.json();
```

**Python:**
```python
response = requests.get(
    'https://api.example.com/v1/users',
    headers={'Authorization': f'Bearer {token}'}
)
users = response.json()
```
```

### 4. Validation

Validate the generated documentation:

```bash
# Validate OpenAPI spec
npx @redocly/cli lint docs/api/openapi.yaml

# Check for broken links
npx markdown-link-check docs/api/**/*.md

# Preview documentation
npx @redocly/cli preview-docs docs/api/openapi.yaml
```

## Output Structure

```
docs/
└── api/
    ├── openapi.yaml          # OpenAPI specification
    ├── README.md             # API overview
    ├── authentication.md     # Auth documentation
    ├── errors.md             # Error handling guide
    └── endpoints/
        ├── users.md          # User endpoints
        ├── products.md       # Product endpoints
        └── orders.md         # Order endpoints
```

## OpenAPI Output Format

```yaml
openapi: 3.0.3
info:
  title: Your API
  version: 1.0.0
  description: |
    API description with overview.

servers:
  - url: https://api.example.com/v1
    description: Production
  - url: http://localhost:3000/v1
    description: Development

tags:
  - name: Users
    description: User management endpoints

paths:
  /users:
    get:
      summary: List all users
      tags: [Users]
      security:
        - bearerAuth: []
      parameters:
        - $ref: '#/components/parameters/PageParam'
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserList'
```

## Configuration

Create `.api-docs.yaml` for customization:

```yaml
# .api-docs.yaml
output:
  directory: docs/api
  format: openapi

options:
  include_examples: true
  generate_curl: true
  generate_sdk_snippets: true

servers:
  - url: https://api.example.com/v1
    description: Production

authentication:
  type: bearer
  description: JWT token authentication
```

## Related

- `/generate-api-tests` - Generate tests from API documentation
- `/generate-onboarding` - Generate onboarding documentation
- `api-documentation` skill - API documentation patterns
- `api-docs-generator` agent - Documentation generation agent
