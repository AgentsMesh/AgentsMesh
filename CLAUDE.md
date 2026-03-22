# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AgentsMesh is a multi-tenant AI Code Agent collaboration platform supporting Claude Code, Codex CLI, Gemini CLI, Aider, and more. It consists of four main components:

- **Backend**: Go API server (Gin + GORM)
- **Web**: Next.js frontend (App Router + TypeScript + Tailwind CSS)
- **Web-Admin**: Admin Console frontend (Next.js + Tailwind CSS) - internal management interface
- **Runner**: Go daemon that executes AI agent tasks in isolated PTY environments

## Development Environment (Docker)

**Always use `deploy/dev` Docker environment for development and debugging.** This setup includes Traefik reverse proxy and mirrors production architecture, helping catch issues early.

### Quick Start (Recommended)

```bash
cd deploy/dev
./dev.sh               # One-click: starts Docker backend + local frontend
./dev.sh --clean       # Stop and clean up everything
```

This script automatically:
1. Generates `.env` with worktree-isolated ports (supports multiple worktrees)
2. Starts Docker backend services (PostgreSQL, Redis, MinIO, Backend, Traefik, Runner)
3. Runs database migrations and seeds test data
4. Starts local frontend (Next.js with Turbopack) for better performance

### Services & Ports (main worktree)

| Service | URL | Description |
|---------|-----|-------------|
| **Frontend** | http://localhost:3000 | Next.js (local, Turbopack) |
| **API** | http://localhost:80/api | Backend API via Traefik |
| **Traefik Dashboard** | http://localhost:8080 | Traefik dashboard (dev only) |
| **Adminer** | http://localhost:8081 | Database management UI |
| **MinIO Console** | http://localhost:9001 | S3-compatible storage UI |
| PostgreSQL | localhost:5432 | Database (user: agentsmesh, pass: agentsmesh_dev) |
| Redis | localhost:6379 | Cache |

Test accounts:
- **User**: dev@agentsmesh.local / devpass123
- **Admin**: admin@agentsmesh.local / adminpass123

> **Note**: All ports are dynamically allocated based on worktree. Check `deploy/dev/.env` for actual ports (e.g., WEB_PORT, HTTP_PORT).

### Logs

```bash
tail -f deploy/dev/web.log       # Frontend logs
docker compose logs -f backend   # Backend logs
docker compose logs -f runner    # Runner logs
```

### Hot Reload

- **Frontend**: Next.js Turbopack fast refresh (local)
- **Backend**: Go code auto-rebuild via Air (Docker)
- **Runner**: Go code auto-rebuild (Docker)

## Build Commands (for CI/testing outside Docker)

### Backend (Go)

```bash
cd backend
go build ./cmd/server            # Build binary
go test ./...                    # Run all tests
go test -v ./internal/service/... -run TestAuth  # Run specific test
```

### Web (Next.js)

```bash
cd web
pnpm install                     # Install dependencies
pnpm build                       # Production build
pnpm lint                        # ESLint
pnpm test                        # Run tests (Vitest)
pnpm test:run                    # Run tests once
pnpm test:coverage               # Test coverage
```

### Web-Admin (Next.js)

```bash
cd web-admin
pnpm install                     # Install dependencies
pnpm build                       # Production build
pnpm lint                        # ESLint
pnpm dev                         # Start development server
```

### Runner (Go)

```bash
cd runner
make build                       # Build binary (no CGO required)
make test                        # Run tests
make lint                        # golangci-lint
make build-all                   # Cross-platform builds
```

### Database Migrations

Migrations are located in `backend/migrations/` using golang-migrate format.

**Development** (via Docker):
```bash
cd deploy/dev
./dev.sh               # Automatically runs all migrations
```

**Production** (via backend container):
```bash
# Inside the backend container, golang-migrate is pre-installed
migrate -path /app/migrations -database "postgres://user:pass@host:5432/db?sslmode=disable" up
migrate -path /app/migrations -database "postgres://user:pass@host:5432/db?sslmode=disable" down 1
migrate -path /app/migrations -database "postgres://user:pass@host:5432/db?sslmode=disable" version
```

**Create new migration**:
```bash
# Install golang-migrate locally
brew install golang-migrate

# Create migration files
migrate create -ext sql -dir backend/migrations -seq add_new_feature
# This creates: 000024_add_new_feature.up.sql and 000024_add_new_feature.down.sql
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Web (Next.js)                            │
│                 localhost:3000                              │
└─────────────────────────────────────────────────────────────┘
                              │
                        REST / WebSocket
                         (terminal/events)
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Backend (Go + Gin)                        │
│            REST: localhost:8080 | gRPC: localhost:9443      │
│  - Auth (JWT + OAuth)                                       │
│  - Organization/Team/User management                        │
│  - Pod lifecycle management                                 │
│  - Ticket/Channel management                                │
│  - PostgreSQL + Redis                                       │
│  - PKI: Runner certificate issuance & revocation            │
└─────────────────────────────────────────────────────────────┘
                              │
                      gRPC + mTLS (port 9443)
                   (bidirectional streaming)
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Runner (Go daemon)                        │
│              Self-hosted by users                           │
│  - Connects via gRPC with mTLS client certificate           │
│  - Creates isolated PTY terminals (Pods)                    │
│  - Executes AI agents (Claude Code, Aider, etc.)            │
│  - Streams terminal output back to server                   │
│  - Auto certificate renewal before expiry                   │
└─────────────────────────────────────────────────────────────┘
```

## Backend Structure

```
backend/
├── cmd/server/           # Entry point
├── internal/
│   ├── api/rest/         # REST API handlers
│   │   └── v1/admin/     # Admin API handlers
│   ├── domain/           # Domain models (DDD style)
│   │   ├── user/         # User entity (includes is_system_admin)
│   │   ├── organization/ # Organization entity
│   │   ├── agentpod/     # AgentPod entity
│   │   ├── agent/        # Agent configuration entity
│   │   ├── ticket/       # Ticket entity
│   │   ├── channel/      # Channel entity
│   │   ├── runner/       # Runner entity
│   │   ├── billing/      # Billing/subscription entity
│   │   ├── invitation/   # Organization invitation
│   │   ├── promocode/    # Promo code entity
│   │   ├── gitprovider/  # Git provider (OAuth) entity
│   │   ├── repository/   # Repository entity
│   │   ├── mesh/         # Mesh topology entity
│   │   ├── file/         # File storage entity
│   │   └── admin/        # Admin audit log entity
│   ├── service/          # Business logic layer
│   │   └── admin/        # Admin service (dashboard, user/org management)
│   ├── infra/            # Infrastructure (DB, cache)
│   ├── config/           # Configuration loading (includes AdminConfig)
│   └── middleware/       # Auth, tenant isolation, AdminMiddleware
├── pkg/                  # Shared packages
│   ├── auth/             # JWT and OAuth utilities
│   ├── crypto/           # Encryption utilities
│   ├── i18n/             # Internationalization
│   └── audit/            # Audit logging
└── migrations/           # SQL migrations
```

## Web Structure

```
web/src/
├── app/                  # Next.js App Router
│   ├── (auth)/           # Auth pages (login, register)
│   ├── (dashboard)/      # Dashboard pages
│   └── api/              # API routes
├── components/           # React components
├── lib/                  # Utilities, API clients
├── stores/               # Zustand state stores
├── hooks/                # Custom React hooks
├── messages/             # i18n translations (next-intl)
└── providers/            # Context providers
```

## Web-Admin Structure (Admin Console)

```
web-admin/src/
├── app/                  # Next.js App Router (basePath: /admin)
│   ├── login/            # GitLab SSO login page
│   ├── auth/callback/    # OAuth callback handler
│   └── (dashboard)/      # Dashboard pages (protected)
│       ├── users/        # User management
│       ├── organizations/ # Organization management
│       ├── runners/      # Runner management
│       └── audit-logs/   # Audit log viewer
├── components/
│   ├── ui/               # Shadcn-style UI components
│   └── layout/           # Sidebar, Header
├── lib/
│   ├── api/              # Admin API client
│   └── utils.ts          # Utility functions
└── stores/
    └── auth.ts           # Zustand auth store (persist to localStorage)
```

## Runner Structure

```
runner/
├── cmd/runner/           # Entry point (register/run/service)
├── internal/
│   ├── runner/           # Core runner logic
│   │   ├── runner.go         # Main Runner struct
│   │   ├── pod_builder.go    # Builder pattern for Pods
│   │   ├── pod_store.go      # Pod storage
│   │   ├── message_handler.go # gRPC message routing
│   │   └── pty_forwarder.go  # Terminal output forwarding
│   ├── client/           # gRPC client (mTLS)
│   │   ├── grpc_connection.go   # gRPC bidirectional stream
│   │   ├── grpc_registration.go # Certificate registration
│   │   └── protocol.go          # Message types
│   ├── terminal/         # PTY management (creack/pty)
│   ├── process/          # Process management
│   ├── sandbox/          # Sandbox environment
│   │   └── plugins/      # worktree, tempdir plugins
│   ├── mcp/              # Model Context Protocol integration
│   ├── workspace/        # Git worktree management
│   └── console/          # Console UI
```

## Key Concepts

**Pod**: An isolated execution environment with PTY terminal, sandbox config, and output forwarder.

**Runner**: Self-hosted daemon that connects to backend via gRPC+mTLS, receives tasks, and manages Pod lifecycle.

**Sandbox**: Configurable environment created by plugins (worktree for Git isolation, tempdir for temporary workspace).

**Channel**: Multi-agent collaboration space where agents can communicate.

**Ticket**: Task management unit with kanban board integration.

## Message Flow (Runner ↔ Backend)

1. Runner registers via gRPC, receives mTLS certificate from PKI
2. Runner connects via gRPC bidirectional stream with mTLS
3. Backend sends `create_pod` → Runner creates Sandbox → Starts PTY/ACP process
4. Backend sends `subscribe_pod` → Runner connects to Relay WebSocket
5. Terminal I/O (data plane): Browser ↔ Relay ↔ Runner (WebSocket binary protocol)
6. Control commands (control plane): Backend → Runner via gRPC (`terminate_pod`, `send_prompt`, etc.)
7. Runner events → Backend via gRPC (`pod_created`, `pod_terminated`, `agent_status`, etc.)
8. Certificate auto-renewal before expiry (checked every hour)

## Configuration

**Development** (Docker): Run `cd deploy/dev && ./dev.sh` - auto-generates all configs

**Runner**: `~/.agentsmesh/config.yaml` (created after `runner register`)

## Testing Patterns

- Backend: Standard Go testing with `testify`
- Web: Vitest + Testing Library
- Runner: Go testing, files ending with `_integration_test.go` for integration tests

## Admin Console

The Admin Console (`web-admin`) is an internal management interface for system administrators.

### Access Control

- **Authentication**: Email + Password login (same as main app)
- **Authorization**: `is_system_admin` flag on user record must be `true`
- **Audit Logging**: All admin actions are logged to `system_admin_audit_logs` table

### Features

- **Dashboard**: System statistics (users, organizations, runners, pods)
- **User Management**: View, disable/enable users, grant/revoke admin privileges
- **Organization Management**: View, update, delete organizations
- **Runner Management**: View, disable/enable, delete runners
- **Audit Logs**: View all admin actions with filtering

### Configuration

Admin Console is enabled by default. All components use unified domain configuration:

```bash
# All components use the same two variables (Backend, Relay, Web, Web-Admin)
PRIMARY_DOMAIN=localhost:10000                  # Primary domain for all URLs
USE_HTTPS=false                                 # Use HTTPS/WSS protocols

# Backend-specific
ADMIN_ENABLED=true                              # Enable admin console (default: true)
```

### Creating Admin Users

To grant admin privileges to a user, set `is_system_admin = true` in the database:

```sql
UPDATE users SET is_system_admin = true WHERE email = 'admin@example.com';
```

Or use an existing admin to grant privileges via the Admin Console UI.


## Principles
* Architecture must conform to SOLID, GRASP, and YAGNI.
* **Hard limit: every file must stay under 200 lines** (excluding test files, which should stay under 400 lines). When a file approaches this limit, proactively split it by SRP — extract types, helpers, hooks, or sub-components into separate files. A 210-line file is acceptable if splitting would break cohesion; a 300+ line file is never acceptable and must be split before committing. 