// Package client provides communication with AgentsMesh server via gRPC.
package client

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// MessageType defines the type of message (used for mock testing).
type MessageType string

const (
	// Event types (Runner -> Backend)
	MsgTypePodCreated     MessageType = "pod_created"
	MsgTypePodTerminated  MessageType = "pod_terminated"
	MsgTypeTerminalOutput MessageType = "terminal_output"
	MsgTypePtyResized     MessageType = "pty_resized"
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
	PodKey       string `json:"pod_key"`
	Status       string `json:"status"`
	ClaudeStatus string `json:"claude_status"`
	Pid          int    `json:"pid"`
}

// ==================== Terminal Data Structures ====================

// TerminalInputRequest is sent to write to PTY.
type TerminalInputRequest struct {
	PodKey string `json:"pod_key"`
	Data   []byte `json:"data"` // Binary data (gRPC uses native bytes, no base64 needed)
}

// TerminalResizeRequest is sent to resize PTY.
type TerminalResizeRequest struct {
	PodKey string `json:"pod_key"`
	Cols   uint16 `json:"cols"`
	Rows   uint16 `json:"rows"`
}

// TerminalRedrawRequest is sent to trigger terminal redraw without changing size.
// Used for restoring terminal state after server restart.
type TerminalRedrawRequest struct {
	PodKey string `json:"pod_key"`
}

// ==================== Message Handler Interface ====================

// MessageHandler handles incoming messages from server.
type MessageHandler interface {
	// OnCreatePod handles pod creation command.
	// Uses Proto type directly for zero-copy message passing.
	OnCreatePod(cmd *runnerv1.CreatePodCommand) error
	OnTerminatePod(req TerminatePodRequest) error
	OnListPods() []PodInfo
	OnTerminalInput(req TerminalInputRequest) error
	OnTerminalResize(req TerminalResizeRequest) error
	OnTerminalRedraw(req TerminalRedrawRequest) error
}
