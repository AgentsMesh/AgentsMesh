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

func TestRecoverSessionsEmpty(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestRecoverSessionsFindsState(t *testing.T) {
	dir := t.TempDir()

	// Create a sandbox with a state file
	sandbox := filepath.Join(dir, "sandbox-1")
	require.NoError(t, os.MkdirAll(sandbox, 0755))

	state := &PodDaemonState{
		PodKey:      "pod-1",
		SandboxPath: sandbox,
		Command:     "echo",
		Args:        []string{"hello"},
		Cols:        80,
		Rows:        24,
		StartedAt:   time.Now().Truncate(time.Millisecond),
	}
	require.NoError(t, SaveState(state))

	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "pod-1", sessions[0].PodKey)
}

func TestRecoverSessionsNonExistentDir(t *testing.T) {
	mgr, err := NewPodDaemonManager("/nonexistent/path")
	require.NoError(t, err)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	assert.Nil(t, sessions)
}

func TestCleanupSession(t *testing.T) {
	dir := t.TempDir()

	state := &PodDaemonState{
		PodKey:      "cleanup-me",
		SandboxPath: dir,
	}
	require.NoError(t, SaveState(state))

	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	require.NoError(t, mgr.CleanupSession(dir))

	_, err = os.Stat(StatePath(dir))
	assert.True(t, os.IsNotExist(err))
}

func TestNewPodDaemonManager(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)
	assert.NotNil(t, mgr)
	assert.Equal(t, dir, mgr.workspaceRoot)
	assert.NotEmpty(t, mgr.runnerBinPath) // should resolve to test binary
}

func TestAttachSessionSuccess(t *testing.T) {
	dir := t.TempDir()
	ipcPath := IPCPath(dir, "a")

	// Start a mock daemon listener
	listener, err := Listen(ipcPath)
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read Attach
		msgType, _, err := ReadMessage(conn)
		if err != nil || msgType != MsgAttach {
			return
		}

		// Send AttachAck
		ack := attachAckPayload{PID: 77, Cols: 100, Rows: 30, Alive: true}
		data, _ := json.Marshal(ack)
		WriteMessage(conn, MsgAttachAck, data)

		// Keep alive
		time.Sleep(1 * time.Second)
	}()

	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	state := &PodDaemonState{
		PodKey:  "attach-pod",
		IPCPath: ipcPath,
	}

	dpty, err := mgr.AttachSession(state)
	require.NoError(t, err)
	defer dpty.Close()

	assert.Equal(t, 77, dpty.Pid())
	cols, rows, _ := dpty.GetSize()
	assert.Equal(t, 100, cols)
	assert.Equal(t, 30, rows)
}

func TestAttachSessionFailsOnBadIPC(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	state := &PodDaemonState{
		PodKey:  "fail-pod",
		IPCPath: filepath.Join(dir, "nonexistent.sock"),
	}

	_, err = mgr.AttachSession(state)
	assert.Error(t, err)
}

func TestCreateSessionMissingSandboxPath(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	_, _, err = mgr.CreateSession(CreateOpts{
		PodKey:  "no-sandbox",
		Command: "echo",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox path is required")
}

func TestRecoverSessionsSkipsCorruptState(t *testing.T) {
	dir := t.TempDir()

	// Create a sandbox with a corrupt state file
	sandbox := filepath.Join(dir, "sandbox-corrupt")
	require.NoError(t, os.MkdirAll(sandbox, 0755))
	require.NoError(t, os.WriteFile(StatePath(sandbox), []byte("{invalid json"), 0644))

	// Create a sandbox with a valid state file
	sandboxOK := filepath.Join(dir, "sandbox-ok")
	require.NoError(t, os.MkdirAll(sandboxOK, 0755))
	state := &PodDaemonState{
		PodKey:      "ok-pod",
		SandboxPath: sandboxOK,
		Command:     "echo",
	}
	require.NoError(t, SaveState(state))

	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "ok-pod", sessions[0].PodKey)
}

func TestRecoverSessionsSkipsFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a regular file (not a directory) — should be skipped
	require.NoError(t, os.WriteFile(filepath.Join(dir, "not-a-dir.txt"), []byte("hello"), 0644))

	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestCleanupSessionNonExistent(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	// Cleaning up a non-existent session should not error
	err = mgr.CleanupSession(filepath.Join(dir, "ghost"))
	assert.NoError(t, err)
}

// TestRecoverSessionsMixedValidity verifies RecoverSessions handles a mix
// of valid, corrupt, missing, and empty state files plus non-directory entries.
func TestRecoverSessionsMixedValidity(t *testing.T) {
	dir := t.TempDir()

	// Valid session 1
	sandbox1 := filepath.Join(dir, "valid-1")
	require.NoError(t, os.MkdirAll(sandbox1, 0755))
	require.NoError(t, SaveState(&PodDaemonState{
		PodKey: "pod-1", Command: "echo", SandboxPath: sandbox1,
	}))

	// Corrupt JSON
	sandboxCorrupt := filepath.Join(dir, "corrupt")
	require.NoError(t, os.MkdirAll(sandboxCorrupt, 0755))
	require.NoError(t, os.WriteFile(StatePath(sandboxCorrupt), []byte("{bad"), 0644))

	// Empty directory (no state file)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "empty"), 0755))

	// Valid session 2
	sandbox2 := filepath.Join(dir, "valid-2")
	require.NoError(t, os.MkdirAll(sandbox2, 0755))
	require.NoError(t, SaveState(&PodDaemonState{
		PodKey: "pod-2", Command: "cat", SandboxPath: sandbox2,
	}))

	// Regular file (not a directory)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "not-a-dir.txt"), []byte("hello"), 0644))

	mgr, err := NewPodDaemonManager(dir)
	require.NoError(t, err)

	sessions, err := mgr.RecoverSessions()
	require.NoError(t, err)
	require.Len(t, sessions, 2, "should find exactly 2 valid sessions")

	keys := map[string]bool{}
	for _, s := range sessions {
		keys[s.PodKey] = true
	}
	assert.True(t, keys["pod-1"])
	assert.True(t, keys["pod-2"])
}
