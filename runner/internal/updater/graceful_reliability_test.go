package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for reliability fixes in graceful.go

// TestGracefulUpdater_ApplyPendingUpdate_RestartErrorPropagation tests that restart errors are propagated.
func TestGracefulUpdater_ApplyPendingUpdate_RestartErrorPropagation(t *testing.T) {
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

	restartErr := errors.New("simulated restart failure")
	g := NewGracefulUpdater(u, nil, WithRestartFunc(func() (int, error) {
		return 0, restartErr
	}))

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// Apply should now return the restart error
	err = g.applyPendingUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart failed")
	assert.Contains(t, err.Error(), "simulated restart failure")
	// State should be reset to Idle after failure
	assert.Equal(t, StateIdle, g.State())
}

// TestGracefulUpdater_ApplyPendingUpdate_HealthCheckFailed_Rollback tests automatic rollback on health check failure.
func TestGracefulUpdater_ApplyPendingUpdate_HealthCheckFailed_Rollback(t *testing.T) {
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
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.applyPendingUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
	assert.Equal(t, StateIdle, g.State())

	// Verify rollback was attempted (binary should be restored)
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "old binary", string(content))
}

// TestGracefulUpdater_ApplyPendingUpdate_RestartFailed_Rollback tests automatic rollback on restart failure.
func TestGracefulUpdater_ApplyPendingUpdate_RestartFailed_Rollback(t *testing.T) {
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

	g := NewGracefulUpdater(u, nil,
		WithRestartFunc(func() (int, error) {
			return 0, errors.New("failed to start new process")
		}),
	)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.applyPendingUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restart failed")
	assert.Equal(t, StateIdle, g.State())

	// Verify rollback was attempted (binary should be restored)
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "old binary", string(content))
}

// TestDefaultHealthChecker_ProcessRunning tests the default health checker with a running process.
func TestDefaultHealthChecker_ProcessRunning(t *testing.T) {
	// Use current process PID which is definitely running
	pid := os.Getpid()

	hc := DefaultHealthChecker(10 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := hc(ctx, pid)
	assert.NoError(t, err)
}

// TestDefaultHealthChecker_ProcessNotFound tests the default health checker with an invalid PID.
func TestDefaultHealthChecker_ProcessNotFound(t *testing.T) {
	// Use an invalid PID that's unlikely to exist
	// Note: On Unix, FindProcess always succeeds, but Signal(0) will fail
	pid := 999999999

	hc := DefaultHealthChecker(10 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := hc(ctx, pid)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "process not running")
}

// TestDefaultHealthChecker_ContextTimeout tests the default health checker timeout behavior.
func TestDefaultHealthChecker_ContextTimeout(t *testing.T) {
	pid := os.Getpid()

	hc := DefaultHealthChecker(5 * time.Second) // Long run time
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := hc(ctx, pid)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestSetState_AtomicUpdate tests that setState updates state atomically.
func TestSetState_AtomicUpdate(t *testing.T) {
	u := New("1.0.0")

	var callbackCount int32
	var lastState State
	var lastInfo *UpdateInfo

	cb := func(state State, info *UpdateInfo, activePods int) {
		atomic.AddInt32(&callbackCount, 1)
		lastState = state
		lastInfo = info
	}

	g := NewGracefulUpdater(u, func() int { return 5 }, WithStatusCallback(cb))

	// Set pending info first
	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// Set state - should atomically capture info and call callback
	g.setState(StateDownloading)

	assert.Equal(t, int32(1), atomic.LoadInt32(&callbackCount))
	assert.Equal(t, StateDownloading, lastState)
	assert.NotNil(t, lastInfo)
	assert.Equal(t, "v2.0.0", lastInfo.LatestVersion)
}

// TestSetState_ConcurrentAccess tests that setState is safe for concurrent access.
func TestSetState_ConcurrentAccess(t *testing.T) {
	u := New("1.0.0")

	var callbackCount int32
	cb := func(state State, info *UpdateInfo, activePods int) {
		atomic.AddInt32(&callbackCount, 1)
	}

	g := NewGracefulUpdater(u, func() int { return 0 }, WithStatusCallback(cb))

	// Run multiple goroutines setting state concurrently
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(state State) {
			for j := 0; j < 100; j++ {
				g.setState(state)
			}
			done <- struct{}{}
		}(State(i % 6))
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all callbacks were called (1000 total)
	assert.Equal(t, int32(1000), atomic.LoadInt32(&callbackCount))
}

// TestWithRestartFuncLegacy tests the legacy restart function adapter.
func TestWithRestartFuncLegacy(t *testing.T) {
	u := New("1.0.0")

	called := false
	legacyFunc := func() error {
		called = true
		return nil
	}

	g := NewGracefulUpdater(u, nil, WithRestartFuncLegacy(legacyFunc))
	assert.NotNil(t, g.restartFunc)

	pid, err := g.restartFunc()
	assert.NoError(t, err)
	assert.Equal(t, 0, pid) // Legacy adapter returns 0 for PID
	assert.True(t, called)
}

// TestWithRestartFuncLegacy_Error tests the legacy restart function adapter with error.
func TestWithRestartFuncLegacy_Error(t *testing.T) {
	u := New("1.0.0")

	expectedErr := errors.New("legacy restart failed")
	legacyFunc := func() error {
		return expectedErr
	}

	g := NewGracefulUpdater(u, nil, WithRestartFuncLegacy(legacyFunc))

	pid, err := g.restartFunc()
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 0, pid)
}

// TestWithHealthChecker tests the health checker option.
func TestWithHealthChecker(t *testing.T) {
	u := New("1.0.0")

	customChecker := func(ctx context.Context, pid int) error {
		return nil
	}

	g := NewGracefulUpdater(u, nil, WithHealthChecker(customChecker))
	assert.NotNil(t, g.healthChecker)
}

// TestWithHealthTimeout tests the health timeout option.
func TestWithHealthTimeout(t *testing.T) {
	u := New("1.0.0")

	g := NewGracefulUpdater(u, nil, WithHealthTimeout(5*time.Minute))
	assert.Equal(t, 5*time.Minute, g.healthTimeout)
}

// TestNewGracefulUpdater_DefaultHealthTimeout tests the default health timeout.
func TestNewGracefulUpdater_DefaultHealthTimeout(t *testing.T) {
	u := New("1.0.0")

	g := NewGracefulUpdater(u, nil)
	assert.Equal(t, 30*time.Second, g.healthTimeout)
}

// TestGracefulUpdater_HealthCheckSuccess tests successful health check scenario.
func TestGracefulUpdater_HealthCheckSuccess(t *testing.T) {
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
			return os.Getpid(), nil // Return current process PID
		}),
		WithHealthChecker(func(ctx context.Context, pid int) error {
			healthCheckCalled = true
			return nil // Health check passes
		}),
		WithHealthTimeout(100*time.Millisecond),
	)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.applyPendingUpdate()
	assert.NoError(t, err)
	assert.True(t, healthCheckCalled)
	assert.Equal(t, StateRestarting, g.State())

	// Verify new binary was applied
	content, err := os.ReadFile(execPath)
	require.NoError(t, err)
	assert.Equal(t, "new binary", string(content))
}

// TestGracefulUpdater_NoHealthChecker tests that updates work without a health checker.
func TestGracefulUpdater_NoHealthChecker(t *testing.T) {
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

	restarted := false
	g := NewGracefulUpdater(u, nil,
		WithRestartFunc(func() (int, error) {
			restarted = true
			return 12345, nil
		}),
		// No health checker set
	)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.applyPendingUpdate()
	assert.NoError(t, err)
	assert.True(t, restarted)
	assert.Equal(t, StateRestarting, g.State())
}

// TestGracefulUpdater_HealthCheckSkippedForZeroPID tests that health check is skipped when PID is 0.
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
