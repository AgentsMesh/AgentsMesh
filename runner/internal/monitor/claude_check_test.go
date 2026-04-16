package monitor

import (
	"sync"
	"testing"
	"time"

	_ "github.com/anthropics/agentsmesh/runner/internal/agents/claude"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// Tests for pod status checking and process inspection

func TestMonitorCheckPodNotRunning(t *testing.T) {
	inspector := newMockInspector()
	monitor := NewMonitorWithInspector(50*time.Millisecond, inspector)

	// Register a pod with a non-running process
	inspector.isRunning[12345] = false

	monitor.Subscribe("test", func(status PodStatus) {
		// callback for status changes
	})

	monitor.RegisterPod("pod-1", 12345)
	monitor.Start()

	// Wait for check to happen
	time.Sleep(150 * time.Millisecond)
	monitor.Stop()

	status, _ := monitor.GetStatus("pod-1")
	if status.IsRunning {
		t.Error("pod should not be running")
	}

	if status.AgentStatus != detector.StateNotRunning {
		t.Errorf("AgentStatus: got %v, want not_running", status.AgentStatus)
	}
}

func TestMonitorCheckPodRunningNoAgent(t *testing.T) {
	inspector := newMockInspector()
	monitor := NewMonitorWithInspector(50*time.Millisecond, inspector)

	// Register a pod with a running process but no agent child
	inspector.isRunning[12345] = true
	inspector.childProcesses[12345] = []int{} // No children

	monitor.RegisterPod("pod-1", 12345)
	monitor.Start()

	// Wait for check to happen
	time.Sleep(150 * time.Millisecond)
	monitor.Stop()

	status, _ := monitor.GetStatus("pod-1")
	if !status.IsRunning {
		t.Error("pod should be running")
	}

	if status.AgentStatus != detector.StateNotRunning {
		t.Errorf("AgentStatus without agent child: got %v, want not_running", status.AgentStatus)
	}
}

func TestMonitorCheckPodWithAgentExecuting(t *testing.T) {
	inspector := newMockInspector()
	monitor := NewMonitorWithInspector(50*time.Millisecond, inspector)

	// Register a pod with agent running and executing
	inspector.isRunning[12345] = true
	inspector.childProcesses[12345] = []int{67890}
	inspector.processNames[67890] = "claude"
	inspector.childProcesses[67890] = []int{11111} // agent has children
	inspector.processStates[11111] = "R"           // child is running

	monitor.RegisterPod("pod-1", 12345)
	monitor.Start()

	// Wait for check to happen
	time.Sleep(150 * time.Millisecond)
	monitor.Stop()

	status, _ := monitor.GetStatus("pod-1")
	if status.AgentPid != 67890 {
		t.Errorf("AgentPid: got %v, want 67890", status.AgentPid)
	}

	if status.AgentStatus != detector.StateExecuting {
		t.Errorf("AgentStatus: got %v, want executing", status.AgentStatus)
	}
}

func TestMonitorCheckPodWithAgentWaiting(t *testing.T) {
	inspector := newMockInspector()
	monitor := NewMonitorWithInspector(50*time.Millisecond, inspector)

	// Register a pod with agent running but waiting (no active children)
	inspector.isRunning[12345] = true
	inspector.childProcesses[12345] = []int{67890}
	inspector.processNames[67890] = "claude"
	inspector.childProcesses[67890] = []int{} // No children

	monitor.RegisterPod("pod-1", 12345)
	monitor.Start()

	// Wait for check to happen
	time.Sleep(150 * time.Millisecond)
	monitor.Stop()

	status, _ := monitor.GetStatus("pod-1")
	if status.AgentStatus != detector.StateWaiting {
		t.Errorf("AgentStatus: got %v, want waiting", status.AgentStatus)
	}
}

func TestMonitorFindAgentProcessRecursive(t *testing.T) {
	inspector := newMockInspector()
	m := NewMonitorWithInspector(time.Second, inspector)

	// Agent is nested under another process
	inspector.childProcesses[1] = []int{2}
	inspector.processNames[2] = "bash"
	inspector.childProcesses[2] = []int{3}
	inspector.processNames[3] = "claude"

	agentPid := m.findAgentProcess(1)
	if agentPid != 3 {
		t.Errorf("findAgentProcess: got %v, want 3", agentPid)
	}
}

func TestMonitorFindAgentProcessNotFound(t *testing.T) {
	inspector := newMockInspector()
	m := NewMonitorWithInspector(time.Second, inspector)

	inspector.childProcesses[1] = []int{2}
	inspector.processNames[2] = "bash"

	agentPid := m.findAgentProcess(1)
	if agentPid != 0 {
		t.Errorf("findAgentProcess: got %v, want 0", agentPid)
	}
}

func TestMonitorStatusChangeCallback(t *testing.T) {
	inspector := newMockInspector()
	monitor := NewMonitorWithInspector(50*time.Millisecond, inspector)

	var receivedStatuses []PodStatus
	var mu sync.Mutex

	monitor.Subscribe("test", func(status PodStatus) {
		mu.Lock()
		receivedStatuses = append(receivedStatuses, status)
		mu.Unlock()
	})

	// Start with running process
	inspector.isRunning[12345] = true
	inspector.childProcesses[12345] = []int{}

	monitor.RegisterPod("pod-1", 12345)
	monitor.Start()

	// Wait for initial check
	time.Sleep(100 * time.Millisecond)

	// Change to have claude running
	inspector.mu.Lock()
	inspector.childProcesses[12345] = []int{67890}
	inspector.processNames[67890] = "claude"
	inspector.mu.Unlock()

	// Wait for another check
	time.Sleep(100 * time.Millisecond)
	monitor.Stop()

	mu.Lock()
	defer mu.Unlock()

	// Should have received at least one status change
	if len(receivedStatuses) == 0 {
		t.Error("should have received status changes")
	}
}
