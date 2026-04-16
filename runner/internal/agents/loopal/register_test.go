package loopal

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
	"github.com/stretchr/testify/assert"
)

func TestLoopalParser_ReturnsNil(t *testing.T) {
	parser := &loopalParser{}
	usage, err := parser.Parse(t.TempDir(), time.Time{})
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestLoopalRegistered(t *testing.T) {
	assert.NotNil(t, tokenusage.GetParser("loopal"))
	assert.True(t, agentkit.IsAgentProcess("loopal"))
}
