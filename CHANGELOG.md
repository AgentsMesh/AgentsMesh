# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
for tagged releases where possible.

## [Unreleased]

### Added

- Channel @mention with pod prompt forwarding
- ChannelPodManager for managing pod membership in channels
- Loop feature - CI/CD for AI Agent Tasks

### Fixed

- Terminal graceful shutdown, pod race conditions, and resource cleanup
- Codex CLI 0.100+ approval mode compatibility & early output capture
- PTY read error detection and propagation to prevent frozen terminals
- Remove hardcoded GitLab PAT from E2E test scenario

## [0.5.0] - 2026-02-27

### Added

- Skills & MCP Server capabilities system for repository extensions
- Market-based skill and MCP server installation
- Custom MCP server support

## [0.4.7] - 2026-02-26

### Added

- Agent version detection, heartbeat reporting, and server-side version adaptation

## [0.4.6] - 2026-02-26

### Fixed

- Deduplicate channel messages to prevent double display
- Render Dialog via portal to prevent scaling in ReactFlow canvas
- Remove unnecessary polling on ticket pods endpoint

### Changed

- Optimize ticket detail interaction to Linear-style UX

## [0.4.5] - 2026-02-26

### Added

- Dual clone URL support (HTTP + SSH) with non-blocking Runner probe
- Image paste support with clipboard shim and native backends

### Fixed

- Runner: increase probe timeout and improve error logging
- Runner: stop reconnecting on fatal auth errors and show actionable hints
- Runner: fix cross-org runner registration causing connection/work failures
- Fix CancelSubscription missing timestamp and org settings UI bugs
- Fix comment mentions JSONB serialization causing 500/404 errors

## [0.4.4] - 2026-02-23

### Added

- Ticket comment system with threaded replies and @mentions
- Command Center blog post with 8-language i18n

### Changed

- Replace all logos and favicons with new Control Tower design
- Unify footer component and fix public page headers

### Fixed

- Auto-join creating pod to channel via MCP
- Use transaction DB for trial subscription creation during org setup
- Preserve Runner Group positions after drag and prevent overlap

## [0.4.0] - 2026-02-23

### Added

- Runner visibility control (organization/private)
- Follow Runner model option for Claude Code pods
- Token-based member invitation with pending list
- MCP: convert BlockNote JSON ticket content to plain text with line-range pagination

### Changed

- Unify ticket identifier→slug naming
- Standardize error responses with structured error codes
- Optimize MCP tool results from JSON to Markdown/text format

### Fixed

- Admin subscription management fixes (GORM Model overriding plan_id)
- Mixed Content errors caused by build-time env inlining
- Duplicate migration sequence number

## [0.3.0] - 2026-02-07

### Added

- Relay server for terminal data streaming (Browser ↔ Relay ↔ Runner)
- LemonSqueezy payment integration and Stripe mock framework
- Autopilot controller implementation with UI redesign
- iOS PWA notification support and OSC terminal notifications
- PR/MR status awareness mechanism

### Changed

- Massive SRP refactoring: split large files (600-1000+ lines) following Single Responsibility Principle
- Extract reusable components (ConfirmDialog, TerminalPane, RepositoriesSidebar)
- Rename binary from 'runner' to 'agentsmesh-runner'

### Fixed

- Terminal cursor and IME input duplication issues on mobile
- Terminal size synchronization issues
- Handle 4xx/5xx API errors in Git provider doRequest

## [0.2.0] - 2026-01-16

### Added

- Multi-language support with 8 locales (i18n)
- IDE-style layout with terminal connection optimization
- PTY size sync with debounce and broadcast
- Mobile UX improvements with sidebar, auth sync, and touch scrolling
- EventBus and WebSocket real-time event system
- Merge skill for MR creation and pipeline monitoring
- Worktree skill for isolated branch development

### Changed

- Unify brand name to AgentsMesh
- Optimize runner build process and GitLab CI tags

## [0.1.0] - 2026-01-11

### Added

- Initial release of AgentsMesh platform
- Backend: Go API server with Gin + GORM, JWT + OAuth authentication
- Web: Next.js frontend with App Router, TypeScript, Tailwind CSS
- Runner: Go daemon with isolated PTY environments
- gRPC + mTLS communication between Runner and Backend
- PKI for runner certificate issuance and revocation
- Multi-tenant organization management
- Pod lifecycle management (create, connect, terminate)
- Channel-based multi-agent collaboration
- Ticket management with kanban board
- Repository management with Git provider integration
- Billing system with quota enforcement
- Desktop mode with system tray for Runner
- Real-time terminal streaming via WebSocket

[Unreleased]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.4.7...v0.5.0
[0.4.7]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.4.6...v0.4.7
[0.4.6]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.4.5...v0.4.6
[0.4.5]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.4.4...v0.4.5
[0.4.4]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.4.0...v0.4.4
[0.4.0]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/AgentsMesh/AgentsMesh/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/AgentsMesh/AgentsMesh/releases/tag/v0.1.0
