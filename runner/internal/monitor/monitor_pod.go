package monitor

import (
	"time"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/safego"
	"github.com/AgentsMesh/AgentsMesh/runner/internal/terminal/detector"
)

// RegisterPod registers a pod for monitoring.
func (m *Monitor) RegisterPod(podID string, pid int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.statuses[podID] = &PodStatus{
		PodID:       podID,
		Pid:         pid,
		AgentStatus: detector.StateUnknown,
		IsRunning:   true,
		UpdatedAt:   time.Now(),
	}

	log.Info("Registered pod for monitoring", "pod_id", podID, "pid", pid)
}

// UnregisterPod removes a pod from monitoring.
func (m *Monitor) UnregisterPod(podID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.statuses, podID)

	log.Info("Unregistered pod from monitoring", "pod_id", podID)
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
	safego.Go("agent-monitor", m.monitorLoop)
	log.Info("Started process monitor", "interval", m.interval)
}

// Stop stops the monitoring loop.
func (m *Monitor) Stop() {
	m.stopOnce.Do(func() {
		m.mu.Lock()
		m.stopped = true
		m.mu.Unlock()
		close(m.stopCh)
		log.Info("Stopped process monitor")
	})
}
