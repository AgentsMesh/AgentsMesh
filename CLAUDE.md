# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AgentsMesh is a multi-tenant AI Code Agent collaboration platform supporting Claude Code, Codex CLI, Gemini CLI, Aider, and more. It consists of three main components:

- **Backend**: Go API server (Gin + GORM)
- **Web**: Next.js frontend (App Router + TypeScript + Tailwind CSS)
- **Runner**: Go daemon that executes AI agent tasks in isolated PTY environments

## Development Environment (Docker)

**Always use `deploy/dev` Docker environment for development and debugging.** This setup includes Nginx reverse proxy and mirrors production architecture, helping catch issues early.

### Quick Start (Recommended)

```bash
cd deploy/dev
./init-worktree.sh               # One-click setup: generates .env, starts services, runs migrations, seeds data
```

This script automatically:
1. Generates `.env` with worktree-isolated ports (supports multiple worktrees)
2. Starts all Docker services
3. Runs database migrations
4. Initializes seed data (test account: dev@agentsmesh.local / devpass123)

### Manual Commands

```bash
cd deploy/dev
docker compose up -d             # Start all services
docker compose logs -f           # View logs
docker compose down              # Stop all services
docker compose down -v           # Stop and remove volumes
./init-worktree.sh --clean       # Clean up environment (when things go wrong)
```

### Services & Ports

| Service | URL | Description |
|---------|-----|-------------|
| **App (Nginx)** | http://localhost | Unified entry point (proxies to web/backend) |
| **Adminer** | http://localhost:8081 | Database management UI |
| **MinIO Console** | http://localhost:9001 | S3-compatible storage UI |
| PostgreSQL | localhost:5432 | Database (user: agentsmesh, pass: agentsmesh_dev) |
| Redis | localhost:6379 | Cache |
| MinIO API | localhost:9000 | S3 API endpoint |

> **Note**: Ports may vary if using multiple worktrees. Check `.env` for actual ports.

### Hot Reload

All services support hot reload - source code is mounted into containers:
- **Backend**: Go code changes auto-rebuild via Air
- **Web**: Next.js fast refresh
- **Runner**: Go code changes auto-rebuild

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

### Runner (Go)

```bash
cd runner
make build                       # Build with desktop support (CGO)
make build-nocgo                 # Build CLI-only (no CGO)
make test                        # Run tests
make lint                        # golangci-lint
make build-all                   # Cross-platform builds
```

### Database Migrations

Migrations are located in `backend/migrations/` using golang-migrate format.

**Development** (via Docker):
```bash
cd deploy/dev
./init-worktree.sh               # Automatically runs all migrations
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
│   ├── domain/           # Domain models (DDD style)
│   │   ├── user/         # User entity
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
│   │   └── file/         # File storage entity
│   ├── service/          # Business logic layer
│   ├── infra/            # Infrastructure (DB, cache)
│   ├── config/           # Configuration loading
│   └── middleware/       # Auth, tenant isolation
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

## Runner Structure

```
runner/
├── cmd/runner/           # Entry point (register/run/service/desktop)
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
│   ├── tray/             # System tray (desktop mode)
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
3. Backend sends `create_pod` → Runner creates Sandbox → Starts PTY
4. Terminal output → PTYForwarder → `terminal_output` via gRPC → WebSocket to web
5. User input from web → WebSocket → `terminal_input` via gRPC → Runner writes to PTY stdin
6. Backend sends `terminate_pod` → Runner stops PTY → Cleans up Sandbox
7. Certificate auto-renewal before expiry (checked every hour)

## Configuration

**Development** (Docker): Run `cd deploy/dev && ./init-worktree.sh` - auto-generates all configs

**Runner**: `~/.agentsmesh/config.yaml` (created after `runner register`)

## Testing Patterns

- Backend: Standard Go testing with `testify`
- Web: Vitest + Testing Library
- Runner: Go testing, files ending with `_integration_test.go` for integration tests
