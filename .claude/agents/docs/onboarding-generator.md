---
name: onboarding-generator
description: Generate comprehensive onboarding documentation for new developers including architecture overviews, setup guides, and code tours
---

# Onboarding Documentation Generator Agent

Create developer onboarding documentation from codebase analysis.

## Purpose

This agent analyzes a codebase to generate comprehensive onboarding documentation:
- Architecture overviews
- Local development setup guides
- Guided code tours
- Decision records (ADRs)

## Skills Used

- `onboarding-docs` - Documentation patterns and templates

## Activation

This agent is triggered by:
- `/generate-onboarding` command
- Initial project setup workflows

## Workflow

### 1. Analyze Project Structure

First, understand the project:

```markdown
## Project Analysis

### Technology Stack
- **Language:** TypeScript
- **Framework:** Express.js
- **Database:** PostgreSQL with Prisma
- **Testing:** Jest
- **Build:** esbuild

### Directory Structure
```
src/
├── api/           # REST endpoints
├── services/      # Business logic
├── models/        # Prisma models
├── lib/           # Utilities
└── config/        # Configuration
```

### Key Entry Points
| File | Purpose |
|------|---------|
| src/index.ts | Application entry |
| src/api/routes.ts | Route definitions |
| src/config/index.ts | Configuration |
```

### 2. Generate Architecture Overview

Create high-level architecture documentation:

```markdown
# System Architecture

## Overview

[Project Name] is a [type] application that [purpose].

## Component Diagram

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────>│ API Gateway │────>│  Services   │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                    ┌─────────────┐     ┌──────▼──────┐
                    │    Cache    │<────│  Database   │
                    └─────────────┘     └─────────────┘
```

## Components

### API Layer
- Handles HTTP requests
- Input validation
- Authentication/Authorization
- Location: `src/api/`

### Service Layer
- Business logic
- Data transformation
- External integrations
- Location: `src/services/`

### Data Layer
- Database operations
- Caching
- Location: `src/models/`
```

### 3. Generate Setup Guide

Create step-by-step setup instructions:

```markdown
# Local Development Setup

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Node.js | 18+ | [nodejs.org](https://nodejs.org) |
| Docker | 20+ | [docker.com](https://docker.com) |
| Git | 2.30+ | [git-scm.com](https://git-scm.com) |

## Quick Start

```bash
# Clone repository
git clone <repo-url>
cd <project>

# Install dependencies
npm install

# Setup environment
cp .env.example .env

# Start services
docker-compose up -d

# Run migrations
npm run db:migrate

# Start development
npm run dev
```

## Verify Setup

1. Open http://localhost:3000
2. Run `npm test` - all tests should pass
3. Try the health endpoint: `curl localhost:3000/health`

## Troubleshooting

### Port in use
```bash
lsof -i :3000
kill -9 <PID>
```

### Database connection failed
```bash
docker-compose logs postgres
docker-compose restart postgres
```
```

### 4. Generate Code Tour

Create guided walkthrough:

```markdown
# Code Tour: Your First Day

## Stop 1: Entry Point

**File:** `src/index.ts`

This is where the application starts. It:
1. Loads configuration
2. Connects to database
3. Starts the HTTP server

## Stop 2: Routes

**File:** `src/api/routes.ts`

All API routes are defined here. Each route maps to a handler function.

## Stop 3: A Typical Handler

**File:** `src/api/users/get-user.ts`

This handler demonstrates the pattern for all endpoints:
1. Validate input
2. Call service
3. Return response

## Stop 4: Service Layer

**File:** `src/services/user-service.ts`

Business logic lives in services. Services:
- Don't know about HTTP
- Use repository pattern for data
- Are easy to test

## Stop 5: Database Access

**File:** `src/models/user.ts`

Prisma models define our database schema. Migrations are in `prisma/migrations/`.
```

### 5. Document Key Decisions

Create or identify existing ADRs:

```markdown
# Architecture Decisions

## ADR-001: TypeScript for Backend

**Status:** Accepted

**Context:** Needed to choose a language for the backend.

**Decision:** Use TypeScript because:
- Type safety reduces bugs
- Team expertise
- Shared code with frontend

## ADR-002: REST over GraphQL

**Status:** Accepted

**Context:** Needed to choose API design.

**Decision:** Use REST because:
- Simpler for current needs
- Better caching
- Team familiarity
```

## Output

### Generated Files

```
docs/
└── onboarding/
    ├── README.md              # Welcome and navigation
    ├── architecture.md        # System architecture
    ├── setup.md               # Development setup
    ├── code-tour.md           # Guided walkthrough
    └── decisions/
        ├── README.md          # Decision index
        └── 001-typescript.md  # Example ADR
```

### Verification

After generation:

1. **Test setup guide:** Follow steps on clean machine
2. **Verify links:** Check all internal links work
3. **Validate diagrams:** Ensure diagrams are current
4. **Review with team:** Get feedback from recent hires

## Integration

This agent integrates with:
- `/generate-onboarding` command for manual generation
- `/workflows:plan` for including architecture context
- Existing documentation in `docs/` directory
