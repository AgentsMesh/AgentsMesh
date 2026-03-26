package autopilot

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestStateDetectorCoordinator_TriggersOnWaiting(t *testing.T) {
	var waitingCount atomic.Int32
	podCtrl := NewMockPodControllerWithStateChange()

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		PodCtrl:      podCtrl,
		OnWaiting:    func() { waitingCount.Add(1) },
		AutopilotKey: "test-1",
	})
	sdc.Start()
	defer sdc.Stop()

	// Simulate executing → waiting transition
	podCtrl.SimulateStateChange("executing")
	podCtrl.SimulateStateChange("waiting")

	time.Sleep(50 * time.Millisecond)
	if got := waitingCount.Load(); got != 1 {
		t.Errorf("OnWaiting called %d times, want 1", got)
	}
}

func TestStateDetectorCoordinator_TriggersOnIdle(t *testing.T) {
	// ACP mode uses "idle" instead of "waiting"
	var waitingCount atomic.Int32
	podCtrl := NewMockPodControllerWithStateChange()

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		PodCtrl:      podCtrl,
		OnWaiting:    func() { waitingCount.Add(1) },
		AutopilotKey: "test-2",
	})
	sdc.Start()
	defer sdc.Stop()

	podCtrl.SimulateStateChange("executing")
	podCtrl.SimulateStateChange("idle")

	time.Sleep(50 * time.Millisecond)
	if got := waitingCount.Load(); got != 1 {
		t.Errorf("OnWaiting called %d times, want 1", got)
	}
}

func TestStateDetectorCoordinator_IgnoresNonExecutingToWaiting(t *testing.T) {
	var waitingCount atomic.Int32
	podCtrl := NewMockPodControllerWithStateChange()

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		PodCtrl:      podCtrl,
		OnWaiting:    func() { waitingCount.Add(1) },
		AutopilotKey: "test-3",
	})
	sdc.Start()
	defer sdc.Stop()

	// idle → waiting should NOT trigger (only executing → waiting)
	podCtrl.SimulateStateChange("idle")
	podCtrl.SimulateStateChange("waiting")

	time.Sleep(50 * time.Millisecond)
	if got := waitingCount.Load(); got != 0 {
		t.Errorf("OnWaiting called %d times, want 0 (idle→waiting should not trigger)", got)
	}
}

func TestStateDetectorCoordinator_MultipleTransitions(t *testing.T) {
	var waitingCount atomic.Int32
	podCtrl := NewMockPodControllerWithStateChange()

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		PodCtrl:      podCtrl,
		OnWaiting:    func() { waitingCount.Add(1) },
		AutopilotKey: "test-4",
	})
	sdc.Start()
	defer sdc.Stop()

	// Two full cycles
	podCtrl.SimulateStateChange("executing")
	podCtrl.SimulateStateChange("waiting")
	podCtrl.SimulateStateChange("executing")
	podCtrl.SimulateStateChange("idle")

	time.Sleep(50 * time.Millisecond)
	if got := waitingCount.Load(); got != 2 {
		t.Errorf("OnWaiting called %d times, want 2", got)
	}
}

func TestStateDetectorCoordinator_NilPodCtrl(t *testing.T) {
	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		PodCtrl:      nil,
		OnWaiting:    func() {},
		AutopilotKey: "test-nil",
	})
	// Should not panic
	sdc.Start()
	sdc.Stop()
}

func TestStateDetectorCoordinator_StopUnsubscribes(t *testing.T) {
	podCtrl := NewMockPodControllerWithStateChange()
	var waitingCount atomic.Int32

	sdc := NewStateDetectorCoordinator(StateDetectorCoordinatorConfig{
		PodCtrl:      podCtrl,
		OnWaiting:    func() { waitingCount.Add(1) },
		AutopilotKey: "test-unsub",
	})
	sdc.Start()
	sdc.Stop()

	// After stop, state changes should not trigger
	podCtrl.SimulateStateChange("executing")
	podCtrl.SimulateStateChange("waiting")

	time.Sleep(50 * time.Millisecond)
	if got := waitingCount.Load(); got != 0 {
		t.Errorf("OnWaiting called %d times after Stop, want 0", got)
	}
}
