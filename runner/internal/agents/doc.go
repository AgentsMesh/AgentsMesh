// Package agents hosts per-agent adaptation plugins.
//
// Each subdirectory represents one AI agent (e.g., claude/, codex/, aider/).
// All agent-specific logic is self-contained within its directory.
//
// # Adding a New Agent
//
// 1. Create internal/agents/<name>/ directory.
//
// 2. Create register.go with an init() function that registers needed capabilities:
//
//	func init() {
//	    // Required for ACP agents with a custom protocol:
//	    acp.RegisterAgent("command-name", transportType, factory)
//
//	    // Required — token usage collection from session files:
//	    tokenusage.RegisterParser([]string{"slug1", "slug2"}, &myParser{})
//
//	    // Required — process name identification for PTY state monitoring:
//	    agentkit.RegisterProcessNames("process-name")
//
//	    // Optional — terminal input sanitization for PTY TUI agents:
//	    agentkit.RegisterInputAdapter("slug", &myAdapter{})
//
//	    // Optional — per-pod home directory isolation + config merging:
//	    agentkit.RegisterAgentHome(agentkit.AgentHomeSpec{...})
//	}
//
// 3. Add a blank import in internal/runner/agents_import.go:
//
//	_ "github.com/anthropics/agentsmesh/runner/internal/agents/<name>"
//
// # Extension Points
//
//   - acp.RegisterAgent: Wire protocol (Transport interface) + command→transport mapping.
//     Only needed for agents with non-standard ACP protocols. Standard ACP agents
//     (JSON-RPC 2.0) need no transport registration — the default applies automatically.
//
//   - tokenusage.RegisterParser: Parses agent session files for token usage data.
//     Called on pod exit to collect cost metrics.
//
//   - agentkit.RegisterProcessNames: Process names used by the monitor to identify
//     agent processes in the process tree (PTY state detection).
//
//   - agentkit.RegisterInputAdapter: Adapts raw terminal input before sending to
//     the agent's PTY. Used when a TUI needs input sanitization (e.g., newline handling).
//
//   - agentkit.RegisterAgentHome: Isolates the agent's config directory per-pod.
//     Copies user config, merges platform MCP settings in the agent's config format.
//
// # Conflict Detection
//
// All Register* functions panic on duplicate registration to catch misconfiguration
// early during init(). This ensures each command name, transport type, parser slug,
// and input adapter slug is owned by exactly one agent.
package agents
