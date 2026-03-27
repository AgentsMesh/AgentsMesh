//go:build integration && !windows

package poddaemon

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateSessionAndIO spawns a real daemon, sends input, reads output,
// detaches, re-attaches, and verifies the session persists.
func TestCreateSessionAndIO(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "io")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "p",
		Agent:       "test",
		Command:     "cat",
		WorkDir:     sandbox,
		Env:         os.Environ(),
		Cols:        80,
		Rows:        24,
		SandboxPath: sandbox,
	}

	dpty, state, err := mgr.CreateSession(opts)
	require.NoError(t, err, "CreateSession failed")
	require.NotNil(t, dpty)
	require.NotNil(t, state)

	t.Cleanup(func() {
		dpty.Kill()
		dpty.Close()
		DeleteState(sandbox)
	})

	t.Logf("daemon PID: %d, child PID: %d", state.DaemonPID, dpty.Pid())
	assert.Greater(t, dpty.Pid(), 0)
	assert.NotEmpty(t, state.IPCAddr, "daemon should have written IPC address")
	assert.NotEmpty(t, state.AuthToken, "session should have auth token")

	// --- Test I/O: write to cat, read echo back ---
	_, err = dpty.Write([]byte("hello world\n"))
	require.NoError(t, err)

	buf := make([]byte, 4096)
	dpty.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := dpty.Read(buf)
	require.NoError(t, err, "failed to read output")
	output := string(buf[:n])
	t.Logf("first read: %q", output)
	assert.Contains(t, output, "hello world")

	// --- Test Resize ---
	require.NoError(t, dpty.Resize(120, 40))

	// --- Test Detach + Re-attach ---
	childPid := dpty.Pid()
	err = dpty.Close()
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	dpty2, err := mgr.AttachSession(state)
	require.NoError(t, err, "AttachSession failed after detach")
	require.NotNil(t, dpty2)
	defer func() {
		dpty2.Kill()
		dpty2.Close()
	}()

	assert.Equal(t, childPid, dpty2.Pid(), "child PID should persist across re-attach")

	_, err = dpty2.Write([]byte("after reattach\n"))
	require.NoError(t, err)

	dpty2.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err = dpty2.Read(buf)
	require.NoError(t, err, "failed to read after re-attach")
	output = string(buf[:n])
	t.Logf("post-reattach read: %q", output)
	assert.Contains(t, output, "after reattach")
}

// TestCreateSessionExitCode verifies daemon reports child's exit code.
func TestCreateSessionExitCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "ex")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "p",
		Agent:       "test",
		Command:     "/bin/sh",
		Args:        []string{"-c", "sleep 1; exit 42"},
		WorkDir:     sandbox,
		Env:         os.Environ(),
		Cols:        80,
		Rows:        24,
		SandboxPath: sandbox,
	}

	dpty, _, err := mgr.CreateSession(opts)
	require.NoError(t, err)
	t.Cleanup(func() {
		dpty.Close()
		DeleteState(sandbox)
	})

	code, err := dpty.Wait()
	require.NoError(t, err)
	assert.Equal(t, 42, code)
}

// TestCreateSessionGracefulStop verifies SIGTERM delivery to child.
func TestCreateSessionGracefulStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "gs")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "p",
		Agent:       "test",
		Command:     "sleep",
		Args:        []string{"3600"},
		WorkDir:     sandbox,
		Env:         os.Environ(),
		Cols:        80,
		Rows:        24,
		SandboxPath: sandbox,
	}

	dpty, _, err := mgr.CreateSession(opts)
	require.NoError(t, err)
	t.Cleanup(func() {
		dpty.Close()
		DeleteState(sandbox)
	})

	require.NoError(t, dpty.GracefulStop())

	code, err := dpty.Wait()
	require.NoError(t, err)
	t.Logf("exit code after GracefulStop: %d", code)
	assert.NotEqual(t, 0, code, "child should not exit cleanly after SIGTERM")
}

// TestRecoverSessionsIntegration creates a daemon, detaches, and recovers via scan.
func TestRecoverSessionsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "rc")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "p",
		Agent:       "test",
		Command:     "cat",
		WorkDir:     sandbox,
		Env:         os.Environ(),
		Cols:        80,
		Rows:        24,
		SandboxPath: sandbox,
	}

	dpty, state, err := mgr.CreateSession(opts)
	require.NoError(t, err)
	t.Cleanup(func() {
		DeleteState(sandbox)
	})

	childPid := dpty.Pid()
	t.Logf("created session, child PID: %d", childPid)

	require.NoError(t, dpty.Close())
	time.Sleep(200 * time.Millisecond)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "p", sessions[0].PodKey)
	assert.Equal(t, state.IPCAddr, sessions[0].IPCAddr)

	dpty2, err := mgr.AttachSession(sessions[0])
	require.NoError(t, err)
	defer func() {
		dpty2.Kill()
		dpty2.Close()
	}()

	assert.Equal(t, childPid, dpty2.Pid())

	_, err = dpty2.Write([]byte("recovered\n"))
	require.NoError(t, err)

	buf := make([]byte, 4096)
	dpty2.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := dpty2.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "recovered")
}
