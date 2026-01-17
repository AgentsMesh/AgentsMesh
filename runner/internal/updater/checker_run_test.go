package updater

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for BackgroundChecker run and doCheck functionality

func TestBackgroundChecker_CheckNow(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = c.CheckNow(ctx)
	assert.False(t, c.LastCheck().IsZero())
}

func TestBackgroundChecker_DoCheck_WithCallbacks(t *testing.T) {
	u := New("999.0.0")

	var errorCalled bool
	var updateCalled bool

	c := NewBackgroundChecker(u, nil, time.Hour,
		WithOnError(func(err error) {
			errorCalled = true
		}),
		WithOnUpdate(func(info *UpdateInfo) {
			updateCalled = true
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = c.CheckNow(ctx)
	_ = errorCalled
	_ = updateCalled
}

func TestBackgroundChecker_DoCheck_Error(t *testing.T) {
	u := New("1.0.0")

	var errorCalled bool
	c := NewBackgroundChecker(u, nil, time.Hour,
		WithOnError(func(err error) {
			errorCalled = true
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	info, err := c.CheckNow(ctx)
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.True(t, errorCalled)
	assert.NotNil(t, c.LastError())
}

func TestBackgroundChecker_DoCheck_NoUpdate(t *testing.T) {
	u := New("999.999.999")
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.CheckNow(ctx)
	if err == nil {
		assert.NotNil(t, info)
		assert.False(t, info.HasUpdate)
	}
}

func TestBackgroundChecker_Run_ContextCanceled(t *testing.T) {
	u := New("1.0.0")
	c := NewBackgroundChecker(u, nil, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())

	c.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	assert.True(t, c.IsRunning())

	cancel()
	time.Sleep(200 * time.Millisecond)

	c.Stop()
	assert.False(t, c.IsRunning())
}
