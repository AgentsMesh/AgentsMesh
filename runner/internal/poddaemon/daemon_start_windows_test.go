//go:build windows

package poddaemon

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

func TestStartDaemonWindows(t *testing.T) {
	sandboxDir := t.TempDir()

	// Write a minimal state file so the daemon can load it (even though
	// cmd.exe won't actually act as a real daemon).
	state := &PodDaemonState{
		PodKey:      "test-daemon",
		SandboxPath: sandboxDir,
		Command:     "ping",
		Args:        []string{"-n", "5", "127.0.0.1"},
		Cols:        80,
		Rows:        24,
	}
	require.NoError(t, SaveState(state))

	configPath := StatePath(sandboxDir)

	// Start a detached process. We use cmd.exe as the "binary" since the
	// real runner binary isn't available in unit tests.
	pid, err := startDaemon(`C:\Windows\System32\cmd.exe`, configPath, sandboxDir, os.Environ())
	require.NoError(t, err)
	assert.Greater(t, pid, 0)

	// Clean up: kill the process tree.
	t.Cleanup(func() {
		_ = process.KillProcessTree(pid)
	})
}

func TestStartDaemonInvalidBin(t *testing.T) {
	sandboxDir := t.TempDir()

	_, err := startDaemon(`C:\nonexistent\path\fake.exe`, "", sandboxDir, nil)
	require.Error(t, err)
}
