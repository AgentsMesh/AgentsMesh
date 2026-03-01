package monitor

import (
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/terminal/detector"
)

// Tests for pod registration and status retrieval

func TestMonitorRegisterPod(t *testing.T) {
	monitor := NewMonitor(time.Second)

	monitor.RegisterPod("pod-1", 12345)

	status, ok := monitor.GetStatus("pod-1")
	if !ok {
		t.Fatal("pod should be registered")
	}

	if status.PodID != "pod-1" {
		t.Errorf("PodID: got %v, want pod-1", status.PodID)
	}

	if status.Pid != 12345 {
		t.Errorf("Pid: got %v, want 12345", status.Pid)
	}

	if status.AgentStatus != detector.StateUnknown {
		t.Errorf("AgentStatus: got %v, want unknown", status.AgentStatus)
	}

	if !status.IsRunning {
		t.Error("IsRunning should be true")
	}
}

func TestMonitorUnregisterPod(t *testing.T) {
	monitor := NewMonitor(time.Second)

	monitor.RegisterPod("pod-1", 12345)
	monitor.UnregisterPod("pod-1")

	_, ok := monitor.GetStatus("pod-1")
	if ok {
		t.Error("pod should be unregistered")
	}
}

func TestMonitorGetStatusNotFound(t *testing.T) {
	monitor := NewMonitor(time.Second)

	_, ok := monitor.GetStatus("nonexistent")
	if ok {
		t.Error("should return false for nonexistent pod")
	}
}

func TestMonitorGetAllStatuses(t *testing.T) {
	monitor := NewMonitor(time.Second)

	monitor.RegisterPod("pod-1", 12345)
	monitor.RegisterPod("pod-2", 67890)

	statuses := monitor.GetAllStatuses()

	if len(statuses) != 2 {
		t.Errorf("statuses length: got %v, want 2", len(statuses))
	}
}

func TestMonitorGetAllStatusesEmpty(t *testing.T) {
	monitor := NewMonitor(time.Second)

	statuses := monitor.GetAllStatuses()

	if len(statuses) != 0 {
		t.Errorf("statuses should be empty, got %v", len(statuses))
	}
}

func TestMonitorStartStop(t *testing.T) {
	monitor := NewMonitor(100 * time.Millisecond)

	monitor.Start()

	// Give it time to run a few cycles
	time.Sleep(250 * time.Millisecond)

	monitor.Stop()

	// Should not panic when called twice
	monitor.Stop()
}
