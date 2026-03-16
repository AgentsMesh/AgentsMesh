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
	// Start a parent process that spawns a child (cmd.exe /c timeout spawns conhost).
	cmd := exec.Command("cmd.exe", "/c", "timeout /t 30 >nul")
	require.NoError(t, cmd.Start())

	pid := cmd.Process.Pid
	inspector := DefaultInspector()

	// Give the process a moment to start.
	time.Sleep(500 * time.Millisecond)
	assert.True(t, inspector.IsRunning(pid), "process should be running before kill")

	// Kill the entire tree.
	err := KillProcessTree(pid)
	assert.NoError(t, err)

	// Give OS time to reap.
	time.Sleep(500 * time.Millisecond)
	assert.False(t, inspector.IsRunning(pid), "process should be dead after KillProcessTree")
}
