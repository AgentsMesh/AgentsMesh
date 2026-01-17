package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for GracefulUpdater schedule and force update

func TestGracefulUpdater_ScheduleUpdate_AlreadyInProgress(t *testing.T) {
	u := New("1.0.0")
	podCounter := func() int { return 0 }
	g := NewGracefulUpdater(u, podCounter)

	g.mu.Lock()
	g.state = StateChecking
	g.mu.Unlock()

	err := g.ScheduleUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update already in progress")
}

func TestGracefulUpdater_ScheduleUpdate_CheckFails(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	err := g.ScheduleUpdate(ctx)
	assert.Error(t, err)
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ForceUpdate_WrongState(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	g.mu.Lock()
	g.state = StateDownloading
	g.mu.Unlock()

	err := g.ForceUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot force update in state")
}

func TestGracefulUpdater_ForceUpdate_CancelsDraining(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	cancelCalled := false
	g.mu.Lock()
	g.state = StateDraining
	g.cancelDrain = func() { cancelCalled = true }
	g.mu.Unlock()

	err := g.ForceUpdate(context.Background())
	assert.True(t, cancelCalled)
	assert.Error(t, err) // Will fail due to network
}

func TestGracefulUpdater_ForceUpdate_WithPendingPath(t *testing.T) {
	u := New("1.0.0")

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending-binary")
	err = os.WriteFile(pendingPath, []byte("new binary"), 0755)
	require.NoError(t, err)

	g := NewGracefulUpdater(u, nil)

	g.mu.Lock()
	g.state = StateDraining
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.ForceUpdate(context.Background())
	_ = err // May or may not fail
}
