// Package detector provides terminal state detection for AI agents.
package detector

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Note: AgentState, StateNotRunning, StateExecuting, StateWaiting, and StateChangeCallback
// are defined in agent_state.go. StateDetector interface and StateChangeEvent are defined
// in state_detector.go.

// Compile-time interface check
var _ StateDetector = (*MultiSignalDetector)(nil)

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
//	    +--------------------------------------+
//	    |                                      |
//	    v                                      |
//	+--------+    confidence > threshold  +---------+
//	|Executing| ----------------------->  | Waiting |
//	+--------+                            +---------+
//	    ^                                      |
//	    |        output received               |
//	    +--------------------------------------+
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
	lastConfidence  float64 // Last calculated waiting confidence

	// Legacy callback (for backward compatibility)
	// Deprecated: Use Subscribe for multiple subscribers support.
	onStateChange StateChangeCallback

	// Multi-subscriber support
	subscribers map[string]func(StateChangeEvent)
	subMu       sync.RWMutex // Separate lock for subscribers to avoid deadlock

	// Screen content for prompt detection
	screenLines []string
}

// NewMultiSignalDetector creates a new multi-signal detector.
func NewMultiSignalDetector(cfg MultiSignalConfig) *MultiSignalDetector {
	cfg.applyDefaults()

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
		subscribers:      make(map[string]func(StateChangeEvent)),
	}
}

// GetState returns the current state without performing detection.
func (d *MultiSignalDetector) GetState() AgentState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentState
}

// SetCallback sets the state change callback.
// Deprecated: Use Subscribe for multiple subscribers support.
func (d *MultiSignalDetector) SetCallback(cb StateChangeCallback) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onStateChange = cb
}

// Subscribe adds a subscriber for state change events.
// The subscriber ID must be unique; duplicate IDs will replace existing subscriptions.
// Callbacks are invoked asynchronously in separate goroutines.
func (d *MultiSignalDetector) Subscribe(id string, cb func(StateChangeEvent)) {
	d.subMu.Lock()
	defer d.subMu.Unlock()
	d.subscribers[id] = cb
}

// Unsubscribe removes a subscriber by ID.
func (d *MultiSignalDetector) Unsubscribe(id string) {
	d.subMu.Lock()
	defer d.subMu.Unlock()
	delete(d.subscribers, id)
}

// Reset resets the detector to initial state.
// Note: Subscribers are NOT cleared; they should unsubscribe explicitly.
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
	d.lastConfidence = 0
	d.screenLines = nil
}

// SetProcessRunning should be called when the agent process starts/stops.
func (d *MultiSignalDetector) SetProcessRunning(running bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !running {
		d.setState(StateNotRunning)
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
