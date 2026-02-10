// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// Phase represents the current phase of an AutopilotController.
type Phase string

const (
	PhaseInitializing    Phase = "initializing"
	PhaseRunning         Phase = "running"
	PhasePaused          Phase = "paused"
	PhaseUserTakeover    Phase = "user_takeover"
	PhaseWaitingApproval Phase = "waiting_approval"
	PhaseCompleted       Phase = "completed"
	PhaseFailed          Phase = "failed"
	PhaseStopped         Phase = "stopped"
	PhaseMaxIterations   Phase = "max_iterations"
)

// Status represents the current status of an AutopilotController.
type Status struct {
	Phase            Phase
	CurrentIteration int
	MaxIterations    int
	PodStatus        string
	StartedAt        time.Time
	LastIterationAt  time.Time
	LastDecision     string // Last Control decision type
	LastDecisionMsg  string // Last Control decision message
}

// EventReporter is the interface for reporting Autopilot events.
type EventReporter interface {
	ReportAutopilotStatus(event *runnerv1.AutopilotStatusEvent)
	ReportAutopilotIteration(event *runnerv1.AutopilotIterationEvent)
	ReportAutopilotCreated(event *runnerv1.AutopilotCreatedEvent)
	ReportAutopilotTerminated(event *runnerv1.AutopilotTerminatedEvent)
	ReportAutopilotThinking(event *runnerv1.AutopilotThinkingEvent)
}

// TargetPodController provides methods to interact with the controlled Pod.
// This interface is implemented by runner.PodControllerImpl.
type TargetPodController interface {
	// SendTerminalText sends text to the pod's terminal.
	SendTerminalText(text string) error
	// GetWorkDir returns the pod's working directory.
	GetWorkDir() string
	// GetPodKey returns the pod's key.
	GetPodKey() string
	// GetAgentStatus returns the pod's agent status (executing/waiting/idle).
	GetAgentStatus() string
	// GetStateDetector returns a StateDetector for the pod.
	// Returns nil if state detection is not available.
	// The StateDetector interface is defined in terminal/detector package,
	// which is a foundational service independent of Autopilot.
	GetStateDetector() detector.StateDetector
}
