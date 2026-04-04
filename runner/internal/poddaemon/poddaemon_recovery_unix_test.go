//go:build integration && !windows

package poddaemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// isProcessAlive checks if a process is alive (not zombie, not exited).
// Unlike kill -0, this correctly reports zombie processes as dead.
func isProcessAlive(pid int) bool {
	out, err := exec.Command("ps", "-o", "state=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(out))
	if state == "" {
		return false
	}
	// "Z" = zombie on macOS/Linux
	return state[0] != 'Z'
}

// TestDaemonSurvivesParentDeath creates a daemon, then simulates the parent
// Runner being killed (by simply disconnecting — the parent process in this
// test IS the test process). It then verifies:
//  1. The daemon process is still alive after parent disconnects.
//  2. A fresh PodDaemonManager can scan and recover the session.
//  3. I/O works normally after recovery.
//
// This is the core scenario for session persistence across Runner restarts.
func TestDaemonSurvivesParentDeath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "survive")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	// Phase 1: Create session (simulates Runner A creating a pod)
	opts := CreateOpts{
		PodKey:      "persist",
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

	daemonPID := state.DaemonPID
	childPID := dpty.Pid()
	ipcAddr := state.IPCAddr
	t.Logf("daemon PID: %d, child PID: %d, IPC: %s", daemonPID, childPID, ipcAddr)

	t.Cleanup(func() {
		// Final cleanup: kill daemon if still alive
		DeleteState(sandbox)
		time.Sleep(200 * time.Millisecond)
		process.KillProcessTree(daemonPID)
	})

	// Verify initial I/O works
	_, err = dpty.Write([]byte("before-kill\n"))
	require.NoError(t, err)

	buf := make([]byte, 4096)
	dpty.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := dpty.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "before-kill")

	// Phase 2: Simulate Runner death — just close the IPC connection (Detach).
	// In real life, Runner crashing would cause the connection to drop.
	err = dpty.Close()
	require.NoError(t, err)
	t.Log("parent detached (simulating Runner death)")

	// Give daemon time to notice the disconnection
	time.Sleep(500 * time.Millisecond)

	// Phase 3: Verify daemon is still alive
	inspector := process.DefaultInspector()
	assert.True(t, inspector.IsRunning(daemonPID),
		"daemon process (PID %d) should still be alive after parent detaches", daemonPID)
	assert.True(t, inspector.IsRunning(childPID),
		"child process (PID %d) should still be alive after parent detaches", childPID)

	// Phase 4: Fresh manager recovers sessions (simulates Runner B starting)
	mgr2 := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	sessions, err := mgr2.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1, "should find exactly one recoverable session")
	assert.Equal(t, "persist", sessions[0].PodKey)
	assert.Equal(t, ipcAddr, sessions[0].IPCAddr)

	// Phase 5: Attach to surviving daemon
	dpty2, err := mgr2.AttachSession(sessions[0])
	require.NoError(t, err, "AttachSession to surviving daemon failed")
	defer func() {
		dpty2.Kill()
		dpty2.Close()
	}()

	assert.Equal(t, childPID, dpty2.Pid(),
		"child PID should persist across parent death and recovery")

	// Phase 6: Verify I/O works after recovery
	_, err = dpty2.Write([]byte("after-recovery\n"))
	require.NoError(t, err)

	dpty2.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err = dpty2.Read(buf)
	require.NoError(t, err, "failed to read after recovery")
	assert.Contains(t, string(buf[:n]), "after-recovery")

	t.Log("full kill-restart-recover cycle succeeded")
}

// TestDaemonSurvivesMultipleReattachCycles verifies that a daemon can handle
// multiple detach→reattach cycles without leaking resources or corrupting I/O.
func TestDaemonSurvivesMultipleReattachCycles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "multi")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "multi",
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
	daemonPID := state.DaemonPID
	childPID := dpty.Pid()

	t.Cleanup(func() {
		DeleteState(sandbox)
		time.Sleep(200 * time.Millisecond)
		process.KillProcessTree(daemonPID)
	})

	const cycles = 5
	buf := make([]byte, 4096)

	for i := range cycles {
		// Write data
		msg := strings.Repeat("x", 10) + "\n"
		_, err = dpty.Write([]byte(msg))
		require.NoError(t, err, "write failed at cycle %d", i)

		dpty.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := dpty.Read(buf)
		require.NoError(t, err, "read failed at cycle %d", i)
		assert.Contains(t, string(buf[:n]), strings.Repeat("x", 10))

		// Detach (simulate Runner crash)
		dpty.Close()
		time.Sleep(300 * time.Millisecond)

		// Recover and reattach
		sessions, err := mgr.RecoverSessions()
		require.NoError(t, err)
		require.Len(t, sessions, 1)

		dpty, err = mgr.AttachSession(sessions[0])
		require.NoError(t, err, "attach failed at cycle %d", i)
		assert.Equal(t, childPID, dpty.Pid(), "PID should be stable across cycles")
	}

	// Final write after all cycles
	_, err = dpty.Write([]byte("final\n"))
	require.NoError(t, err)
	dpty.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := dpty.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "final")

	dpty.Kill()
	dpty.Close()
}

// TestRecoveredSessionResize verifies that Resize works on a recovered session.
func TestRecoveredSessionResize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "rsz")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "rsz",
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
	daemonPID := state.DaemonPID

	t.Cleanup(func() {
		DeleteState(sandbox)
		time.Sleep(200 * time.Millisecond)
		process.KillProcessTree(daemonPID)
	})

	// Detach
	dpty.Close()
	time.Sleep(300 * time.Millisecond)

	// Recover
	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	dpty2, err := mgr.AttachSession(sessions[0])
	require.NoError(t, err)
	defer func() {
		dpty2.Kill()
		dpty2.Close()
	}()

	// Resize on recovered session should work
	require.NoError(t, dpty2.Resize(120, 40))

	cols, rows, err := dpty2.GetSize()
	require.NoError(t, err)
	assert.Equal(t, 120, cols)
	assert.Equal(t, 40, rows)
}

// TestRecoveredSessionGracefulStop verifies GracefulStop works after recovery.
func TestRecoveredSessionGracefulStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "gstop")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "gstop",
		Agent:       "test",
		Command:     "sleep",
		Args:        []string{"3600"},
		WorkDir:     sandbox,
		Env:         os.Environ(),
		Cols:        80,
		Rows:        24,
		SandboxPath: sandbox,
	}

	dpty, state, err := mgr.CreateSession(opts)
	require.NoError(t, err)
	daemonPID := state.DaemonPID

	t.Cleanup(func() {
		DeleteState(sandbox)
		time.Sleep(200 * time.Millisecond)
		process.KillProcessTree(daemonPID)
	})

	// Detach
	dpty.Close()
	time.Sleep(300 * time.Millisecond)

	// Recover
	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	dpty2, err := mgr.AttachSession(sessions[0])
	require.NoError(t, err)

	// GracefulStop on recovered session
	require.NoError(t, dpty2.GracefulStop())

	code, err := dpty2.Wait()
	require.NoError(t, err)
	t.Logf("exit code after GracefulStop on recovered session: %d", code)
	assert.NotEqual(t, 0, code, "child should not exit cleanly after SIGTERM")

	dpty2.Close()
}

// TestOrphanCleanupAfterRecovery verifies that deleting the state file
// causes the daemon to exit (orphan protection still works after recovery).
func TestOrphanCleanupAfterRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "orph")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	// Set orphan check interval to 2 seconds for faster test feedback.
	env := append(os.Environ(), "_AGENTSMESH_ORPHAN_CHECK_INTERVAL_SEC=2")

	opts := CreateOpts{
		PodKey:      "orph",
		Agent:       "test",
		Command:     "sleep",
		Args:        []string{"3600"},
		WorkDir:     sandbox,
		Env:         env,
		Cols:        80,
		Rows:        24,
		SandboxPath: sandbox,
	}

	dpty, state, err := mgr.CreateSession(opts)
	require.NoError(t, err)
	daemonPID := state.DaemonPID

	// Detach
	dpty.Close()
	time.Sleep(300 * time.Millisecond)

	inspector := process.DefaultInspector()
	require.True(t, inspector.IsRunning(daemonPID), "daemon should be alive before cleanup")

	// Delete state file — triggers orphan protection
	require.NoError(t, DeleteState(sandbox))

	// Daemon should exit within ~2s (orphan check) + 5s (graceful stop timeout).
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if !isProcessAlive(daemonPID) {
			t.Log("daemon exited after state file deletion (orphan protection worked)")
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	// Force cleanup
	logData, _ := os.ReadFile(filepath.Join(sandbox, "pod_daemon.log"))
	t.Logf("daemon log:\n%s", string(logData))
	process.KillProcessTree(daemonPID)
	t.Fatal("daemon did not exit within expected time after state file deletion")
}
