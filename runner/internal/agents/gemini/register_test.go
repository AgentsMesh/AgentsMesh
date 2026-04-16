package gemini

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/stretchr/testify/assert"
)

func TestGeminiRegistered(t *testing.T) {
	assert.True(t, agentkit.IsAgentProcess("gemini"))
}
