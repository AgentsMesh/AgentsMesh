package poddaemon

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()

	state := &PodDaemonState{
		PodKey:         "test-pod-123",
		AgentType:      "claude-code",
		IPCPath:        IPCPath(dir, "test-pod-123"),
		DaemonPID:      12345,
		SandboxPath:    dir,
		WorkDir:        "/tmp/work",
		RepositoryURL:  "https://github.com/test/repo",
		Branch:         "main",
		TicketSlug:     "TICK-42",
		Command:        "claude",
		Args:           []string{"--yes"},
		Cols:           120,
		Rows:           40,
		StartedAt:      time.Now().UTC().Truncate(time.Millisecond),
		VTHistoryLimit: 5000,
	}

	require.NoError(t, SaveState(state))

	loaded, err := LoadState(dir)
	require.NoError(t, err)

	// Compare truncated time since JSON doesn't preserve nanosecond precision
	loaded.StartedAt = loaded.StartedAt.Truncate(time.Millisecond)
	assert.Equal(t, state, loaded)
}

func TestStateDelete(t *testing.T) {
	dir := t.TempDir()

	state := &PodDaemonState{
		PodKey:      "delete-me",
		SandboxPath: dir,
	}
	require.NoError(t, SaveState(state))

	// File should exist
	_, err := os.Stat(StatePath(dir))
	require.NoError(t, err)

	require.NoError(t, DeleteState(dir))

	// File should be gone
	_, err = os.Stat(StatePath(dir))
	assert.True(t, os.IsNotExist(err))
}

func TestStateDeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, DeleteState(dir))
}

func TestLoadStateNotFound(t *testing.T) {
	_, err := LoadState("/nonexistent/path")
	assert.Error(t, err)
}

func TestStatePath(t *testing.T) {
	got := StatePath("/my/sandbox")
	assert.Contains(t, got, "pod_daemon.json")
}

// TestStateConcurrentSaveLoad verifies atomic Save + Load doesn't produce
// corrupt data under concurrent access.
func TestStateConcurrentSaveLoad(t *testing.T) {
	dir := t.TempDir()

	state := &PodDaemonState{
		PodKey:      "concurrent-test",
		SandboxPath: dir,
		Command:     "echo",
		Cols:        80,
		Rows:        24,
	}

	var wg sync.WaitGroup
	const iterations = 50

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			s := *state
			s.DaemonPID = i
			SaveState(&s)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			loaded, err := LoadState(dir)
			if err != nil {
				continue // file might not exist yet on first iteration
			}
			assert.Equal(t, "concurrent-test", loaded.PodKey)
			assert.GreaterOrEqual(t, loaded.DaemonPID, 0)
			assert.Less(t, loaded.DaemonPID, iterations)
		}
	}()

	wg.Wait()
}
