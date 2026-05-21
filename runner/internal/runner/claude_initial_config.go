package runner

import "github.com/anthropics/agentsmesh/runner/internal/acp"

// parseClaudeInitialConfig extracts the AgentFile-resolved permission_mode
// and model from the runner's launch_args. Only Claude Code's AgentFile
// produces these specific flags (--permission-mode, --model); codex and
// gemini do not, so this function correctly returns an empty Configuration
// for them. The name is Claude-specific on purpose — callers that want a
// more general config-from-args reader should add a sibling function rather
// than overload this one.
//
// We do not invent a proto field for this — the args are the resolved
// AgentFile output and the source of truth for the initial values.
func parseClaudeInitialConfig(args []string) acp.Configuration {
	var cfg acp.Configuration
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--permission-mode":
			cfg.PermissionMode = args[i+1]
		case "--model":
			cfg.Model = args[i+1]
		}
	}
	return cfg
}
