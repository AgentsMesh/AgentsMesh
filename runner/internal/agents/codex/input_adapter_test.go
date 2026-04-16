package codex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodexInputAdapter_SingleLine(t *testing.T) {
	a := &codexInputAdapter{}
	assert.Equal(t, []byte("hello"), a.Adapt([]byte("hello")))
}

func TestCodexInputAdapter_SingleLineWithEnter(t *testing.T) {
	a := &codexInputAdapter{}
	assert.Equal(t, []byte("hello\r"), a.Adapt([]byte("hello\r")))
}

func TestCodexInputAdapter_MultiLine(t *testing.T) {
	a := &codexInputAdapter{}
	result := a.Adapt([]byte("line1\nline2\r"))
	assert.Equal(t, []byte("line1 line2\r"), result)
}

func TestCodexInputAdapter_CRLF(t *testing.T) {
	a := &codexInputAdapter{}
	result := a.Adapt([]byte("a\r\nb\r\nc\r"))
	assert.Equal(t, []byte("a b c\r"), result)
}

func TestCodexInputAdapter_OnlyNewlines(t *testing.T) {
	a := &codexInputAdapter{}
	assert.Equal(t, []byte("\r"), a.Adapt([]byte("\n\n\r")))
}

func TestCodexInputAdapter_Empty(t *testing.T) {
	a := &codexInputAdapter{}
	assert.Equal(t, []byte{}, a.Adapt([]byte{}))
}

func TestCodexInputAdapter_NoTrailingEnter(t *testing.T) {
	a := &codexInputAdapter{}
	assert.Equal(t, []byte("a b"), a.Adapt([]byte("a\nb")))
}
