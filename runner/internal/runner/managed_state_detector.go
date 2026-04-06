package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// ManagedStateDetector wraps detector.MultiSignalDetector and adds:
// - Background detection loop for timeout-based state transitions
// - Lifecycle management (Start/Stop)
// - Configuration of detection parameters
//
// This implements detector.StateDetector interface, providing a foundation
// service that can be used by Autopilot, Monitor, or any other component.
type ManagedStateDetector struct {
	detector *detector.MultiSignalDetector
	vt       *vt.VirtualTerminal
	ctx      context.Context
	cancel   context.CancelFunc
}

// Compile-time interface check
var _ detector.StateDetector = (*ManagedStateDetector)(nil)

// NewManagedStateDetector creates a new managed state detector.
// It starts a background goroutine to periodically run detection for timeout-based transitions.
func NewManagedStateDetector(vt *vt.VirtualTerminal) *ManagedStateDetector {
	d := detector.NewMultiSignalDetector(detector.MultiSignalConfig{
		// Responsive detection thresholds
		IdleThreshold:    500 * time.Millisecond,
		ConfirmThreshold: 500 * time.Millisecond,
		MinStableTime:    300 * time.Millisecond,
		WaitingThreshold: 0.6,
	})

	ctx, cancel := context.WithCancel(context.Background())
	m := &ManagedStateDetector{
		detector: d,
		vt:       vt,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Subscribe to state changes for structured logging of transitions.
	d.Subscribe("logger", func(event detector.StateChangeEvent) {
		logger.Terminal().Info("Agent state changed",
			"from", event.PrevState, "to", event.NewState)
	})

	// Start background detection loop
	logger.Terminal().Info("State detector started")
	safego.Go("state-detector", m.runDetectionLoop)

	return m
}

// runDetectionLoop periodically runs state detection.
// Screen content is pushed via OnScreenUpdate from OutputHandler (single-direction data flow).
// This loop only triggers periodic DetectState() for timeout-based transitions.
func (m *ManagedStateDetector) runDetectionLoop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.detector.DetectState()
		}
	}
}

// OnScreenUpdate should be called after VirtualTerminal.Feed with the current screen lines.
// This implements single-direction data flow: PTY → VT.Feed → OnScreenUpdate → StateDetector
func (m *ManagedStateDetector) OnScreenUpdate(lines []string) {
	m.detector.OnScreenUpdate(lines)
}

// OnOutput should be called when terminal output is received.
func (m *ManagedStateDetector) OnOutput(bytes int) {
	logger.TerminalTrace().Trace("ManagedStateDetector.OnOutput", "bytes", bytes)
	m.detector.OnOutput(bytes)
}

// OnOSCTitle should be called when an OSC title update is received.
func (m *ManagedStateDetector) OnOSCTitle(title string) {
	m.detector.OnOSCTitle(title)
}

// DetectState analyzes and returns the current agent state.
func (m *ManagedStateDetector) DetectState() detector.AgentState {
	return m.detector.DetectState()
}

// GetState returns the current state without performing detection.
func (m *ManagedStateDetector) GetState() detector.AgentState {
	return m.detector.GetState()
}

// Subscribe adds a subscriber for state change events.
// The subscriber ID must be unique; duplicate IDs will replace existing subscriptions.
func (m *ManagedStateDetector) Subscribe(id string, cb func(detector.StateChangeEvent)) {
	m.detector.Subscribe(id, cb)
}

// Unsubscribe removes a subscriber by ID.
func (m *ManagedStateDetector) Unsubscribe(id string) {
	m.detector.Unsubscribe(id)
}

// Reset resets the detector state.
func (m *ManagedStateDetector) Reset() {
	m.detector.Reset()
}

// Stop stops the background detection loop.
func (m *ManagedStateDetector) Stop() {
	logger.Terminal().Info("State detector stopping")
	m.cancel()
}
