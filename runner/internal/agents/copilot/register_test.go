package copilot

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
	"github.com/stretchr/testify/assert"
)

func TestCopilotRegistered(t *testing.T) {
	// Opt-out covers BOTH the DB slug and the runtime launch_command key.
	assert.True(t, tokenusage.IsParserOptOut("copilot-cli"), "slug copilot-cli should be opt-out from token-usage fixture contract")
	assert.True(t, tokenusage.IsParserOptOut("copilot"), "launch_command copilot (the runtime token-collection key) should be opt-out")
	assert.Nil(t, tokenusage.GetParser("copilot-cli"), "opt-out agents must not register a parser (slug key)")
	assert.Nil(t, tokenusage.GetParser("copilot"), "opt-out agents must not register a parser (runtime launch_command key)")

	assert.True(t, agentkit.IsAgentProcess("copilot"), "process name copilot must be registered")
	assert.False(t, agentkit.IsAgentProcess("copilot-cli"), "the DB slug must NOT be registered as a process name (the binary is copilot)")
}
