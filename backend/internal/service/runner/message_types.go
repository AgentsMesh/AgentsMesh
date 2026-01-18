package runner

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
)

// ========== Message Types ==========

// Runner message types
const (
	// ==================== 初始化流程 (三阶段握手) ====================
	// Runner -> Backend: 初始化请求
	MsgTypeInitialize = "initialize"
	// Backend -> Runner: 初始化响应
	MsgTypeInitializeResult = "initialize_result"
	// Runner -> Backend: 初始化完成确认
	MsgTypeInitialized = "initialized"

	// ==================== 运行时消息: Runner -> Backend ====================
	MsgTypeHeartbeat      = "heartbeat"
	MsgTypePodCreated     = "pod_created"
	MsgTypePodTerminated  = "pod_terminated"
	MsgTypeTerminalOutput = "terminal_output"
	MsgTypeAgentStatus    = "agent_status"
	MsgTypePtyResized     = "pty_resized"
	MsgTypeError          = "error"

	// ==================== 运行时消息: Backend -> Runner ====================
	MsgTypeCreatePod      = "create_pod"
	MsgTypeTerminatePod   = "terminate_pod"
	MsgTypeTerminalInput  = "terminal_input"
	MsgTypeTerminalResize = "terminal_resize"
	MsgTypeSendPrompt     = "send_prompt"
)

// ========== 基础消息结构 ==========

// RunnerMessage represents a message from/to a runner
type RunnerMessage struct {
	Type      string          `json:"type"`
	PodKey    string          `json:"pod_key,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// ========== 初始化流程数据结构 ==========

// InitializeParams 是 Runner 发送的初始化参数
type InitializeParams struct {
	ProtocolVersion int        `json:"protocol_version"`
	RunnerInfo      RunnerInfo `json:"runner_info"`
}

// RunnerInfo 描述 Runner 的基本信息
type RunnerInfo struct {
	Version  string `json:"version"`
	NodeID   string `json:"node_id"`
	MCPPort  int    `json:"mcp_port"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
}

// InitializeResult 是 Backend 返回的初始化结果
type InitializeResult struct {
	ProtocolVersion int             `json:"protocol_version"`
	ServerInfo      ServerInfo      `json:"server_info"`
	AgentTypes      []AgentTypeInfo `json:"agent_types"`
	Features        []string        `json:"features"`
}

// ServerInfo 描述服务端的基本信息
type ServerInfo struct {
	Version string `json:"version"`
}

// AgentTypeInfo 描述单个 Agent 类型的信息
// 用于 Runner 检查本地是否可用
// Type alias to interfaces.AgentTypeInfo for backward compatibility
type AgentTypeInfo = interfaces.AgentTypeInfo

// InitializedParams 是 Runner 发送的初始化完成通知
type InitializedParams struct {
	AvailableAgents []string `json:"available_agents"`
}

// ========== Pod 操作请求结构 ==========
// Note: Pod event data structures (HeartbeatData, PodCreatedData, etc.) have been
// replaced by Proto types in runnerv1 package for zero-copy message passing.

// FileToCreate represents a file to be created in the sandbox
type FileToCreate struct {
	PathTemplate string `json:"path_template"`
	Content      string `json:"content"`
	Mode         int    `json:"mode,omitempty"`
	IsDirectory  bool   `json:"is_directory,omitempty"`
}

// WorkDirConfig represents the working directory configuration
type WorkDirConfig struct {
	Type          string `json:"type"` // "worktree", "tempdir", "local"
	RepositoryURL string `json:"repository_url,omitempty"`
	Branch        string `json:"branch,omitempty"`
	TicketID      string `json:"ticket_id,omitempty"`
	GitToken      string `json:"git_token,omitempty"`
	SSHKeyPath    string `json:"ssh_key_path,omitempty"`
	LocalPath     string `json:"local_path,omitempty"`
}

// CreatePodRequest represents a request to create a pod
// Backend computes all config, Runner just executes
type CreatePodRequest struct {
	PodKey        string            `json:"pod_key"`
	LaunchCommand string            `json:"launch_command"`
	LaunchArgs    []string          `json:"launch_args,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	FilesToCreate []FileToCreate    `json:"files_to_create,omitempty"`
	WorkDirConfig *WorkDirConfig    `json:"work_dir_config,omitempty"`
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

// ========== 协议版本和特性 ==========

const (
	// CurrentProtocolVersion 当前协议版本
	CurrentProtocolVersion = 2

	// MinSupportedProtocolVersion 最低支持的协议版本
	MinSupportedProtocolVersion = 2

	// 协议特性标识
	FeatureFilesToCreate = "files_to_create"
	FeatureWorkDirConfig = "work_dir_config"
)

// SupportedFeatures 返回当前 Backend 支持的特性列表
func SupportedFeatures() []string {
	return []string{
		FeatureFilesToCreate,
		FeatureWorkDirConfig,
	}
}
