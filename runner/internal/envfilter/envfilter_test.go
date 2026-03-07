package envfilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterEnv(t *testing.T) {
	input := []string{
		"HOME=/home/user",
		"PATH=/usr/bin",
		"AGENTSMESH_NODE_ID=abc123",
		"AGENTSMESH_ORG_SLUG=myorg",
		"GRPC_GO_LOG_VERBOSITY_LEVEL=99",
		"GRPC_GO_LOG_SEVERITY_LEVEL=info",
		"AWS_ACCESS_KEY_ID=mykey",
		"OPENAI_API_KEY=sk-xxx",
		"TERM=xterm-256color",
	}

	result := FilterEnv(input)

	// Preserved: user env vars, cloud creds, tool configs
	assert.Contains(t, result, "HOME=/home/user")
	assert.Contains(t, result, "PATH=/usr/bin")
	assert.Contains(t, result, "AWS_ACCESS_KEY_ID=mykey")
	assert.Contains(t, result, "OPENAI_API_KEY=sk-xxx")
	assert.Contains(t, result, "TERM=xterm-256color")

	// Filtered: Runner internals
	for _, e := range result {
		assert.False(t, shouldFilter(e), "should have been filtered: %s", e)
	}
	assert.Len(t, result, 5)
}

func TestFilterEnv_EmptyInput(t *testing.T) {
	result := FilterEnv(nil)
	assert.Empty(t, result)
}

func TestFilterEnv_NothingToFilter(t *testing.T) {
	input := []string{"HOME=/home/user", "PATH=/usr/bin"}
	result := FilterEnv(input)
	assert.Equal(t, input, result)
}
