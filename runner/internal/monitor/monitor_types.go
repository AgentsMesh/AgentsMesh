// Package monitor provides process monitoring functionality.
package monitor

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/process"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// Module logger for monitor
var log = logger.Monitor()

// PodStatus represents the full status of a pod.
type PodStatus struct {
	PodID       string              `json:"pod_id"`
	Pid         int                 `json:"pid"`
	AgentStatus detector.AgentState `json:"agent_status"`
	AgentPid    int                 `json:"agent_pid,omitempty"`
	IsRunning   bool                `json:"is_running"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Monitor monitors pod processes for claude status.
type Monitor struct {
	statuses map[string]*PodStatus
	mu       sync.RWMutex

	// Process inspector (injectable for testing)
	inspector process.Inspector

	// Subscribers for status changes (key: subscriber ID, value: callback)
	// Supports multiple subscribers instead of single callback to allow
	// multiple AutopilotControllers to receive status notifications
	subscribers map[string]func(PodStatus)
	subMu       sync.RWMutex

	// Check interval
	interval time.Duration
	stopCh   chan struct{}
	stopped  bool
	stopOnce sync.Once
}

// NewMonitor creates a new process monitor.
func NewMonitor(interval time.Duration) *Monitor {
	return &Monitor{
		statuses:    make(map[string]*PodStatus),
		subscribers: make(map[string]func(PodStatus)),
		inspector:   process.DefaultInspector(),
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

// NewMonitorWithInspector creates a new monitor with a custom inspector (for testing).
func NewMonitorWithInspector(interval time.Duration, inspector process.Inspector) *Monitor {
	return &Monitor{
		statuses:    make(map[string]*PodStatus),
		subscribers: make(map[string]func(PodStatus)),
		inspector:   inspector,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}
