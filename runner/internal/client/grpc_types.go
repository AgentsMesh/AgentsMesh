// Package client provides gRPC types for Runner communication.
// These types mirror the protobuf definitions in proto/runner/v1/runner.proto
// and will be replaced by generated code once protoc is available.
package client

// Note: This file provides temporary type definitions that match the proto schema.
// Once proto generation is set up, this file should be removed and imports
// updated to use the generated types from proto/gen/go/runner/v1.

// ==================== gRPC Message Types ====================
// These types are used for gRPC communication between Runner and Backend.
// They mirror the protobuf definitions but use Go-native types for easier integration.

// GRPCRunnerInfo represents Runner information sent during initialization.
type GRPCRunnerInfo struct {
	Version  string `json:"version"`
	NodeID   string `json:"node_id"`
	MCPPort  int32  `json:"mcp_port"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
}

// GRPCInitializeRequest represents the initialization request from Runner.
type GRPCInitializeRequest struct {
	ProtocolVersion int32           `json:"protocol_version"`
	RunnerInfo      *GRPCRunnerInfo `json:"runner_info"`
}

// GRPCInitializedConfirm represents the confirmation after initialization.
type GRPCInitializedConfirm struct {
	AvailableAgents []string `json:"available_agents"`
}

// GRPCHeartbeatData represents heartbeat data sent by Runner.
type GRPCHeartbeatData struct {
	NodeID string         `json:"node_id"`
	Pods   []*GRPCPodInfo `json:"pods"`
}

// GRPCPodInfo represents Pod status information.
type GRPCPodInfo struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status"`
	AgentStatus string `json:"agent_status"`
}

// GRPCPodCreatedEvent represents a Pod creation event.
type GRPCPodCreatedEvent struct {
	PodKey string `json:"pod_key"`
	Pid    int32  `json:"pid"`
}

// GRPCPodTerminatedEvent represents a Pod termination event.
type GRPCPodTerminatedEvent struct {
	PodKey       string `json:"pod_key"`
	ExitCode     int32  `json:"exit_code"`
	ErrorMessage string `json:"error_message"`
}

// GRPCTerminalOutputEvent represents terminal output from a Pod.
type GRPCTerminalOutputEvent struct {
	PodKey string `json:"pod_key"`
	Data   []byte `json:"data"` // Binary data, no base64 encoding needed with protobuf
}

// GRPCAgentStatusEvent represents an agent status change.
type GRPCAgentStatusEvent struct {
	PodKey string `json:"pod_key"`
	Status string `json:"status"`
}

// GRPCPtyResizedEvent represents a PTY resize event.
type GRPCPtyResizedEvent struct {
	PodKey string `json:"pod_key"`
	Cols   int32  `json:"cols"`
	Rows   int32  `json:"rows"`
}

// GRPCErrorEvent represents an error event.
type GRPCErrorEvent struct {
	PodKey  string            `json:"pod_key"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

// ==================== Server -> Runner Messages ====================

// GRPCServerInfo represents server information.
type GRPCServerInfo struct {
	Version string `json:"version"`
}

// GRPCAgentTypeInfo represents agent type information.
type GRPCAgentTypeInfo struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	DefaultArgs []string `json:"default_args"`
}

// GRPCInitializeResult represents the initialization result from server.
type GRPCInitializeResult struct {
	ProtocolVersion int32                `json:"protocol_version"`
	ServerInfo      *GRPCServerInfo      `json:"server_info"`
	AgentTypes      []*GRPCAgentTypeInfo `json:"agent_types"`
	Features        []string             `json:"features"`
}

// GRPCFileToCreate represents a file to create in the sandbox.
type GRPCFileToCreate struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    int32  `json:"mode"`
}

// GRPCWorkDirConfig represents working directory configuration.
type GRPCWorkDirConfig struct {
	Type       string `json:"type"` // "worktree", "tempdir", "path"
	RepoPath   string `json:"repo_path"`
	BranchName string `json:"branch_name"`
	BaseBranch string `json:"base_branch"`
	Path       string `json:"path"`
}

// GRPCCreatePodCommand represents a create pod command from server.
type GRPCCreatePodCommand struct {
	PodKey        string              `json:"pod_key"`
	LaunchCommand string              `json:"launch_command"`
	LaunchArgs    []string            `json:"launch_args"`
	EnvVars       map[string]string   `json:"env_vars"`
	FilesToCreate []*GRPCFileToCreate `json:"files_to_create"`
	WorkDirConfig *GRPCWorkDirConfig  `json:"work_dir_config"`
}

// GRPCTerminatePodCommand represents a terminate pod command.
type GRPCTerminatePodCommand struct {
	PodKey string `json:"pod_key"`
	Force  bool   `json:"force"`
}

// GRPCTerminalInputCommand represents terminal input from server.
type GRPCTerminalInputCommand struct {
	PodKey string `json:"pod_key"`
	Data   []byte `json:"data"`
}

// GRPCTerminalResizeCommand represents a terminal resize command.
type GRPCTerminalResizeCommand struct {
	PodKey string `json:"pod_key"`
	Cols   int32  `json:"cols"`
	Rows   int32  `json:"rows"`
}

// GRPCSendPromptCommand represents a send prompt command.
type GRPCSendPromptCommand struct {
	PodKey string `json:"pod_key"`
	Prompt string `json:"prompt"`
}
