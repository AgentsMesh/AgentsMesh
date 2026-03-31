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

func TestGracefulUpdater_ApplyUpdate_RestartErrorExits(t *testing.T) {
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

	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, errors.New("simulated restart failure")
	}))

	exited := false
	g.exitFunc = func(code int) { exited = true }

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// After successful update + failed restart, process exits for service manager
	_ = g.executeUpdate(context.Background())
	assert.True(t, exited)
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

func TestGracefulUpdater_ExecuteUpdate_RestartFailed_NoBackup_Exits(t *testing.T) {
	// When CreateBackup fails (backupPath="") and restart fails, process exits
	// so the service manager restarts with the new binary.
	tmpDir, err := os.MkdirTemp("", "graceful-reliability-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "runner")
	err = os.WriteFile(execPath, []byte("old binary"), 0755)
	require.NoError(t, err)

	mock := &MockReleaseDetector{}
	callCount := 0
	u := New("1.0.0",
		WithReleaseDetector(mock),
		WithExecPathFunc(func() (string, error) {
			callCount++
			if callCount == 1 {
				return filepath.Join(tmpDir, "nonexistent", "runner"), nil
			}
			return execPath, nil
		}),
	)

	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, errors.New("restart failed")
	}))

	exited := false
	g.exitFunc = func(code int) { exited = true }

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	_ = g.executeUpdate(context.Background())
	assert.True(t, exited)
}

func TestGracefulUpdater_ApplyUpdate_RestartFailed_Exits(t *testing.T) {
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

	exited := false
	g.exitFunc = func(code int) { exited = true }

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	_ = g.executeUpdate(context.Background())
	assert.True(t, exited)
}
