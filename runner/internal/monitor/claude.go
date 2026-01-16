// Package monitor provides process monitoring functionality.
package monitor

import (
	"log"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// ClaudeStatus represents the status of claude process.
type ClaudeStatus string

const (
	StatusUnknown    ClaudeStatus = "unknown"
	StatusNotRunning ClaudeStatus = "not_running"
	StatusExecuting  ClaudeStatus = "executing"
	StatusWaiting    ClaudeStatus = "waiting"
)

// PodStatus represents the full status of a pod.
type PodStatus struct {
	PodID        string       `json:"pod_id"`
	Pid          int          `json:"pid"`
	ClaudeStatus ClaudeStatus `json:"claude_status"`
	ClaudePid    int          `json:"claude_pid,omitempty"`
	IsRunning    bool         `json:"is_running"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Monitor monitors pod processes for claude status.
type Monitor struct {
	statuses map[string]*PodStatus
	mu       sync.RWMutex

	// Process inspector (injectable for testing)
	inspector process.Inspector

	// Callback when status changes
	onStatusChange func(PodStatus)

	// Check interval
	interval time.Duration
	stopCh   chan struct{}
	stopped  bool
	stopOnce sync.Once
}

// NewMonitor creates a new process monitor.
func NewMonitor(interval time.Duration) *Monitor {
	return &Monitor{
		statuses:  make(map[string]*PodStatus),
		inspector: process.DefaultInspector(),
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// NewMonitorWithInspector creates a new monitor with a custom inspector (for testing).
func NewMonitorWithInspector(interval time.Duration, inspector process.Inspector) *Monitor {
	return &Monitor{
		statuses:  make(map[string]*PodStatus),
		inspector: inspector,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// SetCallback sets the status change callback.
func (m *Monitor) SetCallback(callback func(PodStatus)) {
	m.onStatusChange = callback
}

// RegisterPod registers a pod for monitoring.
func (m *Monitor) RegisterPod(podID string, pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.statuses[podID] = &PodStatus{
		PodID:        podID,
		Pid:          pid,
		ClaudeStatus: StatusUnknown,
		IsRunning:    true,
		UpdatedAt:    time.Now(),
	}

	log.Printf("[monitor] Registered pod %s (pid: %d) for monitoring", podID, pid)
}

// UnregisterPod removes a pod from monitoring.
func (m *Monitor) UnregisterPod(podID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.statuses, podID)

	log.Printf("[monitor] Unregistered pod %s from monitoring", podID)
}

// GetStatus returns the current status of a pod.
func (m *Monitor) GetStatus(podID string) (PodStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if status, exists := m.statuses[podID]; exists {
		return *status, true
	}
	return PodStatus{}, false
}

// GetAllStatuses returns all pod statuses.
func (m *Monitor) GetAllStatuses() []PodStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]PodStatus, 0, len(m.statuses))
	for _, status := range m.statuses {
		result = append(result, *status)
	}
	return result
}

// Start starts the monitoring loop.
func (m *Monitor) Start() {
	go m.monitorLoop()
	log.Printf("[monitor] Started process monitor with interval %v", m.interval)
}

// Stop stops the monitoring loop.
func (m *Monitor) Stop() {
	m.stopOnce.Do(func() {
		m.mu.Lock()
		m.stopped = true
		m.mu.Unlock()
		close(m.stopCh)
		log.Println("[monitor] Stopped process monitor")
	})
}

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
			log.Printf("[monitor] Pod %s: Claude status changed from %s to %s",
				podID, oldStatus, status.ClaudeStatus)
			changes = append(changes, *status)
		}
	}
	m.mu.Unlock()

	// Call callbacks after releasing the lock to prevent deadlocks
	if m.onStatusChange != nil {
		for _, status := range changes {
			m.onStatusChange(status)
		}
	}
}

// getClaudeStatus checks the status of claude process in the process tree.
func (m *Monitor) getClaudeStatus(shellPid int) (int, ClaudeStatus) {
	// Find claude process in the process tree
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
func (m *Monitor) findClaudeProcess(pid int) int {
	// Get direct children
	children := m.inspector.GetChildProcesses(pid)

	for _, childPid := range children {
		name := m.inspector.GetProcessName(childPid)
		if name == "claude" {
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
