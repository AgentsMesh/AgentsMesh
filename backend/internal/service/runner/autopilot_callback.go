package runner

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

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

type AutopilotIterationChangeFunc func(
	autopilotControllerKey string,
	iteration int32,
	phase string,
	summary string,
	filesChanged []string,
	durationMs int64,
)

type AutopilotThinkingChangeFunc func(runnerID int64, data *runnerv1.AutopilotThinkingEvent)

// SetAutopilotStatusChangeCallback is not safe under concurrent access — call only during init.
func (pc *PodCoordinator) SetAutopilotStatusChangeCallback(fn AutopilotStatusChangeFunc) {
	pc.onAutopilotStatusChange = fn
}

// SetAutopilotIterationChangeCallback is not safe under concurrent access — call only during init.
func (pc *PodCoordinator) SetAutopilotIterationChangeCallback(fn AutopilotIterationChangeFunc) {
	pc.onAutopilotIterationChange = fn
}

// SetAutopilotThinkingCallback is not safe under concurrent access — call only during init.
func (pc *PodCoordinator) SetAutopilotThinkingCallback(fn AutopilotThinkingChangeFunc) {
	pc.onAutopilotThinkingChange = fn
}
