// Package terminal provides terminal state detection for AI agents.
package detector

import (
	"sync"
	"time"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/safego"
)

// OutputActivityDetector detects agent state based on terminal output activity.
// This is the most reliable detection method as it doesn't depend on any
// specific Agent implementation details.
//
// State Machine:
//
//	                 output received
//	    ┌──────────────────────────────────────┐
//	    │                                      │
//	    ▼                                      │
//	┌────────┐    idle > threshold      ┌─────────────┐
//	│Executing│ ─────────────────────►  │PotentialWait│
//	└────────┘                          └─────────────┘
//	    ▲                                      │
//	    │                                      │ confirmed (prompt or timeout)
//	    │        output received               ▼
//	    └────────────────────────────── ┌─────────┐
//	                                    │ Waiting │
//	                                    └─────────┘
type OutputActivityDetector struct {
	mu sync.RWMutex

	// Output tracking
	lastOutputTime time.Time
	outputCount    int64     // bytes in current window
	windowStart    time.Time // start of current measurement window

	// Configuration
	windowDuration   time.Duration // measurement window size
	idleThreshold    time.Duration // how long without output to consider idle
	confirmThreshold time.Duration // additional time to confirm waiting state

	// Current state
	currentState    ActivityState
	idleStartTime   time.Time // when output stopped
	stateChangeTime time.Time // last state change

	// Callback
	onStateChange func(newState, prevState ActivityState)
}

// ActivityState represents the output activity state.
type ActivityState string

const (
	// ActivityStateActive indicates terminal is actively receiving output.
	ActivityStateActive ActivityState = "active"
	// ActivityStatePotentialIdle indicates output stopped but not confirmed idle.
	ActivityStatePotentialIdle ActivityState = "potential_idle"
	// ActivityStateIdle indicates terminal has been idle for a while.
	ActivityStateIdle ActivityState = "idle"
)

// OutputActivityConfig contains configuration for OutputActivityDetector.
type OutputActivityConfig struct {
	// WindowDuration is the measurement window for output rate (default: 1s)
	WindowDuration time.Duration
	// IdleThreshold is how long without output to enter potential idle (default: 500ms)
	IdleThreshold time.Duration
	// ConfirmThreshold is additional time to confirm idle state (default: 1s)
	ConfirmThreshold time.Duration
	// OnStateChange is called when activity state changes
	OnStateChange func(newState, prevState ActivityState)
}

// NewOutputActivityDetector creates a new output activity detector.
func NewOutputActivityDetector(cfg OutputActivityConfig) *OutputActivityDetector {
	if cfg.WindowDuration == 0 {
		cfg.WindowDuration = 1 * time.Second
	}
	if cfg.IdleThreshold == 0 {
		cfg.IdleThreshold = 500 * time.Millisecond
	}
	if cfg.ConfirmThreshold == 0 {
		cfg.ConfirmThreshold = 1 * time.Second
	}

	return &OutputActivityDetector{
		windowDuration:   cfg.WindowDuration,
		idleThreshold:    cfg.IdleThreshold,
		confirmThreshold: cfg.ConfirmThreshold,
		currentState:     ActivityStateIdle,
		onStateChange:    cfg.OnStateChange,
	}
}

// OnOutput should be called whenever terminal output is received.
// This is the primary input signal for the detector.
func (d *OutputActivityDetector) OnOutput(bytes int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	d.lastOutputTime = now

	// Reset window if expired
	if d.windowStart.IsZero() || now.Sub(d.windowStart) > d.windowDuration {
		d.outputCount = 0
		d.windowStart = now
	}
	d.outputCount += int64(bytes)

	// Reset idle tracking
	d.idleStartTime = time.Time{}

	// If we were idle or potentially idle, transition to active
	if d.currentState != ActivityStateActive {
		d.setState(ActivityStateActive)
	}
}

// CheckState checks current state and triggers transitions if needed.
// This should be called periodically (e.g., every 200-500ms).
func (d *OutputActivityDetector) CheckState() ActivityState {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// If never received output, stay idle
	if d.lastOutputTime.IsZero() {
		return d.currentState
	}

	idleDuration := now.Sub(d.lastOutputTime)

	switch d.currentState {
	case ActivityStateActive:
		// Check if we've gone idle
		if idleDuration >= d.idleThreshold {
			d.idleStartTime = d.lastOutputTime
			d.setState(ActivityStatePotentialIdle)
		}

	case ActivityStatePotentialIdle:
		// Check if idle is confirmed
		totalIdleDuration := now.Sub(d.idleStartTime)
		if totalIdleDuration >= d.idleThreshold+d.confirmThreshold {
			d.setState(ActivityStateIdle)
		}
		// Note: transition back to Active happens in OnOutput()

	case ActivityStateIdle:
		// Stay idle until output is received (handled in OnOutput)
	}

	return d.currentState
}

// GetState returns the current activity state.
func (d *OutputActivityDetector) GetState() ActivityState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentState
}

// IsActive returns true if the terminal is actively receiving output.
func (d *OutputActivityDetector) IsActive() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentState == ActivityStateActive
}

// IsIdle returns true if the terminal has been idle (output stopped).
func (d *OutputActivityDetector) IsIdle() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentState == ActivityStateIdle
}

// IsPotentiallyIdle returns true if output stopped but not yet confirmed idle.
func (d *OutputActivityDetector) IsPotentiallyIdle() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentState == ActivityStatePotentialIdle
}

// IdleDuration returns how long since the last output was received.
func (d *OutputActivityDetector) IdleDuration() time.Duration {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.lastOutputTime.IsZero() {
		return 0
	}
	return time.Since(d.lastOutputTime)
}

// GetOutputRate returns the output rate in bytes per second.
func (d *OutputActivityDetector) GetOutputRate() float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.windowStart.IsZero() {
		return 0
	}

	elapsed := time.Since(d.windowStart).Seconds()
	if elapsed <= 0 {
		return 0
	}

	return float64(d.outputCount) / elapsed
}

// Reset resets the detector state.
func (d *OutputActivityDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.lastOutputTime = time.Time{}
	d.outputCount = 0
	d.windowStart = time.Time{}
	d.idleStartTime = time.Time{}
	d.currentState = ActivityStateIdle
	d.stateChangeTime = time.Time{}
}

// SetCallback sets the state change callback.
func (d *OutputActivityDetector) SetCallback(cb func(newState, prevState ActivityState)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onStateChange = cb
}

// setState updates the current state and triggers callback.
func (d *OutputActivityDetector) setState(newState ActivityState) {
	if d.currentState == newState {
		return
	}

	prevState := d.currentState
	d.currentState = newState
	d.stateChangeTime = time.Now()

	if d.onStateChange != nil {
		cb := d.onStateChange
		safego.Go("activity-callback", func() { cb(newState, prevState) })
	}
}
