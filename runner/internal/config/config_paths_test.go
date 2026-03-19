package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSocketDir(t *testing.T) {
	cfg := &Config{}
	got := cfg.GetSocketDir()
	expected := filepath.Join(TempBaseDir(), "sockets")
	assert.Equal(t, expected, got)
}

func TestGetLogPTYDir(t *testing.T) {
	cfg := &Config{}
	got := cfg.GetLogPTYDir()
	expected := filepath.Join(TempBaseDir(), "pty-logs")
	assert.Equal(t, expected, got)
}
