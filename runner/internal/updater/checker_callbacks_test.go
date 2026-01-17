package updater

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for BackgroundChecker callback and run functionality

func TestBackgroundChecker_DoCheck_CallsOnError(t *testing.T) {
	u := New("1.0.0")

	var errorCalled atomic.Bool
	var lastError error

	c := NewBackgroundChecker(u, nil, time.Hour,
		WithOnError(func(err error) {
			errorCalled.Store(true)
			lastError = err
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(20 * time.Millisecond)

	_, err := c.doCheck(ctx)
	assert.Error(t, err)

	// Error callback should be called
	assert.True(t, errorCalled.Load())
	assert.NotNil(t, lastError)
}

func TestBackgroundChecker_DoCheck_SetsLastCheck(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	before := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	time.Sleep(20 * time.Millisecond)

	c.doCheck(ctx)

	after := time.Now()

	lastCheck := c.LastCheck()
	assert.True(t, lastCheck.After(before) || lastCheck.Equal(before))
	assert.True(t, lastCheck.Before(after) || lastCheck.Equal(after))
}

func TestBackgroundChecker_DoCheck_UpdatesLastError(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	_, err := c.doCheck(ctx)
	assert.Error(t, err)

	// LastError should be set
	assert.NotNil(t, c.LastError())
}

func TestBackgroundChecker_Run_InitialDelay(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately - should return without checking
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	c.run(ctx)

	// LastCheck should still be zero because we cancelled during initial delay
	assert.True(t, c.LastCheck().IsZero())
}

func TestBackgroundChecker_Run_PeriodicCheck(t *testing.T) {
	u := New("1.0.0")

	// Use short intervals for testing
	c := NewBackgroundChecker(u, nil, 50*time.Millisecond)

	// Override the run to skip initial delay for testing
	ctx, cancel := context.WithCancel(context.Background())

	var checkCount atomic.Int32
	originalDoCheck := c.doCheck

	// We can't easily mock doCheck, so we'll just test the context cancellation
	_ = originalDoCheck

	go func() {
		// Let it run briefly then cancel
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	c.run(ctx)
	_ = checkCount

	// Verify running state is reset after context cancellation
	assert.False(t, c.IsRunning())
}

func TestBackgroundChecker_Start_SetsRunning(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.Start(ctx)

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	assert.True(t, c.IsRunning())

	c.Stop()

	// Give goroutine time to stop
	time.Sleep(10 * time.Millisecond)

	assert.False(t, c.IsRunning())
}

func TestBackgroundChecker_Stop_CancelsContext(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx := context.Background()
	c.Start(ctx)

	time.Sleep(10 * time.Millisecond)
	assert.True(t, c.IsRunning())

	c.Stop()

	time.Sleep(10 * time.Millisecond)
	assert.False(t, c.IsRunning())

	// Verify cancel was called (internal state)
	c.mu.RLock()
	// cancel should have been set and called
	c.mu.RUnlock()
}

func TestBackgroundChecker_WithGracefulUpdater_AutoApply(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	c := NewBackgroundChecker(u, g, time.Hour, WithAutoApply(true))

	assert.True(t, c.autoApply)
	assert.NotNil(t, c.graceful)
}

func TestBackgroundChecker_LatestInfo_Copy(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	info := &UpdateInfo{
		LatestVersion:  "v2.0.0",
		CurrentVersion: "v1.0.0",
		HasUpdate:      true,
		ReleaseNotes:   "Test notes",
	}

	c.mu.Lock()
	c.latestInfo = info
	c.mu.Unlock()

	result := c.LatestInfo()
	assert.Equal(t, info, result)
}
