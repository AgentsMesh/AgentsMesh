package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for executeUpdate and related functionality

func TestGracefulUpdater_ApplyUpdate_NoPending(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	err := g.executeUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending update to apply")
}

func TestGracefulUpdater_ApplyUpdate_UpdateBinaryFails(t *testing.T) {
	mock := &MockReleaseDetector{
		UpdateError: fmt.Errorf("update failed"),
	}

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)
	g := NewGracefulUpdater(u, nil)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.Error(t, err)
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyUpdate_WithRestartFunc(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{}
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	var restartCalled bool
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		restartCalled = true
		return 12345, nil
	}))

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.NoError(t, err)
	assert.True(t, restartCalled)
}

func TestGracefulUpdater_ApplyUpdate_RestartFuncError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{}
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, fmt.Errorf("restart failed")
	}))

	// After a successful update + failed restart, the process exits so the
	// service manager restarts with the new binary.
	exited := false
	g.exitFunc = func(code int) { exited = true }

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	_ = g.executeUpdate(context.Background())
	assert.True(t, exited)
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

	cancelCalled := false
	g.mu.Lock()
	g.state = StateDraining
	g.draining = true
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.cancelDrain = func() { cancelCalled = true }
	g.mu.Unlock()

	g.CancelPendingUpdate()

	assert.True(t, cancelCalled)
	assert.False(t, g.IsDraining())
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ExecuteUpdate_ContextCancelled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	// Mock that blocks until context is cancelled
	mock := &MockReleaseDetector{
		UpdateError: context.Canceled,
	}
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	g := NewGracefulUpdater(u, nil)
	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = g.executeUpdate(ctx)
	assert.Error(t, err)
	assert.Equal(t, StateIdle, g.State())
}
