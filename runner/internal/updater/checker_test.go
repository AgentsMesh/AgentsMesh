package updater

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackgroundChecker_NewAndDefaults(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	assert.NotNil(t, c)
	assert.Equal(t, time.Hour, c.interval)
	assert.True(t, c.autoApply)
	assert.False(t, c.IsRunning())
}

func TestBackgroundChecker_Options(t *testing.T) {
	u := New("1.0.0")

	t.Run("with auto apply disabled", func(t *testing.T) {
		c := NewBackgroundChecker(u, nil, time.Hour, WithAutoApply(false))
		assert.False(t, c.autoApply)
	})

	t.Run("with on update callback", func(t *testing.T) {
		c := NewBackgroundChecker(u, nil, time.Hour, WithOnUpdate(func(info *UpdateInfo) {}))
		assert.NotNil(t, c.onUpdate)
	})

	t.Run("with on error callback", func(t *testing.T) {
		c := NewBackgroundChecker(u, nil, time.Hour, WithOnError(func(err error) {}))
		assert.NotNil(t, c.onError)
	})
}

func TestBackgroundChecker_StartStop(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	assert.True(t, c.IsRunning())

	c.Start(ctx) // no-op
	assert.True(t, c.IsRunning())

	c.Stop()
	time.Sleep(50 * time.Millisecond)
	assert.False(t, c.IsRunning())

	c.Stop() // no-op
	assert.False(t, c.IsRunning())
}

func TestBackgroundChecker_LastCheck(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	assert.True(t, c.LastCheck().IsZero())
}

func TestBackgroundChecker_UpdateAvailable(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	assert.False(t, c.UpdateAvailable())

	c.mu.Lock()
	c.latestInfo = &UpdateInfo{HasUpdate: true, LatestVersion: "v2.0.0"}
	c.mu.Unlock()

	assert.True(t, c.UpdateAvailable())
}

func TestBackgroundChecker_NextCheckIn(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	assert.Equal(t, time.Duration(0), c.NextCheckIn())

	c.mu.Lock()
	c.lastCheck = time.Now()
	c.mu.Unlock()

	nextCheck := c.NextCheckIn()
	assert.True(t, nextCheck > 59*time.Minute)
	assert.True(t, nextCheck <= time.Hour)
}

func TestBackgroundChecker_NextCheckIn_Past(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	c.mu.Lock()
	c.lastCheck = time.Now().Add(-2 * time.Hour)
	c.mu.Unlock()

	assert.Equal(t, time.Duration(0), c.NextCheckIn())
}

func TestBackgroundChecker_LastError(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	assert.Nil(t, c.LastError())

	c.mu.Lock()
	c.lastError = context.DeadlineExceeded
	c.mu.Unlock()

	assert.Equal(t, context.DeadlineExceeded, c.LastError())
}

func TestBackgroundChecker_LatestInfo(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	assert.Nil(t, c.LatestInfo())

	info := &UpdateInfo{LatestVersion: "v2.0.0", HasUpdate: true}
	c.mu.Lock()
	c.latestInfo = info
	c.mu.Unlock()

	result := c.LatestInfo()
	assert.NotNil(t, result)
	assert.Equal(t, "v2.0.0", result.LatestVersion)
}

func TestBackgroundChecker_WithGracefulUpdater(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	c := NewBackgroundChecker(u, g, time.Hour)
	assert.NotNil(t, c.graceful)
}

func TestBackgroundChecker_UpdateAvailable_NoInfo(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	assert.False(t, c.UpdateAvailable())
}

func TestBackgroundChecker_UpdateAvailable_NoUpdate(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	c.mu.Lock()
	c.latestInfo = &UpdateInfo{HasUpdate: false}
	c.mu.Unlock()

	assert.False(t, c.UpdateAvailable())
}
