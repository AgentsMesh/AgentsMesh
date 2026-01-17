// Package client provides communication with AgentsMesh server via gRPC.
package client

// MessageType defines the type of message (used for mock testing).
type MessageType string

const (
	// Event types (Runner -> Backend)
	MsgTypePodCreated     MessageType = "pod_created"
	MsgTypePodTerminated  MessageType = "pod_terminated"
	MsgTypeTerminalOutput MessageType = "terminal_output"
	MsgTypePtyResized     MessageType = "pty_resized"
)

// ==================== Pod 操作数据结构 ====================

// FileToCreate represents a file to be created in the sandbox.
type FileToCreate struct {
	PathTemplate string `json:"path_template"`
	Content      string `json:"content"`
	Mode         int    `json:"mode,omitempty"`
	IsDirectory  bool   `json:"is_directory,omitempty"`
}

// WorkDirConfig represents the working directory configuration.
type WorkDirConfig struct {
	Type          string `json:"type"` // "worktree", "tempdir", "local"
	RepositoryURL string `json:"repository_url,omitempty"`
	Branch        string `json:"branch,omitempty"`
	TicketID      string `json:"ticket_id,omitempty"`
	GitToken      string `json:"git_token,omitempty"`
	SSHKeyPath    string `json:"ssh_key_path,omitempty"`
	LocalPath     string `json:"local_path,omitempty"`
}

// CreatePodRequest contains pod creation request data.
// Backend computes all config, Runner just executes.
type CreatePodRequest struct {
	PodKey        string            `json:"pod_key"`
	LaunchCommand string            `json:"launch_command"`
	LaunchArgs    []string          `json:"launch_args,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	FilesToCreate []FileToCreate    `json:"files_to_create,omitempty"`
	WorkDirConfig *WorkDirConfig    `json:"work_dir_config,omitempty"`
	InitialPrompt string            `json:"initial_prompt,omitempty"`
}

// TerminatePodRequest contains pod termination request data.
type TerminatePodRequest struct {
	PodKey string `json:"pod_key"`
}

// PodInfo contains pod information for heartbeat messages.
type PodInfo struct {
	PodKey       string `json:"pod_key"`
	Status       string `json:"status"`
	ClaudeStatus string `json:"claude_status"`
	Pid          int    `json:"pid"`
}

// ==================== 终端数据结构 ====================

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

// ==================== 消息处理接口 ====================

// MessageHandler handles incoming messages from server.
type MessageHandler interface {
	OnCreatePod(req CreatePodRequest) error
	OnTerminatePod(req TerminatePodRequest) error
	OnListPods() []PodInfo
	OnTerminalInput(req TerminalInputRequest) error
	OnTerminalResize(req TerminalResizeRequest) error
}
