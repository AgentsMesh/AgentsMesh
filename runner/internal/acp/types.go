package acp

import "errors"

// Errors for control request operations.
var (
	ErrControlNotSupported = errors.New("transport does not support outgoing control requests")
	ErrControlTimeout      = errors.New("control request timed out waiting for response")
)

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

// Configuration captures the live control-plane state of an ACP session.
// Empty string means "unset / unknown" — consumers should treat absent fields
// as not-yet-known rather than defaulting to a hardcoded value.
type Configuration struct {
	PermissionMode string `json:"permissionMode,omitempty"`
	Model          string `json:"model,omitempty"`
}

// ConfigUpdate carries a delta of configuration fields. Empty fields mean
// "unchanged" — callers must merge into the existing Configuration, not replace.
type ConfigUpdate struct {
	PermissionMode string `json:"permissionMode,omitempty"`
	Model          string `json:"model,omitempty"`
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
	OnConfigChange      func(sessionID string, update ConfigUpdate)
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
	Thinkings          []ThinkingUpdate    `json:"thinkings,omitempty"`
	Logs               []LogEntry          `json:"logs,omitempty"`
	PendingPermissions []PermissionRequest `json:"pendingPermissions"`
	Configuration      Configuration       `json:"configuration"`
}

// LogEntry is a single log message captured in a snapshot.
type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
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
