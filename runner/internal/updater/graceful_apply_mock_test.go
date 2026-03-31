package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for executeUpdate with mock

func TestGracefulUpdater_ApplyUpdate_WithMock_RestartError_ExitsForServiceManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-apply-*")
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

	restartErr := errors.New("restart failed")
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, restartErr
	}))

	// Override exitFunc to capture the exit instead of actually exiting.
	var exitCode int
	exited := false
	g.exitFunc = func(code int) {
		exitCode = code
		exited = true
		// In production os.Exit never returns, but in tests we let it return
		// so we can assert on the code.
	}

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// executeUpdate should call exitFunc(1) after successful update + failed restart.
	// Non-zero exit code ensures systemd Restart=on-failure triggers a restart.
	_ = g.executeUpdate(context.Background())
	assert.True(t, exited, "expected process to exit after restart failure")
	assert.Equal(t, 1, exitCode, "expected non-zero exit so service manager restarts")
}

func TestGracefulUpdater_ApplyUpdate_WithMock_BackupFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-apply-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "nonexistent", "runner") // Invalid path

	mock := &MockReleaseDetector{}
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) { return execPath, nil }),
	)

	g := NewGracefulUpdater(u, nil)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// Apply will fail because execPath doesn't exist (updateBinary will fail)
	err = g.executeUpdate(context.Background())
	assert.Error(t, err)
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyUpdate_WithMock_NoPending(t *testing.T) {
	mock := &MockReleaseDetector{}
	u := New("1.0.0", WithReleaseDetector(mock))
	g := NewGracefulUpdater(u, nil)

	err := g.executeUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending update")
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyUpdate_WithMock_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-apply-*")
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

	restarted := false
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		restarted = true
		return 12345, nil
	}))

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.NoError(t, err)
	assert.True(t, restarted)
	assert.Equal(t, StateRestarting, g.State())

	// Verify binary was replaced
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "mock binary", string(content))
}
