// Package client provides communication with AgentsMesh server.
package client

import (
	"encoding/json"
)

// MessageType defines the type of control message.
type MessageType string

const (
	// Client -> Server
	MsgTypeHeartbeat      MessageType = "heartbeat"
	MsgTypePodCreated     MessageType = "pod_created"
	MsgTypePodTerminated  MessageType = "pod_terminated"
	MsgTypeStatusChange   MessageType = "status_change"
	MsgTypePodList        MessageType = "pod_list"
	MsgTypeTerminalOutput MessageType = "terminal_output" // PTY output from runner
	MsgTypePtyResized     MessageType = "pty_resized"     // PTY size changed

	// Server -> Client
	MsgTypeCreatePod     MessageType = "create_pod"
	MsgTypeTerminatePod  MessageType = "terminate_pod"
	MsgTypeListPods      MessageType = "list_pods"
	MsgTypeTerminalInput MessageType = "terminal_input"  // User input to PTY
	MsgTypeTerminalResize MessageType = "terminal_resize" // Terminal resize
)

// ProtocolMessage is the base message structure for the new protocol.
// Matches backend's RunnerMessage struct for compatibility.
type ProtocolMessage struct {
	Type      MessageType     `json:"type"`
	PodKey    string          `json:"pod_key,omitempty"`
	Timestamp int64           `json:"timestamp"`         // Unix milliseconds to match backend
	Data      json.RawMessage `json:"data,omitempty"`
}

// HeartbeatData contains heartbeat information.
type HeartbeatData struct {
	NodeID        string               `json:"node_id"`
	Pods          []PodInfo            `json:"pods"`
	RunnerVersion string               `json:"runner_version,omitempty"`
	Capabilities  []PluginCapability   `json:"capabilities,omitempty"`
}

// PluginCapability represents a plugin's capability for server reporting.
type PluginCapability struct {
	Name            string    `json:"name"`
	Version         string    `json:"version"`
	Description     string    `json:"description"`
	SupportedAgents []string  `json:"supported_agents"`
	Executable      string    `json:"executable,omitempty"` // Required CLI command (if any)
	Available       bool      `json:"available"`            // Whether the executable is available on this system
	UI              *UIConfig `json:"ui,omitempty"`
}

// UIConfig represents the UI configuration for a plugin.
type UIConfig struct {
	Configurable bool      `json:"configurable"`
	Fields       []UIField `json:"fields"`
}

// UIField represents a single UI field configuration.
type UIField struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // boolean, string, select, number, secret
	Label       string      `json:"label"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
	Placeholder string      `json:"placeholder,omitempty"`
	Options     []UIOption  `json:"options,omitempty"`
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
	Required    bool        `json:"required,omitempty"`
}

// UIOption represents an option for select fields.
type UIOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// PodInfo contains pod information for protocol messages.
type PodInfo struct {
	PodKey       string `json:"pod_key"`
	Status       string `json:"status"`
	ClaudeStatus string `json:"claude_status"`
	Pid          int    `json:"pid"`
	ClientCount  int    `json:"client_count"`
}

// PreparationConfig contains workspace preparation configuration.
type PreparationConfig struct {
	Script         string `json:"script,omitempty"`          // Shell script to execute
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"` // Script execution timeout (default: 300)
}

// CreatePodRequest contains pod creation request data.
type CreatePodRequest struct {
	PodKey            string             `json:"pod_key"`
	InitialCommand    string             `json:"initial_command,omitempty"`
	InitialPrompt     string             `json:"initial_prompt,omitempty"`     // Prompt to send after command starts (for interactive mode)
	PermissionMode    string             `json:"permission_mode,omitempty"`    // Permission mode (plan/default/etc). If "plan", will send Shift+Tab to enter Plan Mode
	WorkingDir        string             `json:"working_dir,omitempty"`        // Deprecated: use PluginConfig
	TicketIdentifier  string             `json:"ticket_identifier,omitempty"`  // Deprecated: use PluginConfig
	WorktreeSuffix    string             `json:"worktree_suffix,omitempty"`    // Suffix for worktree path to support multiple instances per ticket
	EnvVars           map[string]string  `json:"env_vars,omitempty"`           // Deprecated: use PluginConfig
	PreparationConfig *PreparationConfig `json:"preparation_config,omitempty"` // Deprecated: use PluginConfig

	// PluginConfig is the unified configuration passed to Sandbox plugins
	// Fields: repository_url, branch, ticket_identifier, git_token, init_script, init_timeout, env_vars
	PluginConfig map[string]interface{} `json:"plugin_config,omitempty"`
}

// TerminatePodRequest contains pod termination request data.
type TerminatePodRequest struct {
	PodKey string `json:"pod_key"`
}

// PodCreatedEvent is sent when a pod is created.
type PodCreatedEvent struct {
	PodKey       string `json:"pod_key"`
	Pid          int    `json:"pid"`
	WorktreePath string `json:"worktree_path,omitempty"` // Worktree path if created
	BranchName   string `json:"branch_name,omitempty"`   // Branch name if worktree created
	PtyCols      uint16 `json:"pty_cols"`                // PTY width in columns
	PtyRows      uint16 `json:"pty_rows"`                // PTY height in rows
}

// PodTerminatedEvent is sent when a pod is terminated.
type PodTerminatedEvent struct {
	PodKey string `json:"pod_key"`
}

// StatusChangeEvent is sent when claude status changes.
type StatusChangeEvent struct {
	PodKey       string `json:"pod_key"`
	ClaudeStatus string `json:"claude_status"`
	ClaudePid    int    `json:"claude_pid,omitempty"`
}

// TerminalOutputEvent is sent when there's PTY output.
type TerminalOutputEvent struct {
	PodKey string `json:"pod_key"`
	Data   string `json:"data"` // Base64 encoded binary data
}

// TerminalInputRequest is sent to write to PTY.
type TerminalInputRequest struct {
	PodKey string `json:"pod_key"`
	Data   string `json:"data"` // Base64 encoded binary data
}

// TerminalResizeRequest is sent to resize PTY.
type TerminalResizeRequest struct {
	PodKey string `json:"pod_key"`
	Cols   uint16 `json:"cols"`
	Rows   uint16 `json:"rows"`
}

// PtyResizedEvent is sent when PTY size changes.
type PtyResizedEvent struct {
	PodKey string `json:"pod_key"`
	Cols   uint16 `json:"cols"`
	Rows   uint16 `json:"rows"`
}

// MessageHandler handles incoming messages from server.
type MessageHandler interface {
	OnCreatePod(req CreatePodRequest) error
	OnTerminatePod(req TerminatePodRequest) error
	OnListPods() []PodInfo
	OnTerminalInput(req TerminalInputRequest) error
	OnTerminalResize(req TerminalResizeRequest) error
	// GetCapabilities returns plugin capabilities for heartbeat reporting.
	// Can return nil if no capabilities are available.
	GetCapabilities() []PluginCapability
}
