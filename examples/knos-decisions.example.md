# Decisions and current work

<!-- Example record. `knos export` writes this file in an adopting repo; it is
     plain markdown, so a fresh clone reads it with nothing installed. -->

## Decisions

- **rule 1** - **Relay** (`relay/`): WebSocket relay for the terminal data plane. Browser / Desktop / iOS ↔ Relay ↔ Runner (binary protocol). Backend never touches PTY bytes. _(source: CLAUDE.md)_
- **rule 2** - **Runner** (`runner/`): Self-hosted daemon. Connects to Backend via gRPC bidi stream. Spawns isolated PTY pods that run the actual AI agents (Claude Code / Codex / Aider / …). _(source: CLAUDE.md)_

## Being worked on right now

_Nothing claimed._

---
<sub>One record every agent working in this repo reads. Claims lapse after 30
minutes or on `knos done`.</sub>
