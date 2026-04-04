//go:build integration && windows

package poddaemon

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// TestDaemonSurvivesParentDeathWindows creates a daemon, disconnects
// (simulating Runner crash), and verifies:
//  1. Daemon process survives (CREATE_NEW_PROCESS_GROUP).
//  2. Fresh manager can scan and recover the session.
//  3. I/O works normally after recovery via TCP loopback.
func TestDaemonSurvivesParentDeathWindows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "survive")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "persist",
		Agent:       "test",
		Command:     "cmd.exe",
		Args:        []string{"/q"},
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
		DeleteState(sandbox)
		time.Sleep(200 * time.Millisecond)
		process.KillProcessTree(daemonPID)
	})

	// Verify initial I/O: send echo command, read output
	_, err = dpty.Write([]byte("echo before-kill\r\n"))
	require.NoError(t, err)

	buf := make([]byte, 4096)
	dpty.SetReadDeadline(time.Now().Add(5 * time.Second))
	var output strings.Builder
	for {
		n, readErr := dpty.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if strings.Contains(output.String(), "before-kill") || readErr != nil {
			break
		}
	}
	assert.Contains(t, output.String(), "before-kill")

	// Phase 2: Detach (simulates Runner death)
	err = dpty.Close()
	require.NoError(t, err)
	t.Log("parent detached (simulating Runner death)")
	time.Sleep(1 * time.Second)

	// Phase 3: Verify daemon is still alive
	inspector := process.DefaultInspector()
	assert.True(t, inspector.IsRunning(daemonPID),
		"daemon process (PID %d) should still be alive after parent detaches", daemonPID)

	// Phase 4: Fresh manager recovers sessions
	mgr2 := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	sessions, err := mgr2.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "persist", sessions[0].PodKey)

	// Phase 5: Attach to surviving daemon
	dpty2, err := mgr2.AttachSession(sessions[0])
	require.NoError(t, err, "AttachSession to surviving daemon failed")
	defer func() {
		dpty2.Kill()
		dpty2.Close()
	}()

	// Phase 6: Verify I/O works after recovery
	_, err = dpty2.Write([]byte("echo after-recovery\r\n"))
	require.NoError(t, err)

	output.Reset()
	dpty2.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		n, readErr := dpty2.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if strings.Contains(output.String(), "after-recovery") || readErr != nil {
			break
		}
	}
	assert.Contains(t, output.String(), "after-recovery")

	t.Log("full kill-restart-recover cycle succeeded on Windows")
}

// TestDaemonSurvivesMultipleReattachCyclesWindows verifies multiple
// detach→reattach cycles on Windows.
func TestDaemonSurvivesMultipleReattachCyclesWindows(t *testing.T) {
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
		Command:     "cmd.exe",
		Args:        []string{"/q"},
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

	buf := make([]byte, 4096)
	const cycles = 3

	for i := range cycles {
		marker := strings.Repeat("x", 5)
		_, err = dpty.Write([]byte("echo " + marker + "\r\n"))
		require.NoError(t, err, "write failed at cycle %d", i)

		var output strings.Builder
		dpty.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			n, readErr := dpty.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if strings.Contains(output.String(), marker) || readErr != nil {
				break
			}
		}
		assert.Contains(t, output.String(), marker)

		// Detach
		dpty.Close()
		time.Sleep(500 * time.Millisecond)

		// Recover and reattach
		sessions, err := mgr.RecoverSessions()
		require.NoError(t, err)
		require.Len(t, sessions, 1)

		dpty, err = mgr.AttachSession(sessions[0])
		require.NoError(t, err, "attach failed at cycle %d", i)
	}

	dpty.Kill()
	dpty.Close()
}

// TestRecoveredSessionResizeWindows verifies Resize on a recovered ConPTY session.
func TestRecoveredSessionResizeWindows(t *testing.T) {
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
		Command:     "cmd.exe",
		Args:        []string{"/q"},
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

	dpty.Close()
	time.Sleep(500 * time.Millisecond)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	dpty2, err := mgr.AttachSession(sessions[0])
	require.NoError(t, err)
	defer func() {
		dpty2.Kill()
		dpty2.Close()
	}()

	require.NoError(t, dpty2.Resize(120, 40))

	cols, rows, err := dpty2.GetSize()
	require.NoError(t, err)
	assert.Equal(t, 120, cols)
	assert.Equal(t, 40, rows)
}

// TestRecoveredSessionKillWindows verifies Kill on a recovered session.
func TestRecoveredSessionKillWindows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "kill")

	mgr := &PodDaemonManager{
		sandboxesDir:  workspace,
		runnerBinPath: binPath,
	}

	opts := CreateOpts{
		PodKey:      "kill",
		Agent:       "test",
		Command:     "cmd.exe",
		Args:        []string{"/c", "timeout /t 300"},
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

	dpty.Close()
	time.Sleep(500 * time.Millisecond)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	dpty2, err := mgr.AttachSession(sessions[0])
	require.NoError(t, err)

	require.NoError(t, dpty2.Kill())
	code, err := dpty2.Wait()
	require.NoError(t, err)
	t.Logf("exit code after Kill on recovered session: %d", code)

	dpty2.Close()
}
