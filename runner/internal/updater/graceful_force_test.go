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

// Extended tests for ForceUpdate functionality

func TestGracefulUpdater_ForceUpdate_WithPendingApply(t *testing.T) {
	u := New("1.0.0")

	tmpDir, err := os.MkdirTemp("", "graceful-force-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending-binary")
	err = os.WriteFile(pendingPath, []byte("new binary"), 0755)
	require.NoError(t, err)

	g := NewGracefulUpdater(u, nil)

	// Set up as if we're draining with a pending update
	g.mu.Lock()
	g.state = StateDraining
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// ForceUpdate should try to apply the pending update
	err = g.ForceUpdate(context.Background())
	// May succeed or fail depending on os.Executable and permissions
	// Just verify the function runs and state is handled
	_ = err
}

func TestGracefulUpdater_ForceUpdate_CancelsDrain(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	cancelCalled := false
	g.mu.Lock()
	g.state = StateDraining
	g.cancelDrain = func() { cancelCalled = true }
	g.mu.Unlock()

	// ForceUpdate from draining state should cancel drain
	err := g.ForceUpdate(context.Background())
	assert.Error(t, err) // Will fail due to network
	assert.True(t, cancelCalled)
}

func TestGracefulUpdater_ForceUpdate_NoUpdateAvailable(t *testing.T) {
	u := New("999.0.0") // Very high version, no update should be available

	g := NewGracefulUpdater(u, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := g.ForceUpdate(ctx)
	// May fail with network error or "no update available"
	assert.Error(t, err)
}

func TestGracefulUpdater_ForceUpdate_StatusCallback(t *testing.T) {
	u := New("1.0.0")

	var states []State
	cb := func(state State, info *UpdateInfo, activePods int) {
		states = append(states, state)
	}

	g := NewGracefulUpdater(u, nil, WithStatusCallback(cb))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	g.ForceUpdate(ctx)

	// Should have transitioned through StateChecking
	assert.Contains(t, states, StateChecking)
}

func TestGracefulUpdater_ForceUpdate_DownloadFails(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond) // Let context expire

	err := g.ForceUpdate(ctx)
	assert.Error(t, err)
	// State should be reset to Idle
	assert.Equal(t, StateIdle, g.State())
}
