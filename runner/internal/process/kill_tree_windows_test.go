//go:build windows

package process

import (
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKillProcessTreeWindows(t *testing.T) {
	// Use "ping -n 60 127.0.0.1" — a reliable long-running process that
	// the current user always has permission to terminate (unlike timeout.exe
	// which spawns conhost with elevated handles in some CI environments).
	cmd := exec.Command("ping", "-n", "60", "127.0.0.1")
	require.NoError(t, cmd.Start())

	pid := cmd.Process.Pid
	inspector := DefaultInspector()

	// Give the process a moment to start.
	time.Sleep(500 * time.Millisecond)
	require.True(t, inspector.IsRunning(pid), "process should be running before kill")

	// Kill the entire tree.
	err := KillProcessTree(pid)
	require.NoError(t, err)

	// Give OS time to reap.
	time.Sleep(500 * time.Millisecond)
	assert.False(t, inspector.IsRunning(pid), "process should be dead after KillProcessTree")
}
