package copilot

import (
	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func init() {
	// GitHub Copilot CLI speaks standard ACP JSON-RPC 2.0 over `copilot --acp`
	// (verified: initialize -> agentCapabilities incl. loadSession), so the
	// default ACP transport applies and no acp.RegisterAgent is needed - same
	// as gemini / opencode.
	//
	// Copilot exposes no documented on-disk session format we can parse for
	// token usage, so opt out of the cross-agent fixture contract rather than
	// ship a no-op parser (per agents/doc.go). Cover BOTH the DB slug
	// ("copilot-cli") and the runtime launch_command key ("copilot") that
	// tokenusage.Collect uses (pod.Agent == LaunchCommand), mirroring cursor.
	tokenusage.RegisterParserOptOut([]string{"copilot-cli", "copilot"})
	agentkit.RegisterProcessNames("copilot")
}
