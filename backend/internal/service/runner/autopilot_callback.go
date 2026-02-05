package runner

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// AutopilotController event callbacks

// onAutopilotStatusChange is the callback for AutopilotController status changes to notify realtime
var onAutopilotStatusChange func(
	autopilotControllerKey string,
	podKey string,
	phase string,
	iteration int32,
	maxIterations int32,
	circuitBreakerState string,
	circuitBreakerReason string,
	userTakeover bool,
)

// onAutopilotIteration is the callback for AutopilotController iteration events to notify realtime
var onAutopilotIterationChange func(
	autopilotControllerKey string,
	iteration int32,
	phase string,
	summary string,
	filesChanged []string,
	durationMs int64,
)

// onAutopilotThinkingChange is the callback for AutopilotController thinking events to notify realtime
var onAutopilotThinkingChange func(runnerID int64, data *runnerv1.AutopilotThinkingEvent)

// SetAutopilotStatusChangeCallback sets the callback for AutopilotController status changes
func (pc *PodCoordinator) SetAutopilotStatusChangeCallback(fn func(
	autopilotControllerKey string,
	podKey string,
	phase string,
	iteration int32,
	maxIterations int32,
	circuitBreakerState string,
	circuitBreakerReason string,
	userTakeover bool,
)) {
	onAutopilotStatusChange = fn
}

// SetAutopilotIterationChangeCallback sets the callback for AutopilotController iteration events
func (pc *PodCoordinator) SetAutopilotIterationChangeCallback(fn func(
	autopilotControllerKey string,
	iteration int32,
	phase string,
	summary string,
	filesChanged []string,
	durationMs int64,
)) {
	onAutopilotIterationChange = fn
}

// SetAutopilotThinkingCallback sets the callback for AutopilotController thinking events
func (pc *PodCoordinator) SetAutopilotThinkingCallback(fn func(runnerID int64, data *runnerv1.AutopilotThinkingEvent)) {
	onAutopilotThinkingChange = fn
}
