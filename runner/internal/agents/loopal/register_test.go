package loopal

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
	"github.com/stretchr/testify/assert"
)

func TestLoopalRegistered(t *testing.T) {
	assert.True(t, tokenusage.IsParserOptOut("loopal"), "loopal should be opt-out from token-usage fixture contract")
	assert.Nil(t, tokenusage.GetParser("loopal"), "opt-out agents must not register a parser")
	assert.True(t, agentkit.IsAgentProcess("loopal"))
}
