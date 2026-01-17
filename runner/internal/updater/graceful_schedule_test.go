package updater

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for GracefulUpdater ScheduleUpdate and related functionality

func TestGracefulUpdater_ScheduleUpdate_StateTransitions(t *testing.T) {
	u := New("1.0.0")
	var states []State
	cb := func(state State, info *UpdateInfo, activePods int) {
		states = append(states, state)
	}

	g := NewGracefulUpdater(u, nil, WithStatusCallback(cb))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will fail due to network, but should transition through states
	err := g.ScheduleUpdate(ctx)
	assert.Error(t, err) // Expected to fail

	// Verify StateChecking was reached
	assert.Contains(t, states, StateChecking)
}

func TestGracefulUpdater_ScheduleUpdate_ConcurrentCalls(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	// Set state to checking to simulate in-progress update
	g.mu.Lock()
	g.state = StateChecking
	g.mu.Unlock()

	// Second call should fail
	err := g.ScheduleUpdate(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update already in progress")

	// Try with other states
	for _, state := range []State{StateDownloading, StateDraining, StateApplying} {
		g.mu.Lock()
		g.state = state
		g.mu.Unlock()

		err := g.ScheduleUpdate(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update already in progress")
	}
}

func TestGracefulUpdater_ForceUpdate_FromIdle(t *testing.T) {
	u := New("1.0.0")

	var states []State
	cb := func(state State, info *UpdateInfo, activePods int) {
		states = append(states, state)
	}

	g := NewGracefulUpdater(u, nil, WithStatusCallback(cb))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Force update from idle state
	err := g.ForceUpdate(ctx)
	assert.Error(t, err) // Expected to fail due to network

	// Verify states
	assert.Contains(t, states, StateChecking)
}

func TestGracefulUpdater_ForceUpdate_InvalidStates(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	invalidStates := []State{StateChecking, StateApplying, StateRestarting}
	for _, state := range invalidStates {
		g.mu.Lock()
		g.state = state
		g.mu.Unlock()

		err := g.ForceUpdate(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot force update in state")
	}
}

func TestGracefulUpdater_DrainContext_Cleanup(t *testing.T) {
	u := New("1.0.0")
	podCount := int32(5)
	podCounter := func() int { return int(atomic.LoadInt32(&podCount)) }

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(100*time.Millisecond),
		WithPollInterval(10*time.Millisecond),
	)

	// Start draining with pods
	g.setState(StateDraining)
	g.mu.Lock()
	g.draining = true
	g.pendingPath = "/tmp/test"
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	// Simulate pods finishing after some time
	go func() {
		time.Sleep(50 * time.Millisecond)
		atomic.StoreInt32(&podCount, 0)
	}()

	// Wait should complete when pods finish
	ctx := context.Background()
	err := g.waitAndApply(ctx)
	// Will fail at apply stage, but drain should complete
	assert.Error(t, err)
}
