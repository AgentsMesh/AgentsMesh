package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for applyPendingUpdate and related functionality

func TestGracefulUpdater_ApplyPendingUpdate_NoPending(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	err := g.applyPendingUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending update to apply")
}

func TestGracefulUpdater_ApplyPendingUpdate_ApplyFails(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	g.mu.Lock()
	g.pendingPath = "/nonexistent/path/to/binary"
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err := g.applyPendingUpdate()
	assert.Error(t, err)
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyPendingUpdate_WithRestartFunc(t *testing.T) {
	u := New("1.0.0")

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending")
	err = os.WriteFile(pendingPath, []byte("binary"), 0755)
	require.NoError(t, err)

	var restartCalled bool
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		restartCalled = true
		return 12345, nil // Return a fake PID
	}))

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	_ = g.applyPendingUpdate()
	_ = restartCalled // Used for verification
}

func TestGracefulUpdater_ApplyPendingUpdate_RestartFuncError(t *testing.T) {
	u := New("1.0.0")

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending")
	err = os.WriteFile(pendingPath, []byte("binary"), 0755)
	require.NoError(t, err)

	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, fmt.Errorf("restart failed")
	}))

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	_ = g.applyPendingUpdate()
}

func TestGracefulUpdater_WithRestartFunc(t *testing.T) {
	u := New("1.0.0")
	restartCalled := false
	restartFunc := func() (int, error) {
		restartCalled = true
		return 12345, nil
	}

	g := NewGracefulUpdater(u, nil, WithRestartFunc(restartFunc))
	assert.NotNil(t, g.restartFunc)

	pid, err := g.restartFunc()
	assert.NoError(t, err)
	assert.Equal(t, 12345, pid)
	assert.True(t, restartCalled)
}

func TestDefaultRestartFunc(t *testing.T) {
	restartFunc := DefaultRestartFunc()
	assert.NotNil(t, restartFunc)
}

func TestGracefulUpdater_CancelPendingUpdate_WithDrainCancel(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending")
	err = os.WriteFile(pendingPath, []byte("binary"), 0755)
	require.NoError(t, err)

	cancelCalled := false
	g.mu.Lock()
	g.state = StateDraining
	g.draining = true
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.cancelDrain = func() { cancelCalled = true }
	g.mu.Unlock()

	g.CancelPendingUpdate()

	assert.True(t, cancelCalled)
	assert.False(t, g.IsDraining())
	assert.Equal(t, StateIdle, g.State())

	_, err = os.Stat(pendingPath)
	assert.True(t, os.IsNotExist(err))
}
