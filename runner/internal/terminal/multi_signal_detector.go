// Package terminal provides terminal state detection for AI agents.
package terminal

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Note: AgentState, StateNotRunning, StateExecuting, StateWaiting, and StateChangeCallback
// are defined in state_detector.go to avoid redeclaration

// MultiSignalDetector detects agent state by fusing multiple signals.
// This approach is Agent-agnostic and doesn't depend on specific implementations.
//
// Signals and their weights:
//   - Output Activity (0.4): Most reliable - based on terminal output patterns
//   - Screen Stability (0.3): Terminal content hasn't changed
//   - Prompt Detection (0.3): Generic prompt patterns detected
//   - OSC Hints (optional): Boost confidence if available, but don't depend on it
//
// State Machine:
//
//	                 output received
//	    ┌──────────────────────────────────────┐
//	    │                                      │
//	    ▼                                      │
//	┌────────┐    confidence > threshold  ┌─────────┐
//	│Executing│ ───────────────────────►  │ Waiting │
//	└────────┘                            └─────────┘
//	    ▲                                      │
//	    │        output received               │
//	    └──────────────────────────────────────┘
type MultiSignalDetector struct {
	mu sync.RWMutex

	// Sub-detectors
	activityDetector *OutputActivityDetector
	promptDetector   *PromptDetector

	// Screen stability tracking
	lastScreenHash   string
	lastScreenTime   time.Time
	screenStableTime time.Duration

	// OSC title tracking (optional signal)
	lastOSCTitle     string
	lastOSCTitleTime time.Time

	// Configuration
	config MultiSignalConfig

	// Current state
	currentState    AgentState
	stateChangeTime time.Time
	lastCheckTime   time.Time

	// Callback
	onStateChange StateChangeCallback

	// Screen content for prompt detection
	screenLines []string
}

// MultiSignalConfig contains configuration for MultiSignalDetector.
type MultiSignalConfig struct {
	// ActivityWeight is the weight for output activity signal (default: 0.4)
	ActivityWeight float64
	// StabilityWeight is the weight for screen stability signal (default: 0.3)
	StabilityWeight float64
	// PromptWeight is the weight for prompt detection signal (default: 0.3)
	PromptWeight float64

	// MinStableTime is the minimum time screen must be stable (default: 500ms)
	MinStableTime time.Duration
	// WaitingThreshold is the confidence threshold to transition to waiting (default: 0.6)
	WaitingThreshold float64

	// IdleThreshold for activity detector (default: 500ms)
	IdleThreshold time.Duration
	// ConfirmThreshold for activity detector (default: 500ms)
	ConfirmThreshold time.Duration

	// MaxPromptLength for prompt detector (default: 100)
	MaxPromptLength int

	// OnStateChange callback
	OnStateChange StateChangeCallback
}

// NewMultiSignalDetector creates a new multi-signal detector.
func NewMultiSignalDetector(cfg MultiSignalConfig) *MultiSignalDetector {
	// Apply defaults
	if cfg.ActivityWeight == 0 {
		cfg.ActivityWeight = 0.4
	}
	if cfg.StabilityWeight == 0 {
		cfg.StabilityWeight = 0.3
	}
	if cfg.PromptWeight == 0 {
		cfg.PromptWeight = 0.3
	}
	if cfg.MinStableTime == 0 {
		cfg.MinStableTime = 500 * time.Millisecond
	}
	if cfg.WaitingThreshold == 0 {
		cfg.WaitingThreshold = 0.6
	}
	if cfg.IdleThreshold == 0 {
		cfg.IdleThreshold = 500 * time.Millisecond
	}
	if cfg.ConfirmThreshold == 0 {
		cfg.ConfirmThreshold = 500 * time.Millisecond
	}
	if cfg.MaxPromptLength == 0 {
		cfg.MaxPromptLength = 100
	}

	// Create sub-detectors
	activityDetector := NewOutputActivityDetector(OutputActivityConfig{
		IdleThreshold:    cfg.IdleThreshold,
		ConfirmThreshold: cfg.ConfirmThreshold,
	})

	promptDetector := NewPromptDetector(PromptDetectorConfig{
		MaxPromptLength: cfg.MaxPromptLength,
	})

	return &MultiSignalDetector{
		activityDetector: activityDetector,
		promptDetector:   promptDetector,
		config:           cfg,
		currentState:     StateNotRunning,
		onStateChange:    cfg.OnStateChange,
	}
}

// GetState returns the current state without performing detection.
func (d *MultiSignalDetector) GetState() AgentState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentState
}

// SetCallback sets the state change callback.
func (d *MultiSignalDetector) SetCallback(cb StateChangeCallback) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onStateChange = cb
}

// Reset resets the detector to initial state.
func (d *MultiSignalDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.activityDetector.Reset()
	d.lastScreenHash = ""
	d.lastScreenTime = time.Time{}
	d.screenStableTime = 0
	d.lastOSCTitle = ""
	d.lastOSCTitleTime = time.Time{}
	d.currentState = StateNotRunning
	d.stateChangeTime = time.Time{}
	d.screenLines = nil
}

// SetProcessRunning should be called when the agent process starts/stops.
func (d *MultiSignalDetector) SetProcessRunning(running bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !running {
		d.setState(StateNotRunning)
	} else if d.currentState == StateNotRunning {
		// Process started, wait for output to transition to Executing
		// Don't transition immediately - wait for actual output
	}
}

// GetActivityDetector returns the underlying activity detector for direct access.
func (d *MultiSignalDetector) GetActivityDetector() *OutputActivityDetector {
	return d.activityDetector
}

// GetPromptDetector returns the underlying prompt detector for direct access.
func (d *MultiSignalDetector) GetPromptDetector() *PromptDetector {
	return d.promptDetector
}

// GetLastPromptResult returns the last prompt detection result.
func (d *MultiSignalDetector) GetLastPromptResult() PromptResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.screenLines) == 0 {
		return PromptResult{}
	}
	return d.promptDetector.DetectPrompt(d.screenLines)
}

// GetScreenStableTime returns how long the screen has been stable.
func (d *MultiSignalDetector) GetScreenStableTime() time.Duration {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.screenStableTime
}

// GetIdleDuration returns how long since the last output.
func (d *MultiSignalDetector) GetIdleDuration() time.Duration {
	return d.activityDetector.IdleDuration()
}

// setState updates the current state and triggers callback.
func (d *MultiSignalDetector) setState(newState AgentState) {
	if d.currentState == newState {
		return
	}

	prevState := d.currentState
	d.currentState = newState
	d.stateChangeTime = time.Now()

	if d.onStateChange != nil {
		cb := d.onStateChange
		go cb(newState, prevState)
	}
}

// OnOutput should be called whenever terminal output is received.
// This is the primary input signal.
func (d *MultiSignalDetector) OnOutput(bytes int) {
	d.mu.Lock()
	currentState := d.currentState

	// Forward to activity detector
	d.activityDetector.OnOutput(bytes)

	// If we were in NotRunning or Waiting, transition to Executing
	if d.currentState != StateExecuting {
		d.setState(StateExecuting)
	}
	d.mu.Unlock()

	// Debug logging OUTSIDE lock to avoid blocking PTY output
	logger.TerminalTrace().Trace("MultiSignalDetector OnOutput called",
		"bytes", bytes,
		"current_state", currentState)
}
