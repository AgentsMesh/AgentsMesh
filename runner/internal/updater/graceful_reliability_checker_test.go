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

// Tests for health checker with successful scenarios

func TestGracefulUpdater_HealthCheckSuccess(t *testing.T) {
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

	healthCheckCalled := false
	g := NewGracefulUpdater(u, nil,
		WithRestartFunc(func() (int, error) {
			return os.Getpid(), nil // Return current process PID
		}),
		WithHealthChecker(func(ctx context.Context, pid int) error {
			healthCheckCalled = true
			return nil // Health check passes
		}),
		WithHealthTimeout(100*time.Millisecond),
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.NoError(t, err)
	assert.True(t, healthCheckCalled)
	assert.Equal(t, StateRestarting, g.State())

	// Verify new binary was applied
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "mock binary", string(content))
}

func TestGracefulUpdater_NoHealthChecker(t *testing.T) {
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

	restarted := false
	g := NewGracefulUpdater(u, nil,
		WithRestartFunc(func() (int, error) {
			restarted = true
			return 12345, nil
		}),
		// No health checker set
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.executeUpdate(context.Background())
	assert.NoError(t, err)
	assert.True(t, restarted)
	assert.Equal(t, StateRestarting, g.State())
}
