<p align="center">
  <a href="https://youtu.be/VaXImaly3dM">
    <img src="https://img.youtube.com/vi/VaXImaly3dM/maxresdefault.jpg" alt="Loopal Demo Video" width="600" />
  </a>
</p>

<p align="center">
  <pre align="center">
  _        ___    ___   ____     _     _
 | |      / _ \  / _ \ |  _ \   / \   | |
 | |     | | | || | | || |_) | / _ \  | |
 | |___  | |_| || |_| ||  __/ / ___ \ | |___
 |_____|  \___/  \___/ |_|   /_/   \_\|_____|
  </pre>
  <em>Rooted in code, Growing with loopal.</em><br/>
  Part of <a href="https://agentsmesh.ai">AgentsMesh.ai</a>
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#license">License</a>
</p>

---

**Loopal** is a terminal-native AI coding agent built in Rust. It connects to LLM providers, reads and edits your codebase, runs commands, and orchestrates multi-agent workflows — all from inside your terminal with a rich TUI.

## Features

- 🚀 **Terminal-native TUI** — Rich interactive interface powered by [Ratatui](https://ratatui.rs), with Markdown rendering, syntax highlighting, and progress indicators.
- 🧠 **Multi-provider LLM support** — Works with OpenAI, Anthropic, and any OpenAI-compatible endpoint. Configurable thinking/reasoning modes (auto, effort levels, token budgets).
- 🔧 **Comprehensive tool suite** — File read/write/edit, multi-edit, apply-patch, grep, glob, ls, bash execution, background tasks, web search, and fetch.
- 🤖 **Multi-agent orchestration** — Spawn sub-agents that run in parallel, communicate via message passing and pub/sub channels, with a shared task store.
- 🔌 **MCP integration** — First-class [Model Context Protocol](https://modelcontextprotocol.io/) support for connecting external tool servers.
- 🖥️ **ACP server mode** — Agent Client Protocol over stdin/stdout JSON-RPC for IDE integration (Zed, JetBrains, Neovim, etc.) via `--acp`.
- 🔒 **Sandbox & permissions** — Configurable sandbox policies for filesystem access, network, and command execution. Supervised or bypass permission modes.
- 🪝 **Lifecycle hooks** — Run custom scripts on agent events (session start, tool calls, etc.).
- 💾 **Session management** — Persist and resume sessions. Context compaction with smart truncation to stay within token limits.
- 🧩 **Context pipeline** — Middleware-based context processing with message size guards, smart compaction, and context guards.
- 📝 **Memory** — Cross-session memory that persists observations and preferences across conversations.
- 🗂️ **Skills** — Extend agent capabilities with project-specific skill definitions.
- 📋 **Plan mode** — Read-only exploration mode for safe planning before making changes.

## Installation

### Prerequisites

- **Rust** (edition 2024, nightly toolchain recommended)
- An API key for your LLM provider (e.g., `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`)

### Build from source

```bash
git clone https://github.com/AgentsMesh/Loopal.git
cd Loopal
cargo build --release
```

The binary will be at `target/release/loopal`. Add it to your `PATH` or install directly:

```bash
cargo install --path .
```

## Quick Start

```bash
# Navigate to your project
cd your-project

# Start Loopal (uses default model from config)
loopal

# Start with a specific model
loopal -m claude-sonnet-4-20250514

# Start in plan mode (read-only exploration)
loopal --plan

# Resume a previous session
loopal -r <session-id>

# Run with an initial prompt
loopal "explain the architecture of this project"

# Bypass permission prompts
loopal -P bypass
```

## Architecture

Loopal is built as a modular Rust workspace with clear separation of concerns:

```
loopal
├── src/                        # Binary entry point, CLI, bootstrap
├── crates/
│   ├── loopal-kernel/          # Core kernel: provider registry, tool dispatch
│   ├── loopal-runtime/         # Agent loop, session management, frontend traits
│   ├── loopal-agent/           # Multi-agent registry, router, task store
│   ├── loopal-session/         # Session state, display messages, event handling
│   ├── loopal-tui/             # Terminal UI (Ratatui-based)
│   ├── loopal-provider/        # LLM provider implementations
│   ├── loopal-provider-api/    # Provider trait definitions
│   ├── loopal-protocol/        # Shared protocol types (events, messages)
│   ├── loopal-message/         # Message types and serialization
│   ├── loopal-config/          # Layered configuration system
│   ├── loopal-context/         # Context pipeline, compaction, system prompt
│   ├── loopal-memory/          # Cross-session persistent memory
│   ├── loopal-storage/         # Session persistence
│   ├── loopal-sandbox/         # Command & filesystem sandboxing
│   ├── loopal-hooks/           # Lifecycle hook system
│   ├── loopal-mcp/             # MCP client integration
│   ├── loopal-acp/             # ACP server for IDE integration
│   ├── loopal-git/             # Git operations & worktree management
│   ├── loopal-tool-api/        # Tool trait definitions
│   ├── loopal-error/           # Error types
│   └── tools/                  # Built-in tool implementations
│       ├── filesystem/         # read, write, edit, multi-edit, apply-patch,
│       │                       # grep, glob, ls, fetch, web-search, file-ops
│       ├── process/            # bash, background task execution
│       └── agent/              # ask-user, plan-mode
```

### Key design decisions

- **Layered configuration** — Settings merge from defaults → global config → project config → CLI flags.
- **Context pipeline** — Middleware chain processes messages before sending to the LLM, handling token limits, compaction, and guards.
- **Unified frontend** — Both TUI and ACP share the same `AgentFrontend` trait, making the agent loop UI-agnostic.
- **Sandbox-first** — File and command access goes through policy checks by default.

## Configuration

Loopal uses a layered configuration system. Create a `.loopal/config.toml` in your project root or `~/.config/loopal/config.toml` for global settings.

```toml
# .loopal/config.toml

# Default model
model = "claude-sonnet-4-20250514"

# Max turns per agent loop
max_turns = 200

# Permission mode: "supervised" or "bypass"
permission_mode = "supervised"

# Max context tokens before compaction
max_context_tokens = 120000

# Thinking/reasoning configuration
[thinking]
type = "auto"                  # "auto", "disabled", "effort", or "budget"
# level = "medium"             # for "effort" type
# tokens = 10000               # for "budget" type

# MCP servers
[mcp_servers.my-server]
command = "npx"
args = ["-y", "@my/mcp-server"]

# Sandbox policy
[sandbox]
policy = "strict"              # "strict", "permissive", or "disabled"
```

### Environment variables

| Variable | Description |
|---|---|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |

## CLI Reference

```
Usage: loopal [OPTIONS] [PROMPT]...

Arguments:
  [PROMPT]...           Initial prompt (non-interactive)

Options:
  -m, --model <MODEL>       Model to use
  -r, --resume <SESSION>    Resume a previous session
  -P, --permission <MODE>   Permission mode (supervised/bypass)
      --plan                Start in plan mode (read-only)
      --no-sandbox          Disable sandbox enforcement
      --acp                 Run as ACP server (stdin/stdout JSON-RPC)
  -h, --help                Print help
```

## License

Proprietary. Copyright (c) 2024-2026 AgentsMesh.ai. All Rights Reserved.

This software is licensed under a proprietary commercial license. See the [LICENSE](./LICENSE) file for the full terms. Unauthorized copying, modification, distribution, or use of this software is strictly prohibited.
