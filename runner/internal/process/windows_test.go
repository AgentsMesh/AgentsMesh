//go:build windows

package process

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChildProcessesWindows(t *testing.T) {
	inspector := DefaultInspector()

	// Start a child process that sleeps.
	cmd := exec.Command("cmd.exe", "/c", "timeout /t 10 >nul")
	require.NoError(t, cmd.Start())
	defer cmd.Process.Kill()

	children := inspector.GetChildProcesses(os.Getpid())
	// Our test process should have at least this child.
	found := false
	for _, pid := range children {
		if pid == cmd.Process.Pid {
			found = true
			break
		}
	}
	assert.True(t, found, "child process %d should appear in GetChildProcesses", cmd.Process.Pid)
}

func TestGetProcessNameWindows(t *testing.T) {
	inspector := DefaultInspector()

	name := inspector.GetProcessName(os.Getpid())
	// The test binary name varies but should not be empty.
	assert.NotEmpty(t, name, "current process should have a name")
}

func TestIsRunningWindows(t *testing.T) {
	inspector := DefaultInspector()

	// Current process should be running.
	assert.True(t, inspector.IsRunning(os.Getpid()))

	// A very unlikely PID should not be running.
	assert.False(t, inspector.IsRunning(99999999))
}

func TestGetStateWindows(t *testing.T) {
	inspector := DefaultInspector()

	// Running process returns "R".
	state := inspector.GetState(os.Getpid())
	assert.Equal(t, "R", state)

	// Non-existent process returns "".
	state = inspector.GetState(99999999)
	assert.Equal(t, "", state)
}

func TestHasOpenFilesWindows(t *testing.T) {
	inspector := DefaultInspector()

	// Just verify it doesn't panic. The result depends on handle count heuristics.
	_ = inspector.HasOpenFiles(os.Getpid())
}

func TestIsAliveWindows(t *testing.T) {
	// Current process should be alive.
	err := IsAlive(os.Getpid())
	assert.NoError(t, err)

	// Non-existent PID should error.
	err = IsAlive(99999999)
	assert.Error(t, err)
}
