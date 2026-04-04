package updater

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// State represents the current update state.
type State int

const (
	// StateIdle indicates no update in progress.
	StateIdle State = iota
	// StateChecking indicates checking for updates.
	StateChecking
	// StateDownloading indicates downloading an update.
	StateDownloading
	// StateDraining indicates waiting for pods to finish before applying update.
	StateDraining
	// StateApplying indicates applying the update.
	StateApplying
	// StateRestarting indicates restarting after update.
	StateRestarting
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateChecking:
		return "checking"
	case StateDownloading:
		return "downloading"
	case StateDraining:
		return "draining"
	case StateApplying:
		return "applying"
	case StateRestarting:
		return "restarting"
	default:
		return "unknown"
	}
}

// PodCounter is a function that returns the number of active pods.
type PodCounter func() int

// StatusCallback is called when the update status changes.
type StatusCallback func(state State, info *UpdateInfo, activePods int)

// RestartFunc is a function that restarts the application.
// Returns the PID of the new process for health checking.
type RestartFunc func() (pid int, err error)

// HealthChecker validates that the new process is healthy.
// It receives the context and PID of the new process.
type HealthChecker func(ctx context.Context, pid int) error

// GracefulUpdater manages graceful updates with pod awareness.
type GracefulUpdater struct {
	updater       *Updater
	podCounter    PodCounter
	maxWaitTime   time.Duration
	pollInterval  time.Duration
	onStatus      StatusCallback
	restartFunc   RestartFunc
	healthChecker HealthChecker
	healthTimeout time.Duration
	exitFunc      func(code int) // defaults to os.Exit; override in tests

	// State
	mu          sync.RWMutex
	state       State
	draining    bool
	pendingInfo *UpdateInfo
	cancelDrain context.CancelFunc
}

// GracefulOption configures the GracefulUpdater.
type GracefulOption func(*GracefulUpdater)

// WithMaxWaitTime sets the maximum time to wait for pods to finish.
func WithMaxWaitTime(d time.Duration) GracefulOption {
	return func(g *GracefulUpdater) {
		g.maxWaitTime = d
	}
}

// WithPollInterval sets how often to check for pod status during draining.
func WithPollInterval(d time.Duration) GracefulOption {
	return func(g *GracefulUpdater) {
		g.pollInterval = d
	}
}

// WithStatusCallback sets a callback for status updates.
func WithStatusCallback(cb StatusCallback) GracefulOption {
	return func(g *GracefulUpdater) {
		g.onStatus = cb
	}
}

// WithRestartFunc sets a custom restart function.
// The function should return the PID of the new process for health checking.
func WithRestartFunc(f RestartFunc) GracefulOption {
	return func(g *GracefulUpdater) {
		g.restartFunc = f
	}
}

// WithHealthChecker sets a health checker function.
// The health checker validates that the new process is running correctly.
func WithHealthChecker(hc HealthChecker) GracefulOption {
	return func(g *GracefulUpdater) {
		g.healthChecker = hc
	}
}

// WithHealthTimeout sets the timeout for health checking.
// Default is 30 seconds if not set.
func WithHealthTimeout(d time.Duration) GracefulOption {
	return func(g *GracefulUpdater) {
		g.healthTimeout = d
	}
}

// NewGracefulUpdater creates a new GracefulUpdater.
func NewGracefulUpdater(updater *Updater, podCounter PodCounter, opts ...GracefulOption) *GracefulUpdater {
	g := &GracefulUpdater{
		updater:       updater,
		podCounter:    podCounter,
		maxWaitTime:   30 * time.Minute, // Default: 30 minutes
		pollInterval:  5 * time.Second,  // Default: check every 5 seconds
		healthTimeout: 30 * time.Second, // Default: 30 seconds for health check
		exitFunc:      os.Exit,
		state:         StateIdle,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// State returns the current update state.
func (g *GracefulUpdater) State() State {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.state
}

// IsDraining returns true if the updater is waiting for pods to finish.
func (g *GracefulUpdater) IsDraining() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.draining
}

// PendingVersion returns the version waiting to be applied, or empty string if none.
func (g *GracefulUpdater) PendingVersion() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.pendingInfo != nil {
		return g.pendingInfo.LatestVersion
	}
	return ""
}

func (g *GracefulUpdater) setState(state State) {
	g.mu.Lock()
	prev := g.state
	g.state = state
	info := g.pendingInfo
	cb := g.onStatus           // Copy callback reference
	podCounter := g.podCounter // Copy podCounter reference
	g.mu.Unlock()

	if prev != state {
		logger.Updater().Info("Update state changed", "from", prev, "to", state)
	}

	// Callback executed outside lock (avoid deadlock), using snapshot from lock
	if cb != nil {
		activePods := 0
		if podCounter != nil {
			activePods = podCounter()
		}
		cb(state, info, activePods)
	}
}

// CancelPendingUpdate cancels any pending update.
func (g *GracefulUpdater) CancelPendingUpdate() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cancelDrain != nil {
		g.cancelDrain()
	}

	logger.Updater().Info("Pending update cancelled", "was_draining", g.draining)

	g.pendingInfo = nil
	g.draining = false
	g.state = StateIdle
}
