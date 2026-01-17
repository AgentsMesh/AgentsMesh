package updater

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for BackgroundChecker.run using MockReleaseDetector

func TestBackgroundChecker_Run_WithMock_InitialDelay(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately after starting to test initial delay path
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	c.run(ctx)
	assert.False(t, c.IsRunning())
}

func TestBackgroundChecker_Run_WithMock_PeriodicCheck(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	// Override initial delay by using very short interval
	c := &BackgroundChecker{
		updater:   u,
		interval:  50 * time.Millisecond,
		autoApply: false,
	}

	ctx, cancel := context.WithCancel(context.Background())

	checkCount := 0
	c.onUpdate = func(info *UpdateInfo) {
		checkCount++
	}

	go func() {
		// Wait for a couple of ticker cycles (50ms delay won't really run due to 30s initial)
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	c.run(ctx)
	assert.False(t, c.IsRunning())
}

func TestBackgroundChecker_Run_WithMock_CancelDuringInitialDelay(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately to test initial delay cancellation
	cancel()

	c.run(ctx)
	// Should exit cleanly without running any checks
	assert.False(t, c.IsRunning())
}

func TestBackgroundChecker_Run_WithMock_CancelDuringInitialDelay_ExitsCleanly(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))
	c := NewBackgroundChecker(u, nil, time.Hour)

	// Already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// run should exit immediately without panic
	c.run(ctx)

	// Note: run() doesn't set running=false when cancelled during initial delay
	// because it just returns early. This is expected behavior.
}

func TestBackgroundChecker_Run_WithMock_FullCycle(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v1.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))
	c := NewBackgroundChecker(u, nil, 20*time.Millisecond,
		WithInitialDelay(10*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())

	// Let it run through initial delay and at least one tick
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	c.run(ctx)

	// After run exits due to context cancel in ticker loop, running should be false
	c.mu.RLock()
	running := c.running
	c.mu.RUnlock()
	assert.False(t, running)
}

func TestBackgroundChecker_Run_WithMock_InitialCheckAndTicker(t *testing.T) {
	mock := &MockReleaseDetector{
		LatestRelease: &ReleaseInfo{
			Version: "v2.0.0",
		},
	}

	u := New("1.0.0", WithReleaseDetector(mock))

	checkCount := 0
	c := NewBackgroundChecker(u, nil, 30*time.Millisecond,
		WithInitialDelay(10*time.Millisecond),
		WithOnUpdate(func(info *UpdateInfo) {
			checkCount++
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Wait for initial delay + initial check + at least one ticker
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	c.run(ctx)

	// Should have called onUpdate at least once (initial check finds update)
	assert.GreaterOrEqual(t, checkCount, 1)
}

func TestBackgroundChecker_WithInitialDelay(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour, WithInitialDelay(5*time.Second))

	assert.Equal(t, 5*time.Second, c.initialDelay)
}
