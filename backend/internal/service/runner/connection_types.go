package runner

import (
	"encoding/json"
	"errors"
)

// ========== Errors ==========

var (
	ErrRunnerNotConnected = errors.New("runner not connected")
	ErrConnectionClosed   = errors.New("connection closed")
)

// ========== Message Types ==========

// Runner message types
const (
	// From runner
	MsgTypeHeartbeat      = "heartbeat"
	MsgTypePodCreated     = "pod_created"
	MsgTypePodTerminated  = "pod_terminated"
	MsgTypeTerminalOutput = "terminal_output"
	MsgTypeAgentStatus    = "agent_status"
	MsgTypePtyResized     = "pty_resized"
	MsgTypeError          = "error"

	// To runner
	MsgTypeCreatePod      = "create_pod"
	MsgTypeTerminatePod   = "terminate_pod"
	MsgTypeTerminalInput  = "terminal_input"
	MsgTypeTerminalResize = "terminal_resize"
	MsgTypeSendPrompt     = "send_prompt"
)

// ========== Message Structures ==========

// RunnerMessage represents a message from/to a runner
type RunnerMessage struct {
	Type      string          `json:"type"`
	PodKey    string          `json:"pod_key,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// HeartbeatData represents heartbeat message data
type HeartbeatData struct {
	Pods          []HeartbeatPod `json:"pods"`
	RunnerVersion string         `json:"runner_version,omitempty"`
}

// HeartbeatPod represents a pod in heartbeat data
type HeartbeatPod struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status,omitempty"`
	AgentStatus string `json:"agent_status,omitempty"`
}

// PodCreatedData represents pod creation event data
type PodCreatedData struct {
	PodKey       string `json:"pod_key"`
	Pid          int    `json:"pid"`
	BranchName   string `json:"branch_name,omitempty"`
	WorktreePath string `json:"worktree_path,omitempty"`
	Cols         int    `json:"cols,omitempty"`
	Rows         int    `json:"rows,omitempty"`
}

// PodTerminatedData represents pod termination event data
type PodTerminatedData struct {
	PodKey   string `json:"pod_key"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// TerminalOutputData represents terminal output data
type TerminalOutputData struct {
	PodKey string `json:"pod_key"`
	Data   []byte `json:"data"`
}

// AgentStatusData represents agent status change data
type AgentStatusData struct {
	PodKey string `json:"pod_key"`
	Status string `json:"status"`
	Pid    int    `json:"pid,omitempty"`
}

// PtyResizedData represents PTY resize event data
type PtyResizedData struct {
	PodKey string `json:"pod_key"`
	Cols   int    `json:"cols"`
	Rows   int    `json:"rows"`
}

// ========== Request Structures ==========

// PreparationConfig contains workspace preparation configuration
type PreparationConfig struct {
	Script         string `json:"script,omitempty"`          // Shell script to execute
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"` // Script execution timeout
}

// CreatePodRequest represents a request to create a pod
// Fields match Runner's client.CreatePodRequest
type CreatePodRequest struct {
	PodKey            string             `json:"pod_key"`
	InitialCommand    string             `json:"initial_command,omitempty"`    // Command to run (e.g., "claude")
	InitialPrompt     string             `json:"initial_prompt,omitempty"`     // Prompt to send after command starts
	PermissionMode    string             `json:"permission_mode,omitempty"`    // Permission mode (plan/default)
	WorkingDir        string             `json:"working_dir,omitempty"`        // Working directory (deprecated, use PluginConfig)
	TicketIdentifier  string             `json:"ticket_identifier,omitempty"`  // For worktree creation (deprecated, use PluginConfig)
	WorktreeSuffix    string             `json:"worktree_suffix,omitempty"`    // Suffix for multiple worktrees per ticket
	EnvVars           map[string]string  `json:"env_vars,omitempty"`           // Environment variables (deprecated, use PluginConfig)
	PreparationConfig *PreparationConfig `json:"preparation_config,omitempty"` // Workspace preparation config (deprecated, use PluginConfig)

	// PluginConfig is the unified configuration passed to Runner's Sandbox plugins
	// Contains: repository_url, branch, ticket_identifier, git_token, init_script, init_timeout, env_vars
	PluginConfig map[string]interface{} `json:"plugin_config,omitempty"`
}

// TerminalInputRequest represents terminal input to send
type TerminalInputRequest struct {
	PodKey string `json:"pod_key"`
	Data   []byte `json:"data"`
}

// TerminalResizeRequest represents terminal resize request
type TerminalResizeRequest struct {
	PodKey string `json:"pod_key"`
	Cols   int    `json:"cols"`
	Rows   int    `json:"rows"`
}
