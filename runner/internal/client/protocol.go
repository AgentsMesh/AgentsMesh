// Package client provides communication with AgentsMesh server via gRPC.
package client

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// MessageType defines the type of message (used for mock testing).
type MessageType string

const (
	// Event types (Runner -> Backend)
	MsgTypePodCreated    MessageType = "pod_created"
	MsgTypePodTerminated MessageType = "pod_terminated"
	MsgTypePodResized    MessageType = "pod_resized"
	// NOTE: MsgTypeTerminalOutput removed - output is exclusively streamed via Relay
)

// ==================== Pod Operation Data Structures ====================
// Note: Pod command types (CreatePodCommand, FileToCreate, SandboxConfig) are now
// defined in Proto (runnerv1 package) for zero-copy message passing.

// TerminatePodRequest contains pod termination request data.
type TerminatePodRequest struct {
	PodKey string `json:"pod_key"`
}

// PodInfo contains pod information for heartbeat messages.
type PodInfo struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status"`
	AgentStatus string `json:"agent_status"`
	Pid         int    `json:"pid"`
}

// RelayConnectionInfo contains relay connection information for heartbeat messages.
// Note: SessionID has been removed - channels are now identified by PodKey only
type RelayConnectionInfo struct {
	PodKey      string `json:"pod_key"`
	RelayURL    string `json:"relay_url"`
	Connected   bool   `json:"connected"`
	ConnectedAt int64  `json:"connected_at"` // Unix milliseconds
}

// ==================== Pod I/O Data Structures ====================

// PodInputRequest contains input data to write to pod stdin.
type PodInputRequest struct {
	PodKey string `json:"pod_key"`
	Data   []byte `json:"data"` // Binary data (gRPC uses native bytes, no base64 needed)
}

// SubscribePodRequest is sent when a browser wants to observe the pod via Relay.
// The Runner should connect to the specified Relay URL and start streaming terminal output.
type SubscribePodRequest struct {
	PodKey          string `json:"pod_key"`
	RelayURL        string `json:"relay_url"`    // Public URL via reverse proxy (e.g. wss://example.com/relay)
	RunnerToken     string `json:"runner_token"` // JWT token for Relay authentication
	IncludeSnapshot bool   `json:"include_snapshot"`
	SnapshotHistory int32  `json:"snapshot_history"`
}

// UnsubscribePodRequest is sent when all browsers have disconnected from the pod.
// The Runner should disconnect from the Relay.
type UnsubscribePodRequest struct {
	PodKey string `json:"pod_key"`
}

// QuerySandboxesRequest is sent to query sandbox status for specified pods.
type QuerySandboxesRequest struct {
	RequestID string                   `json:"request_id"`
	Queries   []*runnerv1.SandboxQuery `json:"queries"`
}

// SandboxStatusInfo contains sandbox status information.
type SandboxStatusInfo struct {
	PodKey                string `json:"pod_key"`
	Exists                bool   `json:"exists"`
	SandboxPath           string `json:"sandbox_path"`
	RepositoryURL         string `json:"repository_url"`
	BranchName            string `json:"branch_name"`
	CurrentCommit         string `json:"current_commit"`
	SizeBytes             int64  `json:"size_bytes"`
	LastModified          int64  `json:"last_modified"`
	HasUncommittedChanges bool   `json:"has_uncommitted_changes"`
	CanResume             bool   `json:"can_resume"`
	Error                 string `json:"error,omitempty"`
}

// ObservePodRequest is sent to query terminal state for a pod.
type ObservePodRequest struct {
	RequestID     string `json:"request_id"`
	PodKey        string `json:"pod_key"`
	Lines         int    `json:"lines"`
	IncludeScreen bool   `json:"include_screen"`
}

// ==================== Message Handler Interface ====================

// MessageHandler handles incoming messages from server.
type MessageHandler interface {
	// OnCreatePod handles pod creation command.
	// Uses Proto type directly for zero-copy message passing.
	OnCreatePod(cmd *runnerv1.CreatePodCommand) error
	OnTerminatePod(req TerminatePodRequest) error
	OnListPods() []PodInfo
	OnListRelayConnections() []RelayConnectionInfo
	OnPodInput(req PodInputRequest) error

	// OnSubscribePod handles subscribe pod command from server.
	// This notifies the Runner that a browser wants to observe the pod via Relay.
	// The Runner should connect to the specified Relay URL and start streaming terminal output.
	OnSubscribePod(req SubscribePodRequest) error

	// OnUnsubscribePod handles unsubscribe pod command from server.
	// This notifies the Runner that all browsers have disconnected.
	// The Runner should disconnect from the Relay.
	OnUnsubscribePod(req UnsubscribePodRequest) error

	// OnQuerySandboxes handles sandbox status query command from server.
	// Returns sandbox status for specified pod keys.
	OnQuerySandboxes(req QuerySandboxesRequest) error

	// OnObservePod handles observe pod command from server.
	// Reads VirtualTerminal state and sends result back via gRPC.
	OnObservePod(req ObservePodRequest) error

	// Autopilot commands
	// OnCreateAutopilot handles Autopilot creation command.
	OnCreateAutopilot(cmd *runnerv1.CreateAutopilotCommand) error

	// OnAutopilotControl handles Autopilot control commands (pause/resume/stop/approve/takeover/handback).
	OnAutopilotControl(cmd *runnerv1.AutopilotControlCommand) error

	// OnUpgradeRunner handles remote upgrade command from server.
	OnUpgradeRunner(cmd *runnerv1.UpgradeRunnerCommand) error

	// OnUploadLogs handles log upload command from server.
	OnUploadLogs(cmd *runnerv1.UploadLogsCommand) error

	// OnSendPrompt handles send_prompt command from server.
	// Routes the prompt to the pod via PodIO.SendInput (mode-agnostic).
	OnSendPrompt(cmd *runnerv1.SendPromptCommand) error
}
