package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/stretchr/testify/assert"
)

func TestAdaptTerminalInput_NonCodex(t *testing.T) {
	data := []byte("hello\nworld\r")
	assert.Equal(t, data, adaptTerminalInput(data, "claude-code"))
	assert.Equal(t, data, adaptTerminalInput(data, "aider"))
	assert.Equal(t, data, adaptTerminalInput(data, ""))
}

func TestAdaptTerminalInput_CodexSingleLine(t *testing.T) {
	assert.Equal(t, []byte("hello"), adaptTerminalInput([]byte("hello"), "codex-cli"))
}

func TestAdaptTerminalInput_CodexSingleLineWithEnter(t *testing.T) {
	assert.Equal(t, []byte("hello\r"), adaptTerminalInput([]byte("hello\r"), "codex-cli"))
}

func TestAdaptTerminalInput_CodexMultiLine(t *testing.T) {
	input := []byte("Message from channel(#dev): fix the bug\n\nPlease reply.\r")
	result := adaptTerminalInput(input, "codex-cli")
	assert.Equal(t, []byte("Message from channel(#dev): fix the bug  Please reply.\r"), result)
	assert.NotContains(t, string(result[:len(result)-1]), "\n")
}

func TestAdaptTerminalInput_CodexCRLF(t *testing.T) {
	input := []byte("line1\r\nline2\r\nline3\r")
	result := adaptTerminalInput(input, "codex")
	assert.Equal(t, []byte("line1 line2 line3\r"), result)
}

func TestAdaptTerminalInput_CodexOnlyNewlines(t *testing.T) {
	result := adaptTerminalInput([]byte("\n\n\r"), "codex-cli")
	assert.Equal(t, []byte("\r"), result)
}

func TestAdaptTerminalInput_CodexEmpty(t *testing.T) {
	assert.Equal(t, []byte{}, adaptTerminalInput([]byte{}, "codex"))
}

func TestAdaptTerminalInput_CodexNoTrailingEnter(t *testing.T) {
	input := []byte("line1\nline2")
	result := adaptTerminalInput(input, "codex-cli")
	assert.Equal(t, []byte("line1 line2"), result)
}

func TestAdaptTerminalInput_BothCodexSlugs(t *testing.T) {
	input := []byte("hello\nworld\r")
	expected := []byte("hello world\r")
	assert.Equal(t, expected, adaptTerminalInput(input, "codex"))
	assert.Equal(t, expected, adaptTerminalInput(input, "codex-cli"))
}

type testAdapter struct{}

func (a *testAdapter) Adapt(data []byte) []byte {
	return []byte("adapted")
}

func TestAdaptTerminalInput_CustomAdapter(t *testing.T) {
	agentkit.RegisterInputAdapter("test-custom-agent", &testAdapter{})
	result := adaptTerminalInput([]byte("hello"), "test-custom-agent")
	assert.Equal(t, []byte("adapted"), result)
}
