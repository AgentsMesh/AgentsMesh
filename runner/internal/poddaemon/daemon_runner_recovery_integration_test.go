//go:build integration && !windows

package poddaemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDaemonRecovery_CreateDetachRecover_Integration verifies the full
// create → detach → recover → re-attach cycle for a single daemon session.
func TestDaemonRecovery_CreateDetachRecover_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "r1")

	mgr := &PodDaemonManager{sandboxesDir: workspace, runnerBinPath: binPath}
	opts := CreateOpts{
		PodKey: "recover-1", Agent: "test", Command: "cat",
		WorkDir: sandbox, Env: os.Environ(), Cols: 80, Rows: 24, SandboxPath: sandbox,
	}

	dpty, state, err := mgr.CreateSession(opts)
	require.NoError(t, err)
	t.Cleanup(func() { DeleteState(sandbox) })

	// Write data and verify I/O works
	_, err = dpty.Write([]byte("before-detach\n"))
	require.NoError(t, err)
	buf := make([]byte, 4096)
	dpty.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := dpty.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "before-detach")

	childPid := dpty.Pid()
	require.NoError(t, dpty.Close()) // detach (simulate runner shutdown)
	time.Sleep(200 * time.Millisecond)

	// Recover
	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "recover-1", sessions[0].PodKey)
	assert.Equal(t, "cat", sessions[0].Command)
	assert.Equal(t, sandbox, sessions[0].SandboxPath)
	assert.Equal(t, state.IPCAddr, sessions[0].IPCAddr)

	// Re-attach and verify daemon is still alive
	dpty2, err := mgr.AttachSession(sessions[0])
	require.NoError(t, err)
	defer func() { dpty2.Kill(); dpty2.Close() }()
	assert.Equal(t, childPid, dpty2.Pid())

	_, err = dpty2.Write([]byte("after-recover\n"))
	require.NoError(t, err)
	dpty2.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err = dpty2.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "after-recover")
}

// TestDaemonRecovery_MultipleSessionsRecovery_Integration creates 3 daemons,
// detaches all, recovers all, and verifies each is independently alive.
func TestDaemonRecovery_MultipleSessionsRecovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	binPath := buildTestRunner(t)
	workspace := t.TempDir()
	mgr := &PodDaemonManager{sandboxesDir: workspace, runnerBinPath: binPath}

	keys := []string{"cat-1", "cat-2", "cat-3"}
	sandboxes := make([]string, 3)
	for i, key := range keys {
		sb := filepath.Join(workspace, key)
		require.NoError(t, os.MkdirAll(sb, 0755))
		sandboxes[i] = sb
		opts := CreateOpts{
			PodKey: key, Agent: "test", Command: "cat",
			WorkDir: sb, Env: os.Environ(), Cols: 80, Rows: 24, SandboxPath: sb,
		}
		dpty, _, err := mgr.CreateSession(opts)
		require.NoError(t, err)
		t.Cleanup(func() { DeleteState(sb) })
		require.NoError(t, dpty.Close()) // detach
	}
	time.Sleep(200 * time.Millisecond)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 3)

	found := map[string]bool{}
	for _, s := range sessions {
		found[s.PodKey] = true
	}
	for _, key := range keys {
		assert.True(t, found[key], "missing recovered session: %s", key)
	}

	// Attach to each and verify alive
	buf := make([]byte, 4096)
	for _, s := range sessions {
		dpty, err := mgr.AttachSession(s)
		require.NoError(t, err, "attach %s", s.PodKey)
		_, err = dpty.Write([]byte("ping-" + s.PodKey + "\n"))
		require.NoError(t, err)
		dpty.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := dpty.Read(buf)
		require.NoError(t, err)
		assert.Contains(t, string(buf[:n]), "ping-"+s.PodKey)
		dpty.Kill()
		dpty.Close()
	}
}

// TestDaemonRecovery_DeadDaemonCleanup_Integration kills a daemon process
// directly and verifies that recovery either skips it or attach fails,
// and CleanupSession removes the state file.
func TestDaemonRecovery_DeadDaemonCleanup_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "dd")
	mgr := &PodDaemonManager{sandboxesDir: workspace, runnerBinPath: binPath}

	opts := CreateOpts{
		PodKey: "dead-1", Agent: "test", Command: "cat",
		WorkDir: sandbox, Env: os.Environ(), Cols: 80, Rows: 24, SandboxPath: sandbox,
	}
	dpty, state, err := mgr.CreateSession(opts)
	require.NoError(t, err)
	t.Cleanup(func() { DeleteState(sandbox) })

	// Kill the daemon process directly via OS signal
	require.Greater(t, state.DaemonPID, 0)
	require.NoError(t, syscall.Kill(state.DaemonPID, syscall.SIGKILL))
	dpty.Close()
	time.Sleep(500 * time.Millisecond) // wait for process to die

	// State file still exists on disk (daemon didn't clean up)
	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1, "state file should still be found")

	// But attaching should fail because daemon is dead
	_, err = mgr.AttachSession(sessions[0])
	assert.Error(t, err, "attach to dead daemon should fail")

	// CleanupSession removes the stale state file
	require.NoError(t, mgr.CleanupSession(sandbox))
	_, err = LoadState(sandbox)
	assert.Error(t, err, "state file should be gone after cleanup")
}

// TestDaemonRecovery_StateFilePersistence_Integration verifies the state file
// contents and that a brand-new Manager instance can recover from it.
func TestDaemonRecovery_StateFilePersistence_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	binPath := buildTestRunner(t)
	workspace, sandbox := shortWorkspace(t, "sf")
	mgr := &PodDaemonManager{sandboxesDir: workspace, runnerBinPath: binPath}

	opts := CreateOpts{
		PodKey: "persist-1", Agent: "test-agent", Command: "cat",
		Args: []string{"-v"}, WorkDir: sandbox, Env: os.Environ(),
		Cols: 120, Rows: 40, SandboxPath: sandbox,
	}
	dpty, _, err := mgr.CreateSession(opts)
	require.NoError(t, err)
	t.Cleanup(func() { dpty.Kill(); dpty.Close(); DeleteState(sandbox) })

	// Read and verify raw state file
	raw, err := os.ReadFile(StatePath(sandbox))
	require.NoError(t, err)
	var state PodDaemonState
	require.NoError(t, json.Unmarshal(raw, &state))
	assert.Equal(t, "persist-1", state.PodKey)
	assert.Equal(t, "test-agent", state.Agent)
	assert.Equal(t, "cat", state.Command)
	assert.Equal(t, []string{"-v"}, state.Args)
	assert.Equal(t, 120, state.Cols)
	assert.Equal(t, 40, state.Rows)
	assert.NotEmpty(t, state.IPCAddr)
	assert.NotEmpty(t, state.AuthToken)

	// Detach
	childPid := dpty.Pid()
	require.NoError(t, dpty.Close())
	time.Sleep(200 * time.Millisecond)

	// Simulate runner restart: create a brand new Manager
	mgr2 := &PodDaemonManager{sandboxesDir: workspace, runnerBinPath: binPath}
	sessions, err := mgr2.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	dpty2, err := mgr2.AttachSession(sessions[0])
	require.NoError(t, err)
	defer func() { dpty2.Kill(); dpty2.Close() }()
	assert.Equal(t, childPid, dpty2.Pid())

	buf := make([]byte, 4096)
	_, err = dpty2.Write([]byte("new-manager\n"))
	require.NoError(t, err)
	dpty2.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := dpty2.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "new-manager")
}
