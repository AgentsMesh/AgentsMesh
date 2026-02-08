package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/autopilot"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

// PodControllerImpl implements autopilot.PodController interface.
// It provides the AutopilotController with the ability to interact with the Pod.
type PodControllerImpl struct {
	pod    *Pod
	runner *Runner
}

// NewPodController creates a new PodController.
func NewPodController(pod *Pod, runner *Runner) *PodControllerImpl {
	return &PodControllerImpl{
		pod:    pod,
		runner: runner,
	}
}

// SendTerminalText sends text to the pod's terminal.
func (c *PodControllerImpl) SendTerminalText(text string) error {
	if c.pod.Terminal == nil {
		return nil
	}
	// Add newline to send the command
	return c.pod.Terminal.Write([]byte(text + "\n"))
}

// GetWorkDir returns the pod's working directory.
func (c *PodControllerImpl) GetWorkDir() string {
	return c.pod.SandboxPath
}

// GetPodKey returns the pod's key.
func (c *PodControllerImpl) GetPodKey() string {
	return c.pod.PodKey
}

// GetAgentStatus returns the pod's agent status.
func (c *PodControllerImpl) GetAgentStatus() string {
	agentStatus, _, _, _ := c.runner.GetPodStatus(c.pod.PodKey)
	return agentStatus
}

// GetStateDetector returns a StateDetector adapter for the pod's terminal.
// Returns nil if the virtual terminal is not available.
// Uses MultiSignalDetector for more reliable, Agent-agnostic state detection.
// Returns the same instance across multiple calls to ensure state continuity.
func (c *PodControllerImpl) GetStateDetector() autopilot.StateDetector {
	if c.pod.VirtualTerminal == nil {
		return nil
	}
	detector := c.pod.GetOrCreateStateDetector()
	if detector == nil {
		return nil
	}
	return detector
}

// multiSignalDetectorAdapter adapts terminal.MultiSignalDetector to autopilot.StateDetector interface.
// It uses multi-signal fusion (output activity + screen stability + prompt detection) for more
// reliable state detection that doesn't depend on specific Agent implementations.
type multiSignalDetectorAdapter struct {
	detector *terminal.MultiSignalDetector
	vt       *terminal.VirtualTerminal
	ctx      context.Context
	cancel   context.CancelFunc
}

// newMultiSignalDetectorAdapter creates a new adapter for multi-signal state detection.
// It starts a background goroutine to periodically update screen state and detect state changes.
func newMultiSignalDetectorAdapter(vt *terminal.VirtualTerminal) *multiSignalDetectorAdapter {
	detector := terminal.NewMultiSignalDetector(terminal.MultiSignalConfig{
		// Shorter thresholds for more responsive detection
		IdleThreshold:    500 * time.Millisecond,
		ConfirmThreshold: 500 * time.Millisecond,
		MinStableTime:    300 * time.Millisecond,
		WaitingThreshold: 0.6,
	})

	ctx, cancel := context.WithCancel(context.Background())
	adapter := &multiSignalDetectorAdapter{
		detector: detector,
		vt:       vt,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Start background detection loop
	go adapter.runDetectionLoop()

	return adapter
}

// runDetectionLoop periodically runs state detection.
// NOTE: Screen content is now pushed via OnScreenUpdate from OutputHandler (single data flow).
// This loop only triggers periodic DetectState() for timeout-based transitions.
func (a *multiSignalDetectorAdapter) runDetectionLoop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			// Only run detection - screen updates come from OutputHandler via OnScreenUpdate
			// This avoids reverse data flow and lock contention with VirtualTerminal.Feed
			a.detector.DetectState()
		}
	}
}

// OnScreenUpdate should be called after VirtualTerminal.Feed with the current screen lines.
// This implements single-direction data flow: PTY → Feed → OnScreenUpdate → StateDetector
func (a *multiSignalDetectorAdapter) OnScreenUpdate(lines []string) {
	a.detector.OnScreenUpdate(lines)
}

// OnOutput should be called when terminal output is received.
// This feeds the output activity detector signal.
func (a *multiSignalDetectorAdapter) OnOutput(bytes int) {
	logger.TerminalTrace().Trace("multiSignalDetectorAdapter.OnOutput ENTER", "bytes", bytes)
	a.detector.OnOutput(bytes)
	logger.TerminalTrace().Trace("multiSignalDetectorAdapter.OnOutput EXIT", "bytes", bytes)
}

// OnOSCTitle should be called when an OSC title update is received.
// This provides an optional hint signal.
func (a *multiSignalDetectorAdapter) OnOSCTitle(title string) {
	a.detector.OnOSCTitle(title)
}

// DetectState analyzes and returns the current agent state.
func (a *multiSignalDetectorAdapter) DetectState() autopilot.AgentState {
	state := a.detector.DetectState()
	return convertTerminalState(state)
}

// GetState returns the current state without performing detection.
func (a *multiSignalDetectorAdapter) GetState() autopilot.AgentState {
	state := a.detector.GetState()
	return convertTerminalState(state)
}

// SetCallback sets the state change callback.
func (a *multiSignalDetectorAdapter) SetCallback(cb autopilot.StateChangeCallback) {
	if cb == nil {
		a.detector.SetCallback(nil)
		return
	}

	// Wrap the autopilot callback to convert terminal states
	a.detector.SetCallback(func(newState, prevState terminal.AgentState) {
		cb(convertTerminalState(newState), convertTerminalState(prevState))
	})
}

// Reset resets the detector state.
func (a *multiSignalDetectorAdapter) Reset() {
	a.detector.Reset()
}

// Stop stops the background detection loop.
func (a *multiSignalDetectorAdapter) Stop() {
	a.cancel()
}

// convertTerminalState converts terminal.AgentState to autopilot.AgentState.
func convertTerminalState(state terminal.AgentState) autopilot.AgentState {
	switch state {
	case terminal.StateNotRunning:
		return autopilot.StateNotRunning
	case terminal.StateExecuting:
		return autopilot.StateExecuting
	case terminal.StateWaiting:
		return autopilot.StateWaiting
	default:
		return autopilot.StateNotRunning
	}
}
