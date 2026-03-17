package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for rollback and error propagation

func TestGracefulUpdater_ApplyUpdate_RestartErrorPropagation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-reliability-*")
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

	restartErr := errors.New("simulated restart failure")
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, restartErr
	}))

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// Apply should now return the restart error
	err = g.executeUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart failed")
	assert.Contains(t, err.Error(), "simulated restart failure")
	// State should be reset to Idle after failure
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyUpdate_HealthCheckFailed_Rollback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-reliability-*")
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

	healthCheckErr := errors.New("health check failed: process crashed")
	g := NewGracefulUpdater(u, nil,
		WithRestartFunc(func() (int, error) {
			return 99999, nil // Return a fake PID (process won't exist)
		}),
		WithHealthChecker(func(ctx context.Context, pid int) error {
			return healthCheckErr
		}),
		WithHealthTimeout(100*time.Millisecond),
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
	assert.Equal(t, StateIdle, g.State())

	// Verify rollback was attempted (binary should be restored)
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "old binary", string(content))
}

func TestGracefulUpdater_ExecuteUpdate_RestartFailed_NoBackup(t *testing.T) {
	// When CreateBackup fails (backupPath=""), restart fails, rollbackUpdate
	// should return "no backup available" and the error is still "restart failed".
	tmpDir, err := os.MkdirTemp("", "graceful-reliability-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{}
	// Make CreateBackup fail by pointing execPathFunc to a non-existent source
	// after the first call (updateBinary succeeds on the real path).
	callCount := 0
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) {
			callCount++
			if callCount == 1 {
				// CreateBackup calls execPathFunc — return invalid path so copyFile fails
				return filepath.Join(tmpDir, "nonexistent", "runner"), nil
			}
			// updateBinary calls execPathFunc — return real path
			return execPath, nil
		}),
	)

	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, errors.New("restart failed")
	}))

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart failed")
	assert.Equal(t, StateIdle, g.State())
}

func TestGracefulUpdater_ApplyUpdate_RestartFailed_Rollback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-reliability-*")
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

	g := NewGracefulUpdater(u, nil,
		WithRestartFunc(func() (int, error) {
			return 0, errors.New("failed to start new process")
		}),
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart failed")
	assert.Equal(t, StateIdle, g.State())

	// Verify rollback was attempted (binary should be restored)
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "old binary", string(content))
}
