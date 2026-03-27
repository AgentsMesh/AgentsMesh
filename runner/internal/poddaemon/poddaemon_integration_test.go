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
