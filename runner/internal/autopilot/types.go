// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"context"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
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
// All methods are mode-agnostic — they work identically for PTY and ACP pods
// because PodControllerImpl delegates to PodIO.
type TargetPodController interface {
	// SendInput sends text to the pod.
	SendInput(text string) error
	// GetWorkDir returns the pod's working directory.
	GetWorkDir() string
	// GetPodKey returns the pod's key.
	GetPodKey() string
	// GetAgentStatus returns the pod's agent status (executing/waiting/idle).
	GetAgentStatus() string
	// SubscribeStateChange registers a callback for agent state changes.
	// The callback receives the new status string ("executing", "waiting", "idle").
	SubscribeStateChange(id string, cb func(newStatus string))
	// UnsubscribeStateChange removes a state change subscription.
	UnsubscribeStateChange(id string)
}

// ControlProcess executes the control agent to make decisions.
// Two implementations:
//   - ExecControlProcess: launches a new CLI process per iteration (os/exec)
//   - AcpControlProcess: maintains a long-lived ACP session (stream-json)
type ControlProcess interface {
	// RunControlProcess executes a single decision cycle.
	RunControlProcess(ctx context.Context, iteration int) (*ControlDecision, error)
	// SetSessionID sets the session ID for resumption (exec mode).
	SetSessionID(id string)
	// GetSessionID returns the current session ID.
	GetSessionID() string
	// Stop gracefully shuts down the control process.
	Stop()
}
