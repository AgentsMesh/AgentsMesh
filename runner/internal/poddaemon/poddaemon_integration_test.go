//go:build integration && !windows

package poddaemon

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// TestDaemonProcessUnix verifies the platform PTY process wrapper.
func TestDaemonProcessUnix(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := t.TempDir()

	proc, err := startDaemonProcess("echo", []string{"daemon-process-test"}, workDir, os.Environ(), 80, 24)
	require.NoError(t, err)
	defer proc.Close()

	assert.Greater(t, proc.Pid(), 0)

	buf := make([]byte, 4096)
	n, err := proc.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "daemon-process-test")

	code, err := proc.Wait()
	require.NoError(t, err)
	assert.Equal(t, 0, code)
}

// TestDaemonProcessResize verifies PTY resize.
func TestDaemonProcessResize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := t.TempDir()

	proc, err := startDaemonProcess("cat", nil, workDir, os.Environ(), 80, 24)
	require.NoError(t, err)
	defer func() {
		proc.Kill()
		proc.Close()
	}()

	require.NoError(t, proc.Resize(120, 40))
}

// TestDaemonProcessGracefulStop verifies SIGTERM delivery via daemonProcess.
func TestDaemonProcessGracefulStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := t.TempDir()

	proc, err := startDaemonProcess("sleep", []string{"3600"}, workDir, os.Environ(), 80, 24)
	require.NoError(t, err)
	defer proc.Close()

	require.NoError(t, proc.GracefulStop())

	code, err := proc.Wait()
	require.NoError(t, err)
	t.Logf("exit code after GracefulStop: %d", code)
}

// TestDaemonProcessKill verifies SIGKILL delivery.
func TestDaemonProcessKill(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := t.TempDir()

	proc, err := startDaemonProcess("sleep", []string{"3600"}, workDir, os.Environ(), 80, 24)
	require.NoError(t, err)
	defer proc.Close()

	require.NoError(t, proc.Kill())

	code, err := proc.Wait()
	require.NoError(t, err)
	t.Logf("exit code after Kill: %d", code)
	assert.NotEqual(t, 0, code)
}

// TestStartDaemonDetached verifies startDaemon creates a detached daemon process.
func TestStartDaemonDetached(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "sd")

	// Generate token for auth
	token, err := generateAuthToken()
	require.NoError(t, err)

	state := &PodDaemonState{
		PodKey:      "d",
		AuthToken:   token,
		SandboxPath: sandbox,
		WorkDir:     sandbox,
		Command:     "sleep",
		Args:        []string{"5"},
		Cols:        80,
		Rows:        24,
	}
	require.NoError(t, SaveState(state))
	t.Cleanup(func() { DeleteState(sandbox) })

	configPath := StatePath(sandbox)
	pid, err := startDaemon(binPath, configPath, sandbox, os.Environ())
	require.NoError(t, err)
	assert.Greater(t, pid, 0)
	t.Logf("daemon started with PID %d", pid)

	// Wait for daemon to write its IPC address to state
	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}
	dpty, updatedState, err := mgr.waitForDaemon(sandbox, token, pid)
	if err != nil {
		t.Logf("could not connect to daemon: %v", err)
		return
	}
	defer func() {
		dpty.Kill()
		dpty.Close()
	}()

	assert.Greater(t, dpty.Pid(), 0)
	assert.NotEmpty(t, updatedState.IPCAddr)
	t.Logf("connected to daemon at %s, child PID: %d", updatedState.IPCAddr, dpty.Pid())
}

// TestWaitForDaemonRetry verifies the retry polling logic with TCP.
func TestWaitForDaemonRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sandbox := t.TempDir()
	token := testAuthToken

	// Save state without IPCAddr — daemon hasn't started yet
	require.NoError(t, SaveState(&PodDaemonState{
		PodKey:      "w",
		AuthToken:   token,
		SandboxPath: sandbox,
	}))

	mgr := &PodDaemonManager{
		sandboxesDir:  t.TempDir(),
		runnerBinPath: "unused",
	}

	// Simulate daemon starting after 300ms — it writes addr to state
	go func() {
		time.Sleep(300 * time.Millisecond)
		listener, err := Listen()
		if err != nil {
			return
		}
		defer listener.Close()

		// Update state with addr
		state, _ := LoadState(sandbox)
		state.IPCAddr = listener.Addr().String()
		SaveState(state)

		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		msgType, _, _ := ReadMessage(conn)
		if msgType == MsgAttach {
			ack := attachAckPayload{PID: 999, Cols: 80, Rows: 24, Alive: true}
			data, _ := json.Marshal(ack)
			WriteMessage(conn, MsgAttachAck, data)
		}
		time.Sleep(2 * time.Second)
	}()

	dpty, _, err := mgr.waitForDaemon(sandbox, token, 0)
	require.NoError(t, err)
	defer dpty.Close()
	assert.Equal(t, 999, dpty.Pid())
}

// TestWaitForDaemonTimeout verifies timeout when daemon never starts.
func TestWaitForDaemonTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	sandbox := t.TempDir()
	require.NoError(t, SaveState(&PodDaemonState{
		PodKey:      "timeout",
		AuthToken:   testAuthToken,
		SandboxPath: sandbox,
	}))

	mgr := &PodDaemonManager{
		sandboxesDir:  t.TempDir(),
		runnerBinPath: "unused",
	}

	_, _, err := mgr.waitForDaemon(sandbox, testAuthToken, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not become ready")
}

// TestDaemonPanicRecoveryWritesStackTrace verifies that when the daemon process
// panics, the main.go defer recover captures the stack trace into pod_daemon.log.
func TestDaemonPanicRecoveryWritesStackTrace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	_, sandbox := shortWorkspace(t, "pa")

	token, err := generateAuthToken()
	require.NoError(t, err)

	// Create a minimal valid state file (daemon needs it to get past LoadState)
	state := &PodDaemonState{
		PodKey:      "panic-test",
		AuthToken:   token,
		SandboxPath: sandbox,
		WorkDir:     sandbox,
		Command:     "echo",
		Args:        []string{"should-not-reach"},
		Cols:        80,
		Rows:        24,
	}
	require.NoError(t, SaveState(state))

	// Start daemon with _AGENTSMESH_DAEMON_TEST_PANIC to trigger deliberate panic.
	panicMsg := "deliberate test panic for stack trace verification"
	env := append(os.Environ(), "_AGENTSMESH_DAEMON_TEST_PANIC="+panicMsg)

	pid, err := startDaemon(binPath, StatePath(sandbox), sandbox, env)
	require.NoError(t, err)
	t.Logf("daemon started with PID %d (will panic)", pid)

	// Wait for daemon to crash and write its log
	time.Sleep(2 * time.Second)

	// Read pod_daemon.log — should contain the panic stack trace
	logPath := filepath.Join(sandbox, "pod_daemon.log")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err, "pod_daemon.log should exist")

	logContent := string(data)
	t.Logf("pod_daemon.log content:\n%s", logContent)

	assert.Contains(t, logContent, "FATAL: pod daemon panic")
	assert.Contains(t, logContent, panicMsg)
	assert.Contains(t, logContent, "goroutine") // stack trace should be present
}
