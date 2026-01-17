package updater

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for applyPendingUpdate with mock

func TestGracefulUpdater_ApplyPendingUpdate_WithMock_RestartError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-apply-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	pendingPath := filepath.Join(tmpDir, "pending-binary")

	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(pendingPath, []byte("new binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{}
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	restartErr := errors.New("restart failed")
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, restartErr
	}))

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// Apply should fail when restart fails (now errors are propagated)
	err = g.applyPendingUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart failed")
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyPendingUpdate_WithMock_BackupFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-apply-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "nonexistent", "runner") // Invalid path
	pendingPath := filepath.Join(tmpDir, "pending-binary")

	err = os.WriteFile(pendingPath, []byte("new binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{}
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	g := NewGracefulUpdater(u, nil)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// Apply will fail because execPath doesn't exist
	err = g.applyPendingUpdate()
	assert.Error(t, err)
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyPendingUpdate_WithMock_NoPending(t *testing.T) {
	mock := &MockReleaseDetector{}
	u := New("1.0.0", WithReleaseDetector(mock))
	g := NewGracefulUpdater(u, nil)

	err := g.applyPendingUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending update")
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyPendingUpdate_WithMock_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-apply-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	pendingPath := filepath.Join(tmpDir, "pending-binary")

	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(pendingPath, []byte("new binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{}
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	restarted := false
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		restarted = true
		return 12345, nil // Return a fake PID
	}))

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.applyPendingUpdate()
	assert.NoError(t, err)
	assert.True(t, restarted)
	assert.Equal(t, StateRestarting, g.State())

	// Verify binary was replaced
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "new binary", string(content))
}
