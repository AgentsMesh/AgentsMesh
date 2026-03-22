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
	// NOTE: These constants are for the legacy JSON protocol (protocol_version=1).
	// Current implementation uses Proto (protocol_version=2), so these are rarely used.
	MsgTypeHeartbeat     = "heartbeat"
	MsgTypePodCreated    = "pod_created"
	MsgTypePodTerminated = "pod_terminated"
	MsgTypeAgentStatus   = "agent_status"
	MsgTypePodResized    = "pod_resized"
	MsgTypeError         = "error"
	// NOTE: terminal_output is NOT here - terminal output is streamed via Relay, not gRPC
	// NOTE: Backend→Runner commands (create_pod, terminate_pod, pod_input, send_prompt, etc.)
	// are sent directly as Proto messages via gRPC. No string-based message type constants needed.
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
	ProtocolVersion int                       `json:"protocol_version"`
	ServerInfo      ServerInfo                `json:"server_info"`
	AgentTypes      []interfaces.AgentTypeInfo `json:"agent_types"`
	Features        []string                  `json:"features"`
}

// ServerInfo 描述服务端的基本信息
type ServerInfo struct {
	Version string `json:"version"`
}

// InitializedParams 是 Runner 发送的初始化完成通知
type InitializedParams struct {
	AvailableAgents []string `json:"available_agents"`
}

// ========== Pod 操作请求结构 ==========
// Note: Pod command types (CreatePodCommand, FileToCreate, SandboxConfig) are now
// defined in Proto (runnerv1 package) for zero-copy message passing.
// Only pod-related request types remain here for internal use.

// PodInputRequest represents pod input to send
type PodInputRequest struct {
	PodKey string `json:"pod_key"`
	Data   []byte `json:"data"`
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
