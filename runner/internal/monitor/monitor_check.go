package monitor

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// monitorLoop periodically checks all pod statuses.
func (m *Monitor) monitorLoop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkAllPods()
		}
	}
}

// checkAllPods checks the status of all registered pods.
// Callbacks are called after releasing the lock to prevent deadlocks.
func (m *Monitor) checkAllPods() {
	// Collect status changes while holding the lock
	var changes []PodStatus

	m.mu.Lock()
	for podID, status := range m.statuses {
		oldStatus := status.AgentStatus

		// Check if shell process is still running
		if !m.inspector.IsRunning(status.Pid) {
			status.IsRunning = false
			status.AgentStatus = detector.StateNotRunning
			status.AgentPid = 0
		} else {
			status.IsRunning = true

			// Check agent status
			agentPid, agentStatus := m.getAgentStatus(status.Pid)
			status.AgentPid = agentPid
			status.AgentStatus = agentStatus
		}

		status.UpdatedAt = time.Now()

		// Collect changes for callback (called after releasing lock)
		if oldStatus != status.AgentStatus {
			log.Info("Agent status changed",
				"pod_id", podID, "old_status", oldStatus, "new_status", status.AgentStatus)
			changes = append(changes, *status)
		}
	}
	m.mu.Unlock()

	// Notify subscribers after releasing the lock to prevent deadlocks
	for _, status := range changes {
		m.notifySubscribers(status)
	}
}

// getAgentStatus checks the status of agent process in the process tree.
func (m *Monitor) getAgentStatus(shellPid int) (int, detector.AgentState) {
	// First check if the shell process itself is an agent (claude/node)
	// This happens when PTY directly runs agent (not via bash)
	shellName := m.inspector.GetProcessName(shellPid)
	if agentkit.IsAgentProcess(shellName) {
		// The shell process IS the agent process
		if m.hasActiveChildren(shellPid) {
			return shellPid, detector.StateExecuting
		}
		return shellPid, detector.StateWaiting
	}

	// Otherwise, find agent process in the process tree
	agentPid := m.findAgentProcess(shellPid)
	if agentPid == 0 {
		return 0, detector.StateNotRunning
	}

	// Check if agent has active child processes
	if m.hasActiveChildren(agentPid) {
		return agentPid, detector.StateExecuting
	}

	return agentPid, detector.StateWaiting
}

// findAgentProcess finds agent process in the process tree rooted at pid.
// It checks registered agent process names via agentkit.IsAgentProcess.
func (m *Monitor) findAgentProcess(pid int) int {
	children := m.inspector.GetChildProcesses(pid)

	for _, childPid := range children {
		name := m.inspector.GetProcessName(childPid)
		if agentkit.IsAgentProcess(name) {
			return childPid
		}

		// Recursively search in children
		if found := m.findAgentProcess(childPid); found != 0 {
			return found
		}
	}

	return 0
}

// hasActiveChildren checks if a process has children that are actively running.
// A process is considered active if:
// - It's in running state (R)
// - It has open file descriptors (doing I/O)
// - It has active grandchildren
func (m *Monitor) hasActiveChildren(pid int) bool {
	children := m.inspector.GetChildProcesses(pid)

	for _, childPid := range children {
		state := m.inspector.GetState(childPid)

		// Check if in running state
		if state == "R" {
			return true
		}

		// Check if process has open files (doing I/O even if sleeping)
		if m.inspector.HasOpenFiles(childPid) {
			return true
		}

		// Recursively check grandchildren
		if m.hasActiveChildren(childPid) {
			return true
		}
	}

	return false
}
