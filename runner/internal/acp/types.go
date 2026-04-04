package acp

// ACP client states.
const (
	StateUninitialized     = "uninitialized"
	StateInitializing      = "initializing"
	StateIdle              = "idle"
	StateProcessing        = "processing"
	StateWaitingPermission = "waiting_permission"
	StateStopped           = "stopped"
)

// ContentChunk represents a streamed content fragment from the agent.
type ContentChunk struct {
	Text string `json:"text"`
	Role string `json:"role"` // "assistant" | "user"
}

// ToolCallUpdate represents a tool execution status change.
type ToolCallUpdate struct {
	ToolCallID    string `json:"toolCallId"`
	ToolName      string `json:"toolName"`
	Status        string `json:"status"` // "running" | "completed" | "failed"
	ArgumentsJSON string `json:"argumentsJson"`
}

// ToolCallResult represents the outcome of a tool execution.
type ToolCallResult struct {
	ToolCallID   string `json:"toolCallId"`
	ToolName     string `json:"toolName"`
	Success      bool   `json:"success"`
	ResultText   string `json:"resultText"`
	ErrorMessage string `json:"errorMessage"`
}

// PlanStep represents a single step in the agent's plan.
type PlanStep struct {
	Title  string `json:"title"`
	Status string `json:"status"` // "pending" | "in_progress" | "completed"
}

// PlanUpdate represents the agent's current execution plan.
type PlanUpdate struct {
	Steps []PlanStep `json:"steps"`
}

// ThinkingUpdate represents the agent's internal reasoning.
type ThinkingUpdate struct {
	Text string `json:"text"`
}

// PermissionRequest represents a tool permission request from the agent.
type PermissionRequest struct {
	SessionID     string `json:"sessionId"`
	RequestID     string `json:"requestId"`
	ToolName      string `json:"toolName"`
	ArgumentsJSON string `json:"argumentsJson"`
	Description   string `json:"description"`
}

// EventCallbacks defines the event handlers for ACP client events.
type EventCallbacks struct {
	OnContentChunk      func(sessionID string, chunk ContentChunk)
	OnToolCallUpdate    func(sessionID string, update ToolCallUpdate)
	OnToolCallResult    func(sessionID string, result ToolCallResult)
	OnPlanUpdate        func(sessionID string, update PlanUpdate)
	OnThinkingUpdate    func(sessionID string, update ThinkingUpdate)
	OnPermissionRequest func(req PermissionRequest)
	OnStateChange       func(newState string)
	OnLog               func(level, message string)
	OnExit              func(exitCode int)
}

// AcpSessionSnapshot captures the current state of an ACP session
// for sending to late-joining subscribers via Relay.
type AcpSessionSnapshot struct {
	SessionID          string              `json:"sessionId"`
	State              string              `json:"state"`
	Messages           []ContentChunk      `json:"messages"`
	ToolCalls          []ToolCallSnapshot  `json:"toolCalls,omitempty"`
	Plan               []PlanStep          `json:"plan,omitempty"`
	PendingPermissions []PermissionRequest `json:"pendingPermissions"`
}

// ToolCallSnapshot is the merged view of a tool call for snapshots,
// combining ToolCallUpdate fields with ToolCallResult fields.
type ToolCallSnapshot struct {
	ToolCallID    string `json:"toolCallId"`
	ToolName      string `json:"toolName"`
	Status        string `json:"status"`
	ArgumentsJSON string `json:"argumentsJson"`
	Success       *bool  `json:"success,omitempty"` // nil until result arrives
	ResultText    string `json:"resultText,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

// AgentCapabilities describes the capabilities reported by the agent
// during the initialize handshake.
type AgentCapabilities struct {
	Streaming   bool
	Permissions bool
	MCPServers  bool
}
