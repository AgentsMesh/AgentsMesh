package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for health checker functionality

func TestDefaultHealthChecker_ProcessRunning(t *testing.T) {
	// Use current process PID which is definitely running
	pid := os.Getpid()

	hc := DefaultHealthChecker(10 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := hc(ctx, pid)
	assert.NoError(t, err)
}

func TestDefaultHealthChecker_ProcessNotFound(t *testing.T) {
	// Use an invalid PID that's unlikely to exist
	// Note: On Unix, FindProcess always succeeds, but Signal(0) will fail
	pid := 999999999

	hc := DefaultHealthChecker(10 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := hc(ctx, pid)
	assert.Error(t, err)
	// Unix: "process not running: ...", Windows: "process not found (pid ...): ..."
	errMsg := err.Error()
	assert.True(t, strings.Contains(errMsg, "process not running") || strings.Contains(errMsg, "process not found"),
		"expected error to contain 'process not running' or 'process not found', got: %s", errMsg)
}

func TestDefaultHealthChecker_ContextTimeout(t *testing.T) {
	pid := os.Getpid()

	hc := DefaultHealthChecker(5 * time.Second) // Long run time
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := hc(ctx, pid)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWithHealthChecker(t *testing.T) {
	u := New("1.0.0")

	customChecker := func(ctx context.Context, pid int) error {
		return nil
	}

	g := NewGracefulUpdater(u, nil, WithHealthChecker(customChecker))
	assert.NotNil(t, g.healthChecker)
}

func TestWithHealthTimeout(t *testing.T) {
	u := New("1.0.0")

	g := NewGracefulUpdater(u, nil, WithHealthTimeout(5*time.Minute))
	assert.Equal(t, 5*time.Minute, g.healthTimeout)
}

func TestNewGracefulUpdater_DefaultHealthTimeout(t *testing.T) {
	u := New("1.0.0")

	g := NewGracefulUpdater(u, nil)
	assert.Equal(t, 30*time.Second, g.healthTimeout)
}

func TestGracefulUpdater_HealthCheckSkippedForZeroPID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "graceful-reliability-*")
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

	healthCheckCalled := false
	g := NewGracefulUpdater(u, nil,
		WithRestartFunc(func() (int, error) {
			return 0, nil // Return PID 0
		}),
		WithHealthChecker(func(ctx context.Context, pid int) error {
			healthCheckCalled = true
			return errors.New("should not be called")
		}),
	)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.applyPendingUpdate()
	assert.NoError(t, err)
	assert.False(t, healthCheckCalled) // Health check should be skipped for PID 0
}
