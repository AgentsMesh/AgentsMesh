package runner

import "github.com/anthropics/agentsmesh/runner/internal/acp"

// parseInitialConfigFromArgs extracts the AgentFile-resolved permission_mode
// and model from the runner's launch_args. Claude Code uses --permission-mode
// and --model flags; other agents (codex / gemini) leave both empty here and
// rely on agent-side defaults or post-start control_requests.
//
// We do not invent a proto field for this — the args are the resolved
// AgentFile output and the source of truth for the initial values.
func parseInitialConfigFromArgs(args []string) acp.Configuration {
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
