package runner

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shutdownMockPodIO implements PodIO with Teardown/Detach tracking for shutdown tests.
type shutdownMockPodIO struct {
	teardownCalled bool
	detachCalled   bool
}

func (m *shutdownMockPodIO) Mode() string                              { return "pty" }
func (m *shutdownMockPodIO) SendInput(string) error                    { return nil }
func (m *shutdownMockPodIO) GetSnapshot(int) (string, error)           { return "", nil }
func (m *shutdownMockPodIO) GetAgentStatus() string                    { return "idle" }
func (m *shutdownMockPodIO) SubscribeStateChange(string, func(string)) {}
func (m *shutdownMockPodIO) UnsubscribeStateChange(string)             {}
func (m *shutdownMockPodIO) GetPID() int                               { return 0 }
func (m *shutdownMockPodIO) Stop()                                     {}
func (m *shutdownMockPodIO) Teardown() string                          { m.teardownCalled = true; return "" }
func (m *shutdownMockPodIO) SetExitHandler(func(int))                  {}
func (m *shutdownMockPodIO) Detach()                                   { m.detachCalled = true }
func (m *shutdownMockPodIO) Start() error                              { return nil }
func (m *shutdownMockPodIO) SetIOErrorHandler(func(error))             {}

// --- CanAcceptPod tests ---

func TestCanAcceptPod_NotDraining_BelowLimit(t *testing.T) {
	r, _ := NewTestRunner(t, WithTestConfig(&config.Config{
		WorkspaceRoot:     t.TempDir(),
		MaxConcurrentPods: 5,
		NodeID:            "test",
	}))

	assert.True(t, r.CanAcceptPod(), "should accept pod when not draining and below limit")
}

func TestCanAcceptPod_Draining(t *testing.T) {
	r, _ := NewTestRunner(t, WithTestConfig(&config.Config{
		WorkspaceRoot:     t.TempDir(),
		MaxConcurrentPods: 5,
		NodeID:            "test",
	}))

	r.SetDraining(true)
	assert.False(t, r.CanAcceptPod(), "should reject pod when draining")
}

func TestCanAcceptPod_AtMaxCapacity(t *testing.T) {
	r, _ := NewTestRunner(t, WithTestConfig(&config.Config{
		WorkspaceRoot:     t.TempDir(),
		MaxConcurrentPods: 2,
		NodeID:            "test",
	}))

	r.podStore.Put("pod-1", &Pod{PodKey: "pod-1"})
	r.podStore.Put("pod-2", &Pod{PodKey: "pod-2"})

	assert.False(t, r.CanAcceptPod(), "should reject pod when at max capacity")
}

// --- GetActivePodCount tests ---

func TestGetActivePodCount(t *testing.T) {
	r, _ := NewTestRunner(t)

	assert.Equal(t, 0, r.GetActivePodCount(), "empty store should return 0")

	r.podStore.Put("pod-a", &Pod{PodKey: "pod-a"})
	assert.Equal(t, 1, r.GetActivePodCount())

	r.podStore.Put("pod-b", &Pod{PodKey: "pod-b"})
	r.podStore.Put("pod-c", &Pod{PodKey: "pod-c"})
	assert.Equal(t, 3, r.GetActivePodCount())
}

// --- stopAllPods tests ---

func TestStopAllPods_Empty(t *testing.T) {
	r, _ := NewTestRunner(t)

	require.Equal(t, 0, r.podStore.Count())
	// Must not panic or hang.
	r.stopAllPods()
}

func TestStopAllPods_WithPods(t *testing.T) {
	r, _ := NewTestRunner(t)

	io1 := &shutdownMockPodIO{}
	io2 := &shutdownMockPodIO{}

	r.podStore.Put("pod-1", &Pod{PodKey: "pod-1", IO: io1, StartedAt: time.Now()})
	r.podStore.Put("pod-2", &Pod{PodKey: "pod-2", IO: io2, StartedAt: time.Now()})
	require.Equal(t, 2, r.podStore.Count())

	r.stopAllPods()

	assert.Equal(t, 0, r.podStore.Count(), "all pods should be removed from store")
	assert.True(t, io1.teardownCalled, "pod-1 IO.Teardown should be called")
	assert.True(t, io1.detachCalled, "pod-1 IO.Detach should be called")
	assert.True(t, io2.teardownCalled, "pod-2 IO.Teardown should be called")
	assert.True(t, io2.detachCalled, "pod-2 IO.Detach should be called")
}

func TestStopAllPods_NilIO(t *testing.T) {
	r, _ := NewTestRunner(t)

	// Pod with nil IO — should not panic.
	r.podStore.Put("pod-nil", &Pod{PodKey: "pod-nil", StartedAt: time.Now()})

	r.stopAllPods()

	assert.Equal(t, 0, r.podStore.Count(), "pod with nil IO should be cleaned up")
}
