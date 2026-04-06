package v1

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// CreateAutopilotControllerRequest represents the request to create a AutopilotController
type CreateAutopilotControllerRequest struct {
	PodKey string `json:"pod_key" binding:"required"`

	// Task
	Prompt string `json:"prompt,omitempty"`

	// Configuration (all optional with defaults)
	MaxIterations         int32  `json:"max_iterations,omitempty"`
	IterationTimeoutSec   int32  `json:"iteration_timeout_sec,omitempty"`
	NoProgressThreshold   int32  `json:"no_progress_threshold,omitempty"`
	SameErrorThreshold    int32  `json:"same_error_threshold,omitempty"`
	ApprovalTimeoutMin    int32  `json:"approval_timeout_min,omitempty"`
	ControlAgentSlug      string `json:"control_agent_slug,omitempty"`
	ControlPromptTemplate string `json:"control_prompt_template,omitempty"`
	MCPConfigJSON         string `json:"mcp_config_json,omitempty"`
}

// AutopilotControllerResponse represents the response for AutopilotController operations
type AutopilotControllerResponse struct {
	ID                     int64  `json:"id"`
	AutopilotControllerKey string `json:"autopilot_controller_key"`
	PodKey                 string `json:"pod_key"`
	Phase                  string `json:"phase"`
	CurrentIteration       int32  `json:"current_iteration"`
	MaxIterations          int32  `json:"max_iterations"`
	CircuitBreaker         struct {
		State  string `json:"state"`
		Reason string `json:"reason,omitempty"`
	} `json:"circuit_breaker"`
	UserTakeover    bool       `json:"user_takeover"`
	Prompt          string     `json:"prompt,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	LastIterationAt *time.Time `json:"last_iteration_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// toAutopilotControllerResponse converts domain model to API response
func toAutopilotControllerResponse(rp *agentpod.AutopilotController) *AutopilotControllerResponse {
	resp := &AutopilotControllerResponse{
		ID:                     rp.ID,
		AutopilotControllerKey: rp.AutopilotControllerKey,
		PodKey:                 rp.PodKey,
		Phase:                  rp.Phase,
		CurrentIteration:       rp.CurrentIteration,
		MaxIterations:          rp.MaxIterations,
		UserTakeover:           rp.UserTakeover,
		Prompt:          rp.Prompt,
		StartedAt:              rp.StartedAt,
		LastIterationAt:        rp.LastIterationAt,
		CreatedAt:              rp.CreatedAt,
	}
	resp.CircuitBreaker.State = rp.CircuitBreakerState
	if rp.CircuitBreakerReason != nil {
		resp.CircuitBreaker.Reason = *rp.CircuitBreakerReason
	}
	return resp
}
