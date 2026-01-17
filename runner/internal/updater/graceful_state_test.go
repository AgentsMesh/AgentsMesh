package updater

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for GracefulUpdater state and options

func TestGracefulUpdater_State(t *testing.T) {
	u := New("1.0.0")
	podCounter := func() int { return 0 }
	g := NewGracefulUpdater(u, podCounter)

	assert.Equal(t, StateIdle, g.State())
	assert.False(t, g.IsDraining())
	assert.Empty(t, g.PendingVersion())
}

func TestGracefulUpdater_Options(t *testing.T) {
	u := New("1.0.0")
	podCounter := func() int { return 0 }

	t.Run("with max wait time", func(t *testing.T) {
		g := NewGracefulUpdater(u, podCounter, WithMaxWaitTime(5*time.Minute))
		assert.Equal(t, 5*time.Minute, g.maxWaitTime)
	})

	t.Run("with poll interval", func(t *testing.T) {
		g := NewGracefulUpdater(u, podCounter, WithPollInterval(10*time.Second))
		assert.Equal(t, 10*time.Second, g.pollInterval)
	})

	t.Run("with status callback", func(t *testing.T) {
		var called atomic.Bool
		cb := func(state State, info *UpdateInfo, activePods int) {
			called.Store(true)
		}
		g := NewGracefulUpdater(u, podCounter, WithStatusCallback(cb))
		assert.NotNil(t, g.onStatus)
	})
}

func TestGracefulUpdater_CancelPendingUpdate(t *testing.T) {
	u := New("1.0.0")
	podCounter := func() int { return 0 }
	g := NewGracefulUpdater(u, podCounter)

	g.mu.Lock()
	g.draining = true
	g.pendingPath = "/tmp/test-update"
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", HasUpdate: true}
	g.mu.Unlock()

	g.CancelPendingUpdate()

	assert.False(t, g.IsDraining())
	assert.Equal(t, StateIdle, g.State())
	assert.Empty(t, g.PendingVersion())
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateIdle, "idle"},
		{StateChecking, "checking"},
		{StateDownloading, "downloading"},
		{StateDraining, "draining"},
		{StateApplying, "applying"},
		{StateRestarting, "restarting"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestGracefulUpdater_SetState(t *testing.T) {
	u := New("1.0.0")
	podCounter := func() int { return 5 }

	var callbackState State
	var callbackInfo *UpdateInfo
	var callbackPods int
	var callbackCalled bool

	cb := func(state State, info *UpdateInfo, activePods int) {
		callbackState = state
		callbackInfo = info
		callbackPods = activePods
		callbackCalled = true
	}

	g := NewGracefulUpdater(u, podCounter, WithStatusCallback(cb))

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	g.setState(StateDownloading)

	assert.True(t, callbackCalled)
	assert.Equal(t, StateDownloading, callbackState)
	assert.Equal(t, "v2.0.0", callbackInfo.LatestVersion)
	assert.Equal(t, 5, callbackPods)
}

func TestGracefulUpdater_SetState_NilPodCounter(t *testing.T) {
	u := New("1.0.0")

	var callbackPods int
	cb := func(state State, info *UpdateInfo, activePods int) {
		callbackPods = activePods
	}

	g := NewGracefulUpdater(u, nil, WithStatusCallback(cb))
	g.setState(StateChecking)

	assert.Equal(t, 0, callbackPods)
}

func TestGracefulUpdater_SetState_NoCallback(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	g.setState(StateChecking)
	assert.Equal(t, StateChecking, g.State())
}

func TestGracefulUpdater_Defaults(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	assert.Equal(t, 30*time.Minute, g.maxWaitTime)
	assert.Equal(t, 5*time.Second, g.pollInterval)
	assert.Equal(t, StateIdle, g.state)
}

func TestGracefulUpdater_PendingVersion_WithInfo(t *testing.T) {
	u := New("1.0.0")
	g := NewGracefulUpdater(u, nil)

	g.mu.Lock()
	g.pendingInfo = &UpdateInfo{LatestVersion: "v3.0.0"}
	g.mu.Unlock()

	assert.Equal(t, "v3.0.0", g.PendingVersion())
}
