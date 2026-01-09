---
name: generate-onboarding
description: Generate onboarding documentation for new developers
argument-hint: "[optional: specific area to document]"
---

# Generate Onboarding Command

Create comprehensive onboarding documentation for new team members.

## Usage

```
/generate-onboarding [area]
```

**Arguments:**
- `area` (optional) - Specific area to document: `architecture`, `setup`, `tour`, `decisions`

## Examples

```bash
# Generate all onboarding docs
/generate-onboarding

# Generate only architecture docs
/generate-onboarding architecture

# Generate only setup guide
/generate-onboarding setup

# Generate code tour
/generate-onboarding tour

# Generate decision records
/generate-onboarding decisions
```

## Workflow

### 1. Project Analysis

First, analyze the project structure:

```markdown
## Project Analysis

### Detected Stack
- **Language:** TypeScript
- **Framework:** Express.js
- **Database:** PostgreSQL
- **ORM:** Prisma
- **Testing:** Jest

### Structure Analysis
```
src/
├── api/           # 12 route handlers
├── services/      # 8 service classes
├── models/        # 6 data models
├── lib/           # 15 utility modules
└── config/        # Configuration
```

### Dependencies
- 45 production dependencies
- 23 development dependencies
- Key: express, prisma, jsonwebtoken
```

### 2. Generate Architecture Overview

Spawn `onboarding-generator` agent:

```markdown
## Architecture Documentation

Creating `docs/onboarding/architecture.md`:

### System Overview
- High-level description
- Component diagram
- Data flow

### Components
- API Layer documentation
- Service Layer documentation
- Data Layer documentation

### External Integrations
- Third-party services
- Authentication providers
- Data stores
```

### 3. Generate Setup Guide

Create step-by-step setup instructions:

```markdown
## Setup Guide

Creating `docs/onboarding/setup.md`:

### Prerequisites
- Node.js 18+
- Docker
- Git

### Quick Start (5 minutes)
1. Clone repository
2. Install dependencies
3. Configure environment
4. Start services
5. Verify setup

### Detailed Instructions
- Environment variables explained
- Database setup
- Running tests
- Common issues
```

### 4. Generate Code Tour

Create guided walkthrough:

```markdown
## Code Tour

Creating `docs/onboarding/code-tour.md`:

### Tour Stops
1. Entry Point (src/index.ts)
2. Route Definitions (src/api/routes.ts)
3. Sample Handler (src/api/users/get-user.ts)
4. Service Layer (src/services/user-service.ts)
5. Data Access (src/models/user.ts)
6. Configuration (src/config/index.ts)

### Additional Tours
- Authentication flow
- Error handling
- Testing patterns
```

### 5. Document Decisions

Create or organize ADRs:

```markdown
## Decision Records

Creating `docs/onboarding/decisions/`:

### Existing Decisions
| ID | Title | Status |
|----|-------|--------|
| 001 | TypeScript backend | Accepted |
| 002 | REST API design | Accepted |
| 003 | PostgreSQL database | Accepted |

### Generated Template
- ADR template for new decisions
- Index document
```

## Output Structure

```
docs/
└── onboarding/
    ├── README.md              # Welcome document
    ├── architecture.md        # System architecture
    ├── setup.md               # Development setup
    ├── code-tour.md           # Guided walkthrough
    ├── first-tasks.md         # First day tasks
    └── decisions/
        ├── README.md          # Decision index
        ├── template.md        # ADR template
        ├── 001-language.md    # Example ADR
        └── 002-api-design.md  # Example ADR
```

## Welcome Document Template

```markdown
# Welcome to [Project Name]

Welcome to the team! This guide will help you get started.

## Quick Links

| Document | Purpose |
|----------|---------|
| [Setup Guide](./setup.md) | Get your environment running |
| [Architecture](./architecture.md) | Understand the system |
| [Code Tour](./code-tour.md) | Navigate the codebase |
| [Decisions](./decisions/) | Why we built it this way |

## First Day Checklist

- [ ] Complete local setup
- [ ] Run the test suite
- [ ] Read architecture overview
- [ ] Complete code tour
- [ ] Make a small change

## Getting Help

- **Slack:** #dev-questions
- **On-call:** @oncall
- **Documentation issues:** File a PR
```

## Customization

Create `.onboarding.yaml` for customization:

```yaml
# .onboarding.yaml
project:
  name: My Project
  description: A brief description

output:
  directory: docs/onboarding

sections:
  architecture: true
  setup: true
  code_tour: true
  decisions: true
  first_tasks: true

contacts:
  slack: "#dev-questions"
  oncall: "@oncall-rotation"

custom_sections:
  - name: "Security Guidelines"
    file: "security.md"
```

## Verification

After generation:

```markdown
## Verification Checklist

- [ ] Setup guide tested on clean machine
- [ ] All code references valid
- [ ] Links not broken
- [ ] Contact info current
- [ ] Diagrams up-to-date
```

## Related

- `/document-api` - Generate API documentation
- `onboarding-docs` skill - Documentation patterns
- `onboarding-generator` agent - Generation agent
