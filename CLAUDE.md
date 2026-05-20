# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AgentsMesh is **The AI Agent Workforce Platform** — where teams scale beyond headcount. It supports Claude Code, Codex CLI, Gemini CLI, Aider, and more.

### Components

Server-side (Go):
- **Backend** (`backend/`): API server (Gin + GORM). REST for clients, gRPC + mTLS to Runner. Owns auth, org/team/user, pod lifecycle, ticket/channel, billing, PKI for runner certs. PostgreSQL + Redis.
- **Relay** (`relay/`): WebSocket relay for the terminal data plane. Browser / Desktop / iOS ↔ Relay ↔ Runner (binary protocol). Backend never touches PTY bytes.
- **Runner** (`runner/`): Self-hosted daemon. Connects to Backend via gRPC bidi stream. Spawns isolated PTY pods that run the actual AI agents (Claude Code / Codex / Aider / …).

Client-side:
- **Rust Core** (`clients/core/`): **Business-logic SSOT.** 10 crates compiled to WASM (Web / Desktop) + native dylib via UniFFI (iOS XCFramework). Owns the authoritative cache, DTOs, and services — auth, blockstore, channels, tickets, mesh, autopilot. Front-ends are thin views over it. **Modify state in Rust, not in Zustand / TCA / etc.**
- **Web** (`clients/web/`): Next.js (App Router + TS + Tailwind). Loads `agentsmesh-wasm` at boot; UI state mirrors Rust selectors via `_tick` triggers.
- **Web-Admin** (`clients/web-admin/`): Next.js admin console mounted at `/admin`. Internal-only — gated on `is_system_admin`.
- **Desktop** (`clients/desktop/`): Electron + electron-vite. Renderer reuses `clients/web` source; main-process node-bridge proxies IPC to the same Rust Core (native NAPI build).
- **iOS** (`clients/ios/`): SwiftUI + TCA. Consumes the Rust Core via UniFFI-generated Swift bindings. Same DTOs, same services, ~zero duplicated business logic.

## Development Environment

**Bazel host-side mode**: Go services (backend / runner / relay) and the
Next.js apps (web / web-admin) all run on your host via `ibazel run` /
`bazel run :next_dev`. Docker only hosts stateful infrastructure
(PostgreSQL, Redis, MinIO, Traefik, Jaeger, Gitea, OTel collector,
Adminer). This means **every Go binary the dev environment runs is the
same artifact CI's `bazel build` produces** — no air, no per-service
Dockerfile, no parallel compile path.

### Quick Start

```bash
bazel run //deploy/dev:up                # docker infra + host backend/relay/runner + host web/web-admin
bazel run //deploy/dev:clean             # stop everything, drop docker volumes, clear runtime/
bazel run //deploy/dev:reset_runners     # only restart host runner+relay (backend stays up)
bazel run //deploy/dev:rebuild_runner    # rebuild runner binary + restart container
bazel run //deploy/dev:backend_only      # CI-style: skip frontends
```

> Backward-compat: `cd deploy/dev && ./dev.sh [--clean|--reset-runners|...]`
> still works — same flags, same behavior.

Prerequisites (one-time):

```bash
brew install bazelisk                        # macOS (bazel)
# ibazel is GitHub-Release-only (no homebrew formula); pick the
# darwin-arm64 binary off https://github.com/bazelbuild/bazel-watcher/releases
npm i -g @anthropic-ai/claude-code @openai/codex @google/gemini-cli  # for runner pods
```

The dev pipeline automatically:
1. Generates `.env` with worktree-isolated ports (3 host service ports added: BACKEND_HTTP_PORT / BACKEND_GRPC_PORT / RELAY_HTTP_PORT)
2. Generates traefik dynamic configs that route `host.docker.internal:<host-port>`
3. Starts the docker infra stack
4. Runs migrations via the `migrate/migrate` oneshot service (no backend container needed)
5. Launches `ibazel run` for backend / relay / runner in the background, with isolated `$HOME` for the runner so its `~/.claude/*` writes don't touch your real configs
6. Starts `bazel run //clients/web:next_dev` and `//clients/web-admin:next_dev`

### Services & Ports (offset 0 / main worktree)

| Service | URL | Notes |
|---------|-----|-------|
| **Frontend** | http://localhost:10007 | Bazel `next_dev` (host) |
| **Admin Console** | http://localhost:10011 | Bazel `next_dev` (host) |
| **API** | http://localhost:10000/api | traefik → host backend :10015 |
| **Relay** | ws://localhost:10000/relay | traefik → host relay :10017 |
| **gRPC mTLS** | grpcs://localhost:10001 | traefik passthrough → host backend :10016 |
| Postgres | localhost:10002 | docker |
| Redis | localhost:10003 | docker |
| MinIO API/Console | localhost:10004 / 10005 | docker |
| Adminer | localhost:10006 | docker |
| Traefik Dashboard | localhost:10008 | docker |
| Gitea HTTP/SSH | localhost:10009 / 10010 | docker |
| OTel gRPC/HTTP | localhost:10012 / 10013 | docker |
| Jaeger UI | localhost:10014 | docker |

Each worktree adds offset×50 to every slot.

Test accounts:
- **User**: dev@agentsmesh.local / devpass123
- **Admin**: admin@agentsmesh.local / adminpass123

### Logs

```bash
tail -f deploy/dev/runtime/backend/backend.log   # ibazel + backend stdout
tail -f deploy/dev/runtime/relay/relay.log
tail -f deploy/dev/runtime/runner/runner.log
tail -f deploy/dev/web.log                       # bazel next_dev (web)
docker compose logs -f postgres                  # docker infra
```

### Hot Reload

- **Frontend (web / web-admin)**: Next.js dev server fast refresh
- **Go services (backend / runner / relay)**: `ibazel run //path:target` — watches Bazel's dependency graph and rebuilds incrementally on `.go` change. Same compile path as `bazel test //...`, no parallel toolchain.

## Build Commands (for CI/testing outside Docker)

> **Note — Bazel migration in progress.** The Bazel-based build system
> (see `.claude/plans/snuggly-spinning-dewdrop.md`) is landing
> incrementally. Until it's complete the **legacy commands below stay
> authoritative**; Bazel targets listed under "Bazel (migration)" are
> opt-in and verified in CI on `bazel.yml` only.

### Bazel (migration in progress)

```bash
# One-shot validation that the workspace still parses
bazel info workspace
bazel run //:buildifier_check

# Regenerate Go BUILD.bazel files after editing imports / adding packages
bazel run //:gazelle

# Build a Go binary + its OCI image
bazel build //backend/cmd/server:server
bazel build //backend/cmd/server:image
bazel run //backend/cmd/server:image_tarball   # → docker load

# Build the Rust → XCFramework chain (once Phase 3b is live)
bazel build //clients/core/crates/ffi:AgentsMeshCore

# Build the iOS app (once Phase 5 is live)
bazel build //clients/ios:AgentsMesh
bazel run //clients/ios:AgentsMesh_xcodeproj   # → Xcode project
```

### Backend (Go)

```bash
bazel build //backend/cmd/server:server                   # Build binary
bazel test //backend/...                                  # Run all tests
bazel test //backend/internal/service/... --test_filter=TestAuth  # Run specific test
bazel run //backend:lint                                  # golangci-lint
```

### Web (Next.js)

所有前端的依赖（web / web-admin / desktop）统一放在根 `package.json`。
Per-app `package.json` 已删（desktop 仅留 thin shell 满足 electron-builder
读 `name`/`version`/`main` 的需求）。Lint / type-check / 单测全部走 Bazel：

```bash
pnpm install                              # Install at repo root (one-shot)
bazel run //clients/web:next_dev          # Dev server (preferred)
bazel build //clients/web:image           # Production OCI image
bazel test //clients/web:unit             # Vitest (1510 tests)
bazel test //clients/web:lint             # ESLint
bazel build //clients/web:src             # tsc --noEmit (type check)
bazel test //clients/web-admin:lint       # ESLint web-admin
bazel build //clients/web-admin:src       # tsc --noEmit web-admin

# Dev server shell alternative (for IDE / non-Bazel workflows)
(cd clients/web && node ../../node_modules/next/dist/bin/next dev --turbopack)
```

#### Wasm 加载边界（路由分层）

为避免 21MB wasm 在静态/营销页 block 渲染（手机基本跑不动），WasmProvider
**仅挂在三组 layout 中**，营销页保持 0 wasm：

| Layout | wasm | 路由 |
|---|---|---|
| `app/layout.tsx` (root) | ❌ | 全站基底，无 wasm |
| `app/(dashboard)/layout.tsx` | ✅ | `(dashboard)/[org]/**`、`/settings`、`/support` |
| `app/(auth)/layout.tsx` | ✅ | `/login`、`/register`、OAuth callback、verify-email、invite、onboarding、runners |
| `app/popout/layout.tsx` | ✅ | `/popout/terminal/[podKey]` |
| 其它营销/文档 (`/`、`/docs`、`/about`、`/blog`、`/changelog`、`/demo`、`/enterprise`、`/privacy`、`/terms`、`/mock-checkout` 等) | ❌ | 通过 `lib/light-session.ts` 直读 localStorage 判 auth；公开 API 走 `lib/public-api.ts` 的 fetch |

**约束**（违反会让营销页重新加载 wasm）：
- 营销页组件**不要 import** `@/lib/wasm-core` / `@agentsmesh/service-runtime` / `agentsmesh-wasm` / `@/stores/auth`（任意一个会通过依赖图把 21MB 拉进 chunk）
- 需要"已登录吗 + 当前 org slug"用 `useLightSession`（来自 `@/hooks/useLightSession`）
- 需要 CTA 用 `LightAuthButtons`（不是 `AuthButtons`）
- 需要公开 API（pricing 等）用 `fetch` 或 `lib/public-api.ts` 包装，不走 wasm

**校验**：CI / 本地构建后跑 `bash clients/web/scripts/check-no-wasm-in-marketing.sh` 验证营销 chunk 不含 wasm 符号。

### Web-Admin (Next.js)

```bash
bazel run //clients/web-admin:next_dev
bazel build //clients/web-admin:image
bazel test //clients/web-admin:lint
bazel build //clients/web-admin:src
```

### Desktop (Electron)

Desktop 也走单根 `package.json`（thin shell 仅含 `name`/`version`/`main`，
所有 deps 在根）。Bazel-native build pipeline（自写 `electron_builder_app`
macro，包住 electron-vite + electron-builder）：

```bash
# Production build — main + preload + renderer bundles (Bazel)
bazel build //clients/desktop:out

# Package — .dmg / .zip / .app (macOS), .exe (Win), .AppImage (Linux)
# `electron_builder` tag opts in to the heavy packaging targets that
# `bazel build //...` skips. Output: bazel-bin/clients/desktop/dist/
bazel build //clients/desktop:dist --build_tag_filters=electron_builder

# Dev (electron-vite dev server still goes through node)
(cd clients/desktop && node ../../node_modules/electron-vite/bin/electron-vite.js dev)

# E2E (requires `:out` already built)
bazel test //clients/desktop:e2e --test_tag_filters=e2e
```

### Runner (Go)

```bash
bazel build //runner/cmd/runner:runner            # Native binary
bazel test //runner/...                            # Run tests
bazel build //runner/cmd/runner:image              # OCI image (distroless)
bazel build //runner/cmd/runner:release_assets     # 6-platform tar.gz/zip + checksums
bazel run //runner:lint                            # golangci-lint (hermetic v2.11.4)
```

Release assets (`release_assets`) replace the previous GoReleaser
pipeline: 6 cross-compiled binaries (linux/darwin/windows ×
amd64/arm64) packaged as tar.gz/zip, plus `checksums.txt`. The
release.yml workflow stamps version into the staged filenames and
runs `rcodesign` over darwin binaries before `gh release create`.

### Go Lint (backend / runner / relay)

```bash
bazel run //backend:lint    # golangci-lint over backend/
bazel run //runner:lint     # golangci-lint over runner/
bazel run //relay:lint      # golangci-lint over relay/
```

The golangci-lint binary is fetched hermetically by `rules_multitool`
(`multitool.lock.json` pins v2.11.4 across linux/macOS amd64+arm64).
Each module reads its own `.golangci.yml`. CI runs the same
`bazel run //<module>:lint` commands — no parallel `golangci-lint-action`.

### Rust Core (Bazel-only — no Cargo workspace)

Rust 业务代码（`clients/core/crates/`）的构建/测试/lint **完全走 Bazel**。
仓库**没有** `Cargo.toml` workspace、`Cargo.lock`、或 `.cargo/config.toml`。
依赖在 `MODULE.bazel` 的 `crate.spec()` 块声明（SSOT），BUILD.bazel
通过 `@crates//:<name>` 引用。

```bash
bazel test //clients/core/crates/auth:auth_test
bazel build //clients/core/crates/ffi:ffi
bazel build //clients/core/crates/wasm:wasm_lib
bazel build //clients/core/crates/node-bridge:node_bridge

# Generate rust-project.json for IDE / rust-analyzer
bazel run //:rust_project
```

**例外**：`clients/core/crates/ffi/Cargo.toml` 保留为 stub（仅 `[package]`
三行），因为 uniffi 的 `#[uniffi::export]` proc-macro 在编译时通过
`$CARGO_MANIFEST_DIR/Cargo.toml` 读取 crate name。**不要在这个 stub 加任何
dependencies — Bazel 不读它**。

**加新依赖**：编辑 `MODULE.bazel` 的 `crate.spec()` 块 → 加 `BUILD.bazel`
的 deps 引用 `@crates//:<name>`。**不要新建 Cargo.toml**。

### iOS (SwiftUI + TCA, powered by Rust Core via UniFFI)

```bash
# One-time setup
rustup target add aarch64-apple-ios aarch64-apple-ios-sim x86_64-apple-ios

# Build the signed .ipa (Bazel builds the XCFramework as a transitive dep)
bazel build //clients/ios:AgentsMesh

# Or build just the XCFramework (consumed by clients/ios SPM package)
bazel build //clients/core/crates/ffi:AgentsMeshCore

# Run on a booted simulator (auto-boots iPhone 17 Pro if none)
bazel run //clients/ios:AgentsMesh_sim

# Develop in Xcode — regenerable, never commit the .xcodeproj
bazel run //clients/ios:AgentsMesh_xcodeproj
open AgentsMesh.xcodeproj
```

Bazel tree artifacts:
- `bazel-bin/clients/core/crates/ffi/AgentsMeshCore.xcframework/` — device + universal-sim slices + `Info.plist`
- `bazel-bin/clients/core/crates/ffi/AgentsMeshCore_bindings_out/AgentsMeshCore.swift` — Swift glue (~18k lines)

The SPM package at `clients/ios/Packages/AgentsMeshCore/` references those
artifacts directly through Bazel's `ios_app` macro — no source-tree
symlinks, no manual `make` step.

Layout:
- `clients/ios/Packages/AgentsMeshCore/` — SPM facade: CoreBridge (singleton),
  KeychainStorage (StorageCallback impl), EventStream, PodOutputDispatcher
- `clients/ios/Packages/AgentsMeshFeatures/` — TCA reducers + SwiftUI views:
  AppFeature (root), AuthFeature (login), WorkspaceFeature (pod list),
  TerminalFeature (SwiftTerm + Relay WS), DesignSystem (tokens + primitives)
- `clients/ios/App/` — Xcode App target (@main entry, Info.plist)

Requirements: macOS with Xcode 15+. CI: `.github/workflows/ios.yml` (macOS runner).

### Database Migrations

Migrations are located in `backend/migrations/` using golang-migrate format.

**Development** (via Docker):
```bash
bazel run //deploy/dev:up    # automatically runs all migrations
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
clients/web/src/
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
clients/web-admin/src/
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

**Development** (Docker): Run `bazel run //deploy/dev:up` - auto-generates all configs

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
* **代码即 SSOT — 不要解释 what，只解释 why。** 注释能删则删。可以写注释的场景：业务约束、跨模块契约、解决方案的非显然取舍 (workaround 原因)。绝不能写：复述函数名/类型名的注释、`// 创建 X` 之上紧跟 `CreateX()`、section banner、文档化签名的 JSDoc。代码不够自解释就改代码，不要靠注释补救。
* **Hard limit: every file must stay under 200 lines** (excluding test files, which should stay under 400 lines). When a file approaches this limit, proactively split it by SRP — extract types, helpers, hooks, or sub-components into separate files. A 210-line file is acceptable if splitting would break cohesion; a 300+ line file is never acceptable and must be split before committing.
* **Code is the single source of truth — comments that can be eliminated, must be eliminated.** Only comment to explain **why** something non-obvious exists (business constraints, cross-module contracts, workarounds). Never comment **what** code does — if the code isn't self-explanatory, rewrite the code. No JSDoc that restates the function signature, no `// Create X` above `CreateX()`, no section banners.
* **File names must be specific and descriptive.** Never use generic names like `helpers`, `utils`, `common`, `misc`, `shared`. Name files after what they contain — e.g., `mesh-status-info.ts` not `mesh-helpers.ts`, `runner-display-info.ts` not `runner-utils.ts`. 

## Identifier 字段契约

任何 UNIQUE string 列、URL path 段、@mention key、lookup 主键 **都是 identifier**，必须满足 `backend/pkg/slugkit` 的规则：`^[a-z0-9]+(-[a-z0-9]+)*$`，长度 2-100。

### 字段身份分层
- **认证身份** (`users.email`): 用户输入凭证，合法 email 格式即可
- **公开身份 / identifier** (`users.username`, `organizations.slug`, `channels.slug`, `pods.pod_key`...): 系统派生 + 严格 sanitize，全小写 + 数字 + 连字符
- **呈现层** (`users.name`, `channels.name`...): 任意 Unicode，仅用于 UI 显示

**`name` 字段不得当 identifier 用**。如果需要 lookup，加 `slug` 列。

### 写入路径强制约束
- 外部 raw 字符串（OAuth login / SAML attr / email local-part / AI 输出）**禁止**直接赋值到 identifier 字段
- 必须通过对应 service helper：`userService.EnsureUniqueUsername`、`orgService.CreatePersonal`、`channelService.EnsureUniqueSlug` 等
- helper 内部走 `slugkit.GenerateUnique(seed, dbExistsCheck)`，带 collision retry

### 新增 identifier 字段 checklist
1. migration 加 column + `CHECK (col ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(col) BETWEEN 2 AND 100)`
2. domain model 字段类型用 `slugkit.Slug`（新代码） 或 `string`（兼容老代码）
3. 加 `BeforeSave` hook 调 `slugkit.ValidateIdentifier("<table>.<col>", value)`
4. service 包加 `*Registry` helper 封装 `slugkit.GenerateUnique`
5. 单测覆盖含 `.`/`_`/uppercase/unicode 的输入，断言落库值通过 `slugkit.Validate`

参见 `backend/pkg/slugkit/doc.go` 完整说明，`.claude/plans/sharded-imagining-bird.md` 重构 plan。