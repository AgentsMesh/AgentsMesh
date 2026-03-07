package runner

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Autopilot callback type definitions

// AutopilotStatusChangeFunc is the callback type for AutopilotController status changes
type AutopilotStatusChangeFunc func(
	autopilotControllerKey string,
	podKey string,
	phase string,
	iteration int32,
	maxIterations int32,
	circuitBreakerState string,
	circuitBreakerReason string,
	userTakeover bool,
)

// AutopilotIterationChangeFunc is the callback type for AutopilotController iteration events
type AutopilotIterationChangeFunc func(
	autopilotControllerKey string,
	iteration int32,
	phase string,
	summary string,
	filesChanged []string,
	durationMs int64,
)

// AutopilotThinkingChangeFunc is the callback type for AutopilotController thinking events
type AutopilotThinkingChangeFunc func(runnerID int64, data *runnerv1.AutopilotThinkingEvent)

// SetAutopilotStatusChangeCallback sets the callback for AutopilotController status changes.
// Must be called during initialization before concurrent access.
func (pc *PodCoordinator) SetAutopilotStatusChangeCallback(fn AutopilotStatusChangeFunc) {
	pc.onAutopilotStatusChange = fn
}

// SetAutopilotIterationChangeCallback sets the callback for AutopilotController iteration events.
// Must be called during initialization before concurrent access.
func (pc *PodCoordinator) SetAutopilotIterationChangeCallback(fn AutopilotIterationChangeFunc) {
	pc.onAutopilotIterationChange = fn
}

// SetAutopilotThinkingCallback sets the callback for AutopilotController thinking events.
// Must be called during initialization before concurrent access.
func (pc *PodCoordinator) SetAutopilotThinkingCallback(fn AutopilotThinkingChangeFunc) {
	pc.onAutopilotThinkingChange = fn
}
