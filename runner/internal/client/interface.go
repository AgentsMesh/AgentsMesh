package client

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Connection defines the interface for server communication.
// This interface abstracts GRPCConnection for testing and decoupling.
type Connection interface {
	// SetHandler sets the message handler for incoming messages.
	SetHandler(handler MessageHandler)

	// Connect establishes a connection to the server.
	Connect() error

	// Start starts the connection management loop (heartbeat, reconnect).
	Start()

	// Stop stops the connection and releases resources.
	Stop()

	// gRPC send methods

	// SendPodCreated sends a pod_created event to the server.
	// Includes sandbox_path and branch_name for Resume functionality.
	SendPodCreated(podKey string, pid int32, sandboxPath, branchName string) error

	// SendPodTerminated sends a pod_terminated event to the server.
	SendPodTerminated(podKey string, exitCode int32, errorMsg string) error

	// NOTE: SendTerminalOutput removed - terminal output is exclusively streamed via Relay

	// SendPtyResized sends a PTY resize event to the server.
	SendPtyResized(podKey string, cols, rows int32) error

	// SendError sends an error event to the server.
	SendError(podKey, code, message string) error

	// SendPodInitProgress sends a pod initialization progress event to the server.
	SendPodInitProgress(podKey, phase string, progress int32, message string) error

	// SendRequestRelayToken sends a request for a new relay token to the server.
	// This is called when the relay connection fails due to token expiration.
	// Note: SessionID has been removed - channels are now identified by PodKey only
	SendRequestRelayToken(podKey, relayURL string) error

	// SendSandboxesStatus sends sandbox status query response to the server.
	SendSandboxesStatus(requestID string, sandboxes []*SandboxStatusInfo) error

	// SendOSCNotification sends an OSC notification event to the server.
	// This is triggered by OSC 777 (iTerm2/Kitty) or OSC 9 (ConEmu/Windows Terminal) sequences.
	SendOSCNotification(podKey, title, body string) error

	// SendOSCTitle sends an OSC title change event to the server.
	// This is triggered by OSC 0/2 sequences for window/tab title changes.
	SendOSCTitle(podKey, title string) error

	// SendMessage sends a raw RunnerMessage to the server.
	// Used for Autopilot events and other custom messages.
	SendMessage(msg *runnerv1.RunnerMessage) error

	// QueueLength returns the current send queue length.
	QueueLength() int

	// QueueCapacity returns the send queue capacity.
	QueueCapacity() int

	// QueueUsage returns the terminal queue usage ratio (0.0 to 1.0).
	// Used for monitoring queue pressure.
	QueueUsage() float64

	// SetOrgSlug sets the organization slug.
	SetOrgSlug(orgSlug string)

	// GetOrgSlug returns the organization slug.
	GetOrgSlug() string
}

// Ensure GRPCConnection implements Connection interface.
var _ Connection = (*GRPCConnection)(nil)
