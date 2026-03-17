package updater

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for drainPods functionality

func TestGracefulUpdater_DrainPods_ContextCanceled(t *testing.T) {
	u := New("1.0.0")

	var podCount int32 = 5
	podCounter := func() int { return int(atomic.LoadInt32(&podCount)) }

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(10*time.Second),
		WithPollInterval(50*time.Millisecond),
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- g.drainPods(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	err := <-errCh
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	assert.False(t, g.IsDraining())
}

func TestGracefulUpdater_DrainPods_Timeout(t *testing.T) {
	u := New("1.0.0")
	podCounter := func() int { return 1 }

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(100*time.Millisecond),
		WithPollInterval(20*time.Millisecond),
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	err := g.drainPods(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postponed")
	assert.Equal(t, StateIdle, g.State())

	g.mu.RLock()
	assert.Nil(t, g.pendingInfo)
	g.mu.RUnlock()
}

func TestGracefulUpdater_DrainPods_PodsFinish(t *testing.T) {
	u := New("1.0.0")

	var podCount int32 = 2
	podCounter := func() int { return int(atomic.LoadInt32(&podCount)) }

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(5*time.Second),
		WithPollInterval(50*time.Millisecond),
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	go func() {
		time.Sleep(100 * time.Millisecond)
		atomic.StoreInt32(&podCount, 1)
		time.Sleep(100 * time.Millisecond)
		atomic.StoreInt32(&podCount, 0)
	}()

	err := g.drainPods(context.Background())
	assert.NoError(t, err)
}

func TestGracefulUpdater_DrainPods_StatusCallback(t *testing.T) {
	u := New("1.0.0")

	var podCount int32 = 1
	podCounter := func() int { return int(atomic.LoadInt32(&podCount)) }

	var mu sync.Mutex
	var states []State
	cb := func(state State, info *UpdateInfo, activePods int) {
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
	}

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(500*time.Millisecond),
		WithPollInterval(50*time.Millisecond),
		WithStatusCallback(cb),
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	// Let it timeout so we get drain callbacks
	_ = g.drainPods(context.Background())

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, states, StateDraining)
}

func TestGracefulUpdater_DrainPods_NilPodCounter(t *testing.T) {
	u := New("1.0.0")

	g := NewGracefulUpdater(u, nil,
		WithMaxWaitTime(100*time.Millisecond),
		WithPollInterval(20*time.Millisecond),
	)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	// With nil pod counter, pods are always 0, so drain should succeed immediately
	err := g.drainPods(context.Background())
	assert.NoError(t, err)
}
