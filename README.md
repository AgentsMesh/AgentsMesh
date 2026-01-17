# AgentsMesh

Multi-tenant AI Code Agent collaboration platform supporting Claude Code, Codex CLI, Gemini CLI, Aider, and more.

## Features

- **AgentPod**: Remote AI development workstation with real-time terminal streaming
- **AgentsMesh**: Multi-agent collaboration with channel communication and pod binding
- **Tickets**: Task management with kanban board and merge request integration
- **Multi-tenant**: Organization > Teams > Users hierarchy with row-level isolation
- **Multi-Agent Support**: Claude Code, Codex CLI, Gemini CLI, OpenCode, Aider, and custom agents
- **Multi-Git Provider**: GitLab, GitHub, Gitee support
- **Self-hosted Runners**: Users deploy their own runners

## Tech Stack

- **Backend**: Go (Gin + GORM)
- **Frontend**: Next.js 14 (App Router) + TypeScript + Tailwind CSS
- **Mobile**: Flutter (planned)
- **Database**: PostgreSQL + Redis
- **API**: REST + gRPC
- **Real-time**: gRPC bidirectional streaming (Runner ↔ Backend)
- **Security**: mTLS (mutual TLS) for Runner connections

## Project Structure

```
AgentsMesh/
├── backend/               # Go backend
│   ├── cmd/server/        # Application entry point
│   ├── internal/
│   │   ├── api/           # REST handlers
│   │   ├── config/        # Configuration
│   │   ├── domain/        # Domain models
│   │   ├── infra/         # Infrastructure (database, cache)
│   │   ├── middleware/    # Auth, tenant middleware
│   │   └── service/       # Business logic
│   ├── migrations/        # Database migrations
│   └── pkg/               # Shared packages
├── web/                   # Next.js frontend
│   ├── src/
│   │   ├── app/           # App Router pages
│   │   ├── components/    # React components
│   │   ├── lib/           # Utilities & API client
│   │   ├── stores/        # Zustand stores
│   │   └── messages/      # i18n translations
├── runner/                # Runner daemon (gRPC + mTLS client)
├── mobile/                # Flutter mobile app (planned)
├── deploy/                # Docker & Kubernetes configs
└── scripts/               # Development scripts
```

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- pnpm
- Docker & Docker Compose
- golang-migrate (optional, for migrations)

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/anthropics/agentsmesh.git
   cd agentsmesh
   ```

2. **Start infrastructure**
   ```bash
   ./scripts/dev.sh
   ```
   This starts PostgreSQL, Redis, and Adminer.

3. **Start the backend**
   ```bash
   cd backend
   cp .env.example .env
   go run ./cmd/server
   ```

4. **Start the frontend** (in another terminal)
   ```bash
   cd web
   cp .env.example .env.local
   pnpm install
   pnpm dev
   ```

5. **Access the application**
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080
   - Adminer (DB UI): http://localhost:8081

### Database Migrations

```bash
# Apply all migrations
./scripts/migrate.sh up

# Rollback last migration
./scripts/migrate.sh down 1

# Create new migration
./scripts/migrate.sh create add_new_table

# Check current version
./scripts/migrate.sh version
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/login` - Email/password login
- `POST /api/v1/auth/register` - User registration
- `GET /api/v1/auth/oauth/:provider` - OAuth redirect
- `POST /api/v1/auth/refresh` - Token refresh
- `POST /api/v1/auth/logout` - Logout

### Organizations
- `GET /api/v1/organizations` - List organizations
- `POST /api/v1/organizations` - Create organization
- `GET /api/v1/organizations/:slug` - Get organization
- `PUT /api/v1/organizations/:slug` - Update organization

### Pods (AgentPod)
- `GET /api/v1/org/pods` - List pods
- `POST /api/v1/org/pods` - Create pod
- `GET /api/v1/org/pods/:key` - Get pod
- `POST /api/v1/org/pods/:key/terminate` - Terminate pod

### Channels (AgentsMesh)
- `GET /api/v1/org/channels` - List channels
- `POST /api/v1/org/channels` - Create channel
- `GET /api/v1/org/channels/:id/messages` - Get messages
- `POST /api/v1/org/channels/:id/messages` - Send message

### Tickets
- `GET /api/v1/org/tickets` - List tickets
- `POST /api/v1/org/tickets` - Create ticket
- `GET /api/v1/org/tickets/:identifier` - Get ticket
- `PUT /api/v1/org/tickets/:identifier` - Update ticket

## Supported Code Agents

| Agent | Provider | Description |
|-------|----------|-------------|
| Claude Code | Anthropic | Claude CLI tool |
| Codex CLI | OpenAI | OpenAI's code generation CLI |
| Gemini CLI | Google | Google Gemini CLI |
| OpenCode | Open Source | Open source AI coding tool |
| Aider | Open Source | Popular AI coding assistant |
| Custom | Any | Any terminal-based agent |

## Configuration

### Backend Environment Variables

```env
# Server
SERVER_ADDRESS=:8080
DEBUG=true

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=agentsmesh
DB_PASSWORD=agentsmesh_dev
DB_NAME=agentsmesh
DB_SSLMODE=disable

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRATION_HOURS=24

# OAuth Providers
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
```

### Frontend Environment Variables

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## License

MIT
