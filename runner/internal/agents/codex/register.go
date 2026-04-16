package codex

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

const TransportType = "codex"

func init() {
	acp.RegisterAgent("codex", TransportType, func(cb acp.EventCallbacks, l *slog.Logger) acp.Transport {
		return newTransport(cb, l)
	})

	tokenusage.RegisterParser([]string{"codex", "codex-cli"}, &codexParser{})

	adapter := &codexInputAdapter{}
	agentkit.RegisterInputAdapter("codex", adapter)
	agentkit.RegisterInputAdapter("codex-cli", adapter)

	agentkit.RegisterAgentHome(agentkit.AgentHomeSpec{
		EnvVar:      "CODEX_HOME",
		UserDirName: ".codex",
		MergeConfig: mergeTomlMcpServers,
	})

	agentkit.RegisterProcessNames("codex")
}
