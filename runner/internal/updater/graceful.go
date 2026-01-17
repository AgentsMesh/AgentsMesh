package updater

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
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

	// State
	mu            sync.RWMutex
	state         State
	draining      bool
	pendingPath   string
	pendingInfo   *UpdateInfo
	cancelDrain   context.CancelFunc
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

// WithRestartFuncLegacy provides backward compatibility for old restart functions.
// Use WithRestartFunc for new code that supports health checking.
func WithRestartFuncLegacy(f func() error) GracefulOption {
	return func(g *GracefulUpdater) {
		g.restartFunc = func() (int, error) {
			return 0, f()
		}
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
	g.state = state
	info := g.pendingInfo
	cb := g.onStatus         // Copy callback reference
	podCounter := g.podCounter // Copy podCounter reference
	g.mu.Unlock()

	// Callback executed outside lock (avoid deadlock), using snapshot from lock
	if cb != nil {
		activePods := 0
		if podCounter != nil {
			activePods = podCounter()
		}
		cb(state, info, activePods)
	}
}

// ScheduleUpdate checks for updates and schedules a graceful update if available.
// It downloads the update, then waits for all pods to finish before applying.
// If maxWaitTime is reached, the update is postponed to the next check cycle.
func (g *GracefulUpdater) ScheduleUpdate(ctx context.Context) error {
	// Atomically check and set state to avoid race condition
	g.mu.Lock()
	if g.state != StateIdle {
		currentState := g.state
		g.mu.Unlock()
		return fmt.Errorf("update already in progress (state: %s)", currentState)
	}
	g.state = StateChecking
	g.mu.Unlock()

	// Notify status change
	if g.onStatus != nil {
		g.onStatus(StateChecking, nil, 0)
	}
	info, err := g.updater.CheckForUpdate(ctx)
	if err != nil {
		g.setState(StateIdle)
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !info.HasUpdate {
		g.setState(StateIdle)
		return nil
	}

	log.Printf("[updater] New version available: %s -> %s", info.CurrentVersion, info.LatestVersion)

	// Download update
	g.setState(StateDownloading)
	g.mu.Lock()
	g.pendingInfo = info
	g.mu.Unlock()

	tmpPath, err := g.updater.Download(ctx, info.LatestVersion, nil)
	if err != nil {
		g.setState(StateIdle)
		g.mu.Lock()
		g.pendingInfo = nil
		g.mu.Unlock()
		return fmt.Errorf("failed to download update: %w", err)
	}

	g.mu.Lock()
	g.pendingPath = tmpPath
	g.mu.Unlock()

	log.Printf("[updater] Update downloaded to %s, waiting for pods to finish...", tmpPath)

	// Wait for pods to finish
	return g.waitAndApply(ctx)
}

// waitAndApply waits for all pods to finish, then applies the update.
func (g *GracefulUpdater) waitAndApply(ctx context.Context) error {
	g.setState(StateDraining)
	g.mu.Lock()
	g.draining = true

	// Create cancellable context for draining
	drainCtx, cancel := context.WithTimeout(ctx, g.maxWaitTime)
	g.cancelDrain = cancel
	g.mu.Unlock()

	defer func() {
		cancel() // Always release context resources
		g.mu.Lock()
		g.draining = false
		g.cancelDrain = nil
		g.mu.Unlock()
	}()

	// Poll until no pods are active or timeout
	ticker := time.NewTicker(g.pollInterval)
	defer ticker.Stop()

	for {
		activePods := 0
		if g.podCounter != nil {
			activePods = g.podCounter()
		}

		if activePods == 0 {
			log.Printf("[updater] No active pods, applying update...")
			break
		}

		log.Printf("[updater] Waiting for %d active pod(s) to finish...", activePods)

		// Notify status
		if g.onStatus != nil {
			g.mu.RLock()
			info := g.pendingInfo
			g.mu.RUnlock()
			g.onStatus(StateDraining, info, activePods)
		}

		select {
		case <-drainCtx.Done():
			// Timeout or cancelled
			if drainCtx.Err() == context.DeadlineExceeded {
				log.Printf("[updater] Max wait time reached with %d active pods, postponing update", activePods)
				// Clean up pending update
				g.mu.Lock()
				if g.pendingPath != "" {
					os.Remove(g.pendingPath)
					g.pendingPath = ""
				}
				g.pendingInfo = nil
				g.mu.Unlock()
				g.setState(StateIdle)
				return fmt.Errorf("update postponed: max wait time reached with active pods")
			}
			// Cancelled
			g.mu.Lock()
			if g.pendingPath != "" {
				os.Remove(g.pendingPath)
				g.pendingPath = ""
			}
			g.pendingInfo = nil
			g.mu.Unlock()
			g.setState(StateIdle)
			return fmt.Errorf("update cancelled")
		case <-ticker.C:
			// Continue polling
		}
	}

	// Apply the update
	return g.applyPendingUpdate()
}

// applyPendingUpdate applies the downloaded update.
func (g *GracefulUpdater) applyPendingUpdate() error {
	g.mu.Lock()
	tmpPath := g.pendingPath
	info := g.pendingInfo
	g.mu.Unlock()

	if tmpPath == "" {
		g.setState(StateIdle)
		return fmt.Errorf("no pending update to apply")
	}

	g.setState(StateApplying)

	// Create backup for potential rollback
	backupPath, err := g.updater.CreateBackup()
	if err != nil {
		log.Printf("[updater] Warning: failed to create backup: %v", err)
		// Continue without backup - rollback won't be possible
	}

	// Apply update
	if err := g.updater.Apply(tmpPath); err != nil {
		g.mu.Lock()
		g.pendingPath = ""
		g.pendingInfo = nil
		g.mu.Unlock()
		g.setState(StateIdle)
		return fmt.Errorf("failed to apply update: %w", err)
	}

	g.mu.Lock()
	g.pendingPath = ""
	g.pendingInfo = nil
	g.mu.Unlock()

	log.Printf("[updater] Update applied successfully: %s -> %s", info.CurrentVersion, info.LatestVersion)

	// Restart
	g.setState(StateRestarting)
	if g.restartFunc != nil {
		pid, err := g.restartFunc()
		if err != nil {
			log.Printf("[updater] Restart failed, attempting rollback: %v", err)
			if rbErr := g.rollbackUpdate(backupPath); rbErr != nil {
				log.Printf("[updater] Rollback also failed: %v", rbErr)
			}
			g.setState(StateIdle)
			return fmt.Errorf("restart failed: %w", err)
		}

		// Health check if configured
		if g.healthChecker != nil && pid > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), g.healthTimeout)
			defer cancel()

			if err := g.healthChecker(ctx, pid); err != nil {
				log.Printf("[updater] Health check failed, attempting rollback: %v", err)
				// Terminate the unhealthy new process
				if proc, findErr := os.FindProcess(pid); findErr == nil && proc != nil {
					_ = proc.Kill()
				}
				if rbErr := g.rollbackUpdate(backupPath); rbErr != nil {
					log.Printf("[updater] Rollback also failed: %v", rbErr)
				}
				g.setState(StateIdle)
				return fmt.Errorf("health check failed: %w", err)
			}
			log.Printf("[updater] Health check passed for new process (PID: %d)", pid)
		}
	}

	return nil
}

// rollbackUpdate attempts to restore the previous version from backup.
func (g *GracefulUpdater) rollbackUpdate(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("no backup available for rollback")
	}
	if err := g.updater.Rollback(); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	log.Printf("[updater] Successfully rolled back to previous version")
	return nil
}

// ForceUpdate applies the update immediately without waiting for pods.
func (g *GracefulUpdater) ForceUpdate(ctx context.Context) error {
	g.mu.Lock()
	if g.state != StateIdle && g.state != StateDraining {
		currentState := g.state
		g.mu.Unlock()
		return fmt.Errorf("cannot force update in state: %s", currentState)
	}

	// Cancel ongoing drain if any
	if g.cancelDrain != nil {
		g.cancelDrain()
	}

	// Check if we have a pending update while still holding the lock
	hasPending := g.pendingPath != ""
	if !hasPending {
		// Set state to checking before releasing lock to prevent race
		g.state = StateChecking
	}
	g.mu.Unlock()

	// If we have a pending update, apply it
	if hasPending {
		return g.applyPendingUpdate()
	}

	// Notify status change (state already set above)
	if g.onStatus != nil {
		g.onStatus(StateChecking, nil, 0)
	}
	info, err := g.updater.CheckForUpdate(ctx)
	if err != nil {
		g.setState(StateIdle)
		return err
	}

	if !info.HasUpdate {
		g.setState(StateIdle)
		return fmt.Errorf("no update available")
	}

	g.setState(StateDownloading)
	g.mu.Lock()
	g.pendingInfo = info
	g.mu.Unlock()

	tmpPath, err := g.updater.Download(ctx, info.LatestVersion, nil)
	if err != nil {
		g.setState(StateIdle)
		return err
	}

	g.mu.Lock()
	g.pendingPath = tmpPath
	g.mu.Unlock()

	return g.applyPendingUpdate()
}

// CancelPendingUpdate cancels any pending update.
func (g *GracefulUpdater) CancelPendingUpdate() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cancelDrain != nil {
		g.cancelDrain()
	}

	if g.pendingPath != "" {
		os.Remove(g.pendingPath)
		g.pendingPath = ""
	}
	g.pendingInfo = nil
	g.draining = false
	g.state = StateIdle
}

// DefaultRestartFunc returns a restart function that re-executes the current binary.
// Note: This function starts a new process and signals the caller to exit gracefully.
// The caller should handle process termination appropriately.
// Returns the PID of the new process for health checking.
func DefaultRestartFunc() RestartFunc {
	return func() (int, error) {
		execPath, err := os.Executable()
		if err != nil {
			return 0, fmt.Errorf("failed to get executable path: %w", err)
		}

		// Start new process
		cmd := exec.Command(execPath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = os.Environ()

		if err := cmd.Start(); err != nil {
			return 0, fmt.Errorf("failed to start new process: %w", err)
		}

		log.Printf("[updater] New process started (PID: %d), current process should exit", cmd.Process.Pid)
		// Note: Caller is responsible for graceful shutdown after this returns
		// Do NOT call os.Exit() here as it prevents proper cleanup
		return cmd.Process.Pid, nil
	}
}

// DefaultHealthChecker returns a health checker that validates the new process is running.
// minRunTime: the minimum time the new process should run before being considered healthy.
func DefaultHealthChecker(minRunTime time.Duration) HealthChecker {
	return func(ctx context.Context, pid int) error {
		// Wait for the specified minimum run time
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(minRunTime):
		}

		// Check if the process is still running
		proc, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("process not found: %w", err)
		}

		// On Unix systems, Signal(0) checks if the process exists without sending a signal
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			return fmt.Errorf("process not running: %w", err)
		}

		return nil
	}
}
