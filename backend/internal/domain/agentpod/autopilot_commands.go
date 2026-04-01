package agentpod

// CreateAutopilotControllerCommand represents a command to create a AutopilotController
type CreateAutopilotControllerCommand struct {
	AutopilotControllerKey string `json:"autopilot_controller_key"`
	PodKey                 string `json:"pod_key,omitempty"`

	// Configuration
	InitialPrompt         string `json:"initial_prompt,omitempty"`
	MaxIterations         int32  `json:"max_iterations,omitempty"`
	IterationTimeoutSec   int32  `json:"iteration_timeout_sec,omitempty"`
	NoProgressThreshold   int32  `json:"no_progress_threshold,omitempty"`
	SameErrorThreshold    int32  `json:"same_error_threshold,omitempty"`
	ApprovalTimeoutMin    int32  `json:"approval_timeout_min,omitempty"`
	ControlAgentSlug      string `json:"control_agent_slug,omitempty"`
	ControlPromptTemplate string `json:"control_prompt_template,omitempty"`
	MCPConfigJSON         string `json:"mcp_config_json,omitempty"`
}

// AutopilotControlAction represents control action types
type AutopilotControlAction string

const (
	AutopilotControlPause    AutopilotControlAction = "pause"
	AutopilotControlResume   AutopilotControlAction = "resume"
	AutopilotControlStop     AutopilotControlAction = "stop"
	AutopilotControlApprove  AutopilotControlAction = "approve"
	AutopilotControlTakeover AutopilotControlAction = "takeover"
	AutopilotControlHandback AutopilotControlAction = "handback"
)

// AutopilotApproveOptions represents options for approval action
type AutopilotApproveOptions struct {
	ContinueExecution    bool  `json:"continue_execution"`
	AdditionalIterations int32 `json:"additional_iterations,omitempty"`
}
