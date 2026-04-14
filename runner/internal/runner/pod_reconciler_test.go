package runner

import (
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// reconcilerMockPodIO implements PodIO with a configurable PID for reconciler tests.
type reconcilerMockPodIO struct {
	pid int
}

func (m *reconcilerMockPodIO) Mode() string                              { return "pty" }
func (m *reconcilerMockPodIO) SendInput(string) error                    { return nil }
func (m *reconcilerMockPodIO) GetSnapshot(int) (string, error)           { return "", nil }
func (m *reconcilerMockPodIO) GetAgentStatus() string                    { return "idle" }
func (m *reconcilerMockPodIO) SubscribeStateChange(string, func(string)) {}
func (m *reconcilerMockPodIO) UnsubscribeStateChange(string)             {}
func (m *reconcilerMockPodIO) GetPID() int                               { return m.pid }
func (m *reconcilerMockPodIO) Stop()                                     {}
func (m *reconcilerMockPodIO) Teardown() string                          { return "" }
func (m *reconcilerMockPodIO) SetExitHandler(func(int))                  {}
func (m *reconcilerMockPodIO) SetIOErrorHandler(func(error))             {}
func (m *reconcilerMockPodIO) Detach()                                   {}
func (m *reconcilerMockPodIO) Start() error                              { return nil }

func TestPodReconciler_CleansUpDeadProcess(t *testing.T) {
	store := NewInMemoryPodStore()
	pod := &Pod{
		PodKey: "dead-pod",
		Status: PodStatusRunning,
		IO:     &reconcilerMockPodIO{pid: 999999},
	}
	store.Put(pod.PodKey, pod)

	var cleaned atomic.Bool
	reconciler := NewPodReconciler(store, func(podKey string, exitCode int, stopIO bool) {
		if podKey == "dead-pod" && exitCode == -1 && stopIO {
			cleaned.Store(true)
		}
	}, 50*time.Millisecond)

	reconciler.reconcile()

	if !cleaned.Load() {
		t.Error("expected dead pod to be cleaned up")
	}
}

func TestPodReconciler_SkipsLiveProcess(t *testing.T) {
	store := NewInMemoryPodStore()
	pod := &Pod{
		PodKey: "live-pod",
		Status: PodStatusRunning,
		IO:     &reconcilerMockPodIO{pid: os.Getpid()},
	}
	store.Put(pod.PodKey, pod)

	var cleaned atomic.Bool
	reconciler := NewPodReconciler(store, func(_ string, _ int, _ bool) {
		cleaned.Store(true)
	}, 50*time.Millisecond)

	reconciler.reconcile()

	if cleaned.Load() {
		t.Error("live pod should not be cleaned up")
	}
}

func TestPodReconciler_SkipsNonRunningStatus(t *testing.T) {
	store := NewInMemoryPodStore()
	for _, status := range []string{PodStatusInitializing, PodStatusStopped, PodStatusFailed} {
		pod := &Pod{
			PodKey: "pod-" + status,
			Status: status,
			IO:     &reconcilerMockPodIO{pid: 999999},
		}
		store.Put(pod.PodKey, pod)
	}

	var cleaned atomic.Bool
	reconciler := NewPodReconciler(store, func(_ string, _ int, _ bool) {
		cleaned.Store(true)
	}, 50*time.Millisecond)

	reconciler.reconcile()

	if cleaned.Load() {
		t.Error("non-running pods should not be cleaned up")
	}
}

func TestPodReconciler_SkipsZeroPID(t *testing.T) {
	store := NewInMemoryPodStore()
	pod := &Pod{
		PodKey: "acp-pod",
		Status: PodStatusRunning,
		IO:     &reconcilerMockPodIO{pid: 0},
	}
	store.Put(pod.PodKey, pod)

	var cleaned atomic.Bool
	reconciler := NewPodReconciler(store, func(_ string, _ int, _ bool) {
		cleaned.Store(true)
	}, 50*time.Millisecond)

	reconciler.reconcile()

	if cleaned.Load() {
		t.Error("zero-PID pod should not be cleaned up")
	}
}

func TestPodReconciler_SkipsNilIO(t *testing.T) {
	store := NewInMemoryPodStore()
	pod := &Pod{
		PodKey: "no-io-pod",
		Status: PodStatusRunning,
		IO:     nil,
	}
	store.Put(pod.PodKey, pod)

	var cleaned atomic.Bool
	reconciler := NewPodReconciler(store, func(_ string, _ int, _ bool) {
		cleaned.Store(true)
	}, 50*time.Millisecond)

	reconciler.reconcile()

	if cleaned.Load() {
		t.Error("nil-IO pod should not be cleaned up")
	}
}

func TestPodReconciler_IdempotentCleanup(t *testing.T) {
	store := NewInMemoryPodStore()
	pod := &Pod{
		PodKey: "dead-pod",
		Status: PodStatusRunning,
		IO:     &reconcilerMockPodIO{pid: 999999},
	}
	store.Put(pod.PodKey, pod)

	var cleanupCount atomic.Int32
	reconciler := NewPodReconciler(store, func(podKey string, _ int, _ bool) {
		cleanupCount.Add(1)
		store.Delete(podKey)
	}, 50*time.Millisecond)

	reconciler.reconcile()
	reconciler.reconcile()

	if got := cleanupCount.Load(); got != 1 {
		t.Errorf("cleanup called %d times, want 1", got)
	}
}
