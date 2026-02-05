package monitor

import (
	"time"
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
		oldStatus := status.ClaudeStatus

		// Check if shell process is still running
		if !m.inspector.IsRunning(status.Pid) {
			status.IsRunning = false
			status.ClaudeStatus = StatusNotRunning
			status.ClaudePid = 0
		} else {
			status.IsRunning = true

			// Check claude status
			claudePid, claudeStatus := m.getClaudeStatus(status.Pid)
			status.ClaudePid = claudePid
			status.ClaudeStatus = claudeStatus
		}

		status.UpdatedAt = time.Now()

		// Collect changes for callback (called after releasing lock)
		if oldStatus != status.ClaudeStatus {
			log.Info("Claude status changed",
				"pod_id", podID, "old_status", oldStatus, "new_status", status.ClaudeStatus)
			changes = append(changes, *status)
		}
	}
	m.mu.Unlock()

	// Notify subscribers after releasing the lock to prevent deadlocks
	for _, status := range changes {
		m.notifySubscribers(status)
	}
}

// getClaudeStatus checks the status of claude process in the process tree.
func (m *Monitor) getClaudeStatus(shellPid int) (int, ClaudeStatus) {
	// First check if the shell process itself is claude/node
	// This happens when PTY directly runs claude (not via bash)
	shellName := m.inspector.GetProcessName(shellPid)
	if shellName == "claude" || shellName == "node" {
		// The shell process IS the claude process
		if m.hasActiveChildren(shellPid) {
			return shellPid, StatusExecuting
		}
		return shellPid, StatusWaiting
	}

	// Otherwise, find claude process in the process tree
	claudePid := m.findClaudeProcess(shellPid)
	if claudePid == 0 {
		return 0, StatusNotRunning
	}

	// Check if claude has active child processes
	if m.hasActiveChildren(claudePid) {
		return claudePid, StatusExecuting
	}

	return claudePid, StatusWaiting
}

// findClaudeProcess finds claude process in the process tree rooted at pid.
// It looks for processes named "claude" or "node" (since Claude CLI is Node.js based).
func (m *Monitor) findClaudeProcess(pid int) int {
	// Get direct children
	children := m.inspector.GetChildProcesses(pid)

	for _, childPid := range children {
		name := m.inspector.GetProcessName(childPid)
		// Claude CLI can appear as "claude" or "node" depending on how it's invoked
		if name == "claude" || name == "node" {
			return childPid
		}

		// Recursively search in children
		if found := m.findClaudeProcess(childPid); found != 0 {
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
