//go:build !windows

package poddaemon

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrphanCheckerTriggersOnDeletedState(t *testing.T) {
	dir := t.TempDir()

	state := &PodDaemonState{
		PodKey:      "orphan-test",
		SandboxPath: dir,
	}
	require.NoError(t, SaveState(state))

	proc := newMockProcess(1)
	d := &daemonServer{
		proc:                proc,
		exitDone:            make(chan struct{}),
		orphanCh:            make(chan struct{}),
		log:                 slog.Default(),
		state:               state,
		orphanCheckInterval: 50 * time.Millisecond, // fast check for testing
	}

	// Delete the state file so orphanChecker detects it
	os.Remove(StatePath(dir))

	done := make(chan struct{})
	go func() {
		d.orphanChecker()
		close(done)
	}()

	select {
	case <-done:
		// orphanChecker returned
		select {
		case <-d.orphanCh:
			// success — channel was closed
		default:
			t.Fatal("orphanCh was not closed")
		}
	case <-time.After(2 * time.Second):
		close(d.exitDone) // unblock
		t.Fatal("orphanChecker did not trigger in time")
	}
}

func TestOrphanCheckerStopsOnProcessExit(t *testing.T) {
	dir := t.TempDir()

	state := &PodDaemonState{
		PodKey:      "exit-test",
		SandboxPath: dir,
	}
	require.NoError(t, SaveState(state))

	proc := newMockProcess(1)
	d := &daemonServer{
		proc:                proc,
		exitDone:            make(chan struct{}),
		orphanCh:            make(chan struct{}),
		log:                 slog.Default(),
		state:               state,
		orphanCheckInterval: 50 * time.Millisecond,
	}

	done := make(chan struct{})
	go func() {
		d.orphanChecker()
		close(done)
	}()

	// Signal process exit
	close(d.exitDone)

	select {
	case <-done:
		// orphanChecker exited due to exitDone
	case <-time.After(2 * time.Second):
		t.Fatal("orphanChecker did not stop on process exit")
	}

	// orphanCh should NOT be closed
	select {
	case <-d.orphanCh:
		t.Fatal("orphanCh should not be closed when process exits normally")
	default:
		// expected
	}
}

func TestOrphanCheckerStateFileExists(t *testing.T) {
	dir := t.TempDir()

	state := &PodDaemonState{
		PodKey:      "alive-test",
		SandboxPath: dir,
	}
	require.NoError(t, SaveState(state))

	proc := newMockProcess(1)
	d := &daemonServer{
		proc:                proc,
		exitDone:            make(chan struct{}),
		orphanCh:            make(chan struct{}),
		log:                 slog.Default(),
		state:               state,
		orphanCheckInterval: 50 * time.Millisecond,
	}

	done := make(chan struct{})
	go func() {
		d.orphanChecker()
		close(done)
	}()

	// Let a few ticks pass
	time.Sleep(200 * time.Millisecond)

	// orphanCh should NOT be closed
	select {
	case <-d.orphanCh:
		t.Fatal("orphanCh should not be closed when state file exists")
	default:
		// expected
	}

	// Clean up
	close(d.exitDone)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("orphanChecker did not stop")
	}

	_, err := os.Stat(StatePath(dir))
	assert.NoError(t, err, "state file should still exist")
}
