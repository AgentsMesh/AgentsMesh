//go:build integration && !windows

package poddaemon

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPodDaemon_StateFileCorruption_Integration creates a daemon, detaches,
// corrupts the state file, and verifies RecoverSessions skips it.
func TestPodDaemon_StateFileCorruption_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "cr")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	dpty, _, err := mgr.CreateSession(CreateOpts{
		PodKey: "p", Agent: "test", Command: "cat",
		WorkDir: sandbox, Env: os.Environ(),
		Cols: 80, Rows: 24, SandboxPath: sandbox,
	})
	require.NoError(t, err)

	// Detach (daemon keeps running)
	require.NoError(t, dpty.Close())
	time.Sleep(200 * time.Millisecond)

	// Corrupt the state file with invalid JSON
	statePath := StatePath(sandbox)
	require.NoError(t, os.WriteFile(statePath, []byte("{{{invalid json"), 0600))

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	assert.Empty(t, sessions, "corrupted session should be skipped")
}

// TestPodDaemon_EnvVarsPassed_Integration verifies env vars reach the child.
// Uses cat (long-lived) so daemon stays up, then writes a shell command via PTY.
func TestPodDaemon_EnvVarsPassed_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "ev")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	// Use sh (interactive) so we can write a command; env var is passed via Env.
	dpty, _, err := mgr.CreateSession(CreateOpts{
		PodKey:  "p",
		Agent:   "test",
		Command: "sh",
		WorkDir: sandbox,
		Env:     append(os.Environ(), "MY_TEST_VAR=hello123"),
		Cols:    80, Rows: 24,
		SandboxPath: sandbox,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		dpty.Kill()
		dpty.Close()
		DeleteState(sandbox)
	})

	// Write a command that prints the env var
	_, err = dpty.Write([]byte("echo MY_VAL=$MY_TEST_VAR\n"))
	require.NoError(t, err)

	// Read until we see the value (PTY may split output across reads)
	buf := make([]byte, 4096)
	var output string
	for i := 0; i < 20; i++ {
		dpty.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := dpty.Read(buf)
		if err != nil {
			break
		}
		output += string(buf[:n])
		if strings.Contains(output, "MY_VAL=hello123") {
			break
		}
	}
	assert.Contains(t, output, "MY_VAL=hello123")
}

// TestPodDaemon_IPCProtocolMessages_Integration verifies data round-trip
// through the daemon and resize command handling.
func TestPodDaemon_IPCProtocolMessages_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "ip")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	dpty, _, err := mgr.CreateSession(CreateOpts{
		PodKey: "p", Agent: "test", Command: "cat",
		WorkDir: sandbox, Env: os.Environ(),
		Cols: 80, Rows: 24, SandboxPath: sandbox,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		dpty.Kill()
		dpty.Close()
		DeleteState(sandbox)
	})

	// Write data, read echo back (cat mirrors input)
	payload := []byte("exact-bytes-test\n")
	_, err = dpty.Write(payload)
	require.NoError(t, err)

	buf := make([]byte, 4096)
	dpty.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := dpty.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "exact-bytes-test")

	// Resize should succeed without error
	require.NoError(t, dpty.Resize(200, 50))
	cols, rows, err := dpty.GetSize()
	require.NoError(t, err)
	assert.Equal(t, 200, cols)
	assert.Equal(t, 50, rows)
}

// TestPodDaemon_GracefulStopExitCode_Integration verifies SIGTERM yields
// non-zero exit and natural exit preserves the child's code.
func TestPodDaemon_GracefulStopExitCode_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)

	t.Run("sigterm", func(t *testing.T) {
		workspace, sandbox := shortWorkspace(t, "g1")
		mgr := &PodDaemonManager{
			sandboxesDir:  workspace,
			runnerBinPath: binPath,
		}

		dpty, _, err := mgr.CreateSession(CreateOpts{
			PodKey: "p", Agent: "test", Command: "sleep", Args: []string{"60"},
			WorkDir: sandbox, Env: os.Environ(),
			Cols: 80, Rows: 24, SandboxPath: sandbox,
		})
		require.NoError(t, err)
		t.Cleanup(func() { dpty.Close(); DeleteState(sandbox) })

		require.NoError(t, dpty.GracefulStop())
		code, err := dpty.Wait()
		require.NoError(t, err)
		assert.NotEqual(t, 0, code, "SIGTERM should produce non-zero exit")
	})

	t.Run("natural_exit_42", func(t *testing.T) {
		workspace, sandbox := shortWorkspace(t, "g2")
		mgr := &PodDaemonManager{
			sandboxesDir:  workspace,
			runnerBinPath: binPath,
		}

		dpty, _, err := mgr.CreateSession(CreateOpts{
			PodKey: "p", Agent: "test",
			Command: "/bin/sh", Args: []string{"-c", "sleep 1; exit 42"},
			WorkDir: sandbox, Env: os.Environ(),
			Cols: 80, Rows: 24, SandboxPath: sandbox,
		})
		require.NoError(t, err)
		t.Cleanup(func() { dpty.Close(); DeleteState(sandbox) })

		code, err := dpty.Wait()
		require.NoError(t, err)
		assert.Equal(t, 42, code)
	})
}
