package agentpod

import (
	"time"
)

// AutopilotController phase constants
const (
	AutopilotPhaseInitializing    = "initializing"
	AutopilotPhaseRunning         = "running"
	AutopilotPhasePaused          = "paused"
	AutopilotPhaseUserTakeover    = "user_takeover"
	AutopilotPhaseWaitingApproval = "waiting_approval"
	AutopilotPhaseMaxIterations   = "max_iterations"
	AutopilotPhaseCompleted       = "completed"
	AutopilotPhaseFailed          = "failed"
	AutopilotPhaseStopped         = "stopped"
)

// Circuit breaker state constants
const (
	CircuitBreakerClosed   = "closed"
	CircuitBreakerHalfOpen = "half_open"
	CircuitBreakerOpen     = "open"
)

// Default configuration values for AutopilotController.
// Centralized here so that all callers (REST handler, LoopOrchestrator, etc.)
// share the same defaults instead of duplicating magic numbers.
const (
	DefaultMaxIterations       int32 = 10
	DefaultIterationTimeoutSec int32 = 300
	DefaultNoProgressThreshold int32 = 3
	DefaultSameErrorThreshold  int32 = 5
	DefaultApprovalTimeoutMin  int32 = 30
)

// IsAutopilotPhaseTerminal returns true if the given phase string represents a terminal state.
func IsAutopilotPhaseTerminal(phase string) bool {
	return phase == AutopilotPhaseCompleted ||
		phase == AutopilotPhaseFailed ||
		phase == AutopilotPhaseStopped ||
		phase == AutopilotPhaseMaxIterations
}

// IsAutopilotPhaseActive returns true if the given phase string represents an active state.
func IsAutopilotPhaseActive(phase string) bool {
	return phase == AutopilotPhaseRunning ||
		phase == AutopilotPhaseInitializing ||
		phase == AutopilotPhasePaused ||
		phase == AutopilotPhaseWaitingApproval
}

// TerminalPhases returns the list of terminal autopilot phases.
func TerminalPhases() []string {
	return []string{AutopilotPhaseCompleted, AutopilotPhaseFailed, AutopilotPhaseStopped, AutopilotPhaseMaxIterations}
}

// ApplyDefaults fills zero-valued configuration fields with domain defaults.
func ApplyDefaults(maxIter, iterTimeout, noProg, sameErr, approvalTimeout int32) (int32, int32, int32, int32, int32) {
	if maxIter == 0 {
		maxIter = DefaultMaxIterations
	}
	if iterTimeout == 0 {
		iterTimeout = DefaultIterationTimeoutSec
	}
	if noProg == 0 {
		noProg = DefaultNoProgressThreshold
	}
	if sameErr == 0 {
		sameErr = DefaultSameErrorThreshold
	}
	if approvalTimeout == 0 {
		approvalTimeout = DefaultApprovalTimeoutMin
	}
	return maxIter, iterTimeout, noProg, sameErr, approvalTimeout
}

// AutopilotController represents an event-driven automation controller for Pod
type AutopilotController struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	// Key identifiers
	AutopilotControllerKey string `gorm:"size:100;not null;uniqueIndex" json:"autopilot_controller_key"`
	PodKey                 string `gorm:"size:100;not null;index" json:"pod_key"`
	PodID                  int64  `gorm:"not null;index" json:"pod_id"`
	RunnerID               int64  `gorm:"not null;index" json:"runner_id"`

	// Task
	InitialPrompt string `gorm:"type:text" json:"initial_prompt,omitempty"`

	// Status
	Phase               string `gorm:"size:50;not null;default:'initializing'" json:"phase"`
	CurrentIteration    int32  `gorm:"not null;default:0" json:"current_iteration"`
	MaxIterations       int32  `gorm:"not null;default:10" json:"max_iterations"`
	IterationTimeoutSec int32  `gorm:"not null;default:300" json:"iteration_timeout_sec"`

	// Circuit breaker
	CircuitBreakerState  string  `gorm:"size:50;not null;default:'closed'" json:"circuit_breaker_state"`
	CircuitBreakerReason *string `gorm:"size:500" json:"circuit_breaker_reason,omitempty"`
	NoProgressThreshold  int32   `gorm:"not null;default:3" json:"no_progress_threshold"`
	SameErrorThreshold   int32   `gorm:"not null;default:5" json:"same_error_threshold"`
	ApprovalTimeoutMin   int32   `gorm:"not null;default:30" json:"approval_timeout_min"`

	// Control agent configuration
	ControlAgentSlug      *string `gorm:"size:50" json:"control_agent_slug,omitempty"` // default: claude
	ControlPromptTemplate *string `gorm:"type:text" json:"control_prompt_template,omitempty"`
	MCPConfigJSON         *string `gorm:"type:text" json:"mcp_config_json,omitempty"`

	// User takeover
	UserTakeover bool `gorm:"not null;default:false" json:"user_takeover"`

	// Timestamps
	StartedAt         *time.Time `json:"started_at,omitempty"`
	LastIterationAt   *time.Time `json:"last_iteration_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	ApprovalRequestAt *time.Time `json:"approval_request_at,omitempty"` // When circuit breaker triggered

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	Pod *Pod `gorm:"foreignKey:PodID" json:"pod,omitempty"`
}

func (AutopilotController) TableName() string {
	return "autopilot_controllers"
}

// IsActive returns true if AutopilotController is actively running
func (r *AutopilotController) IsActive() bool {
	return IsAutopilotPhaseActive(r.Phase)
}

// IsTerminal returns true if AutopilotController is in a terminal state
func (r *AutopilotController) IsTerminal() bool {
	return IsAutopilotPhaseTerminal(r.Phase)
}

// CanResume returns true if AutopilotController can be resumed
func (r *AutopilotController) CanResume() bool {
	return r.Phase == AutopilotPhasePaused ||
		r.Phase == AutopilotPhaseWaitingApproval
}
