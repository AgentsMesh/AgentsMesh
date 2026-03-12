package client

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ConnectionLifecycle defines lifecycle management operations.
// Use this interface when you only need to manage connection state.
type ConnectionLifecycle interface {
	// SetHandler sets the message handler for incoming messages.
	SetHandler(handler MessageHandler)

	// Connect establishes a connection to the server.
	Connect() error

	// Start starts the connection management loop (heartbeat, reconnect).
	Start()

	// Stop stops the connection and releases resources.
	Stop()
}

// ConnectionSender defines methods for sending messages to the server.
// Use this interface when you only need to send data without managing connection lifecycle.
type ConnectionSender interface {
	// SendPodCreated sends a pod_created event to the server.
	// Includes sandbox_path and branch_name for Resume functionality.
	SendPodCreated(podKey string, pid int32, sandboxPath, branchName string) error

	// SendPodTerminated sends a pod_terminated event to the server.
	SendPodTerminated(podKey string, exitCode int32, errorMsg string) error

	// SendPtyResized sends a PTY resize event to the server.
	SendPtyResized(podKey string, cols, rows int32) error

	// SendError sends an error event to the server.
	SendError(podKey, code, message string) error

	// SendPodInitProgress sends a pod initialization progress event to the server.
	SendPodInitProgress(podKey, phase string, progress int32, message string) error

	// SendRequestRelayToken sends a request for a new relay token to the server.
	// This is called when the relay connection fails due to token expiration.
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

	// SendAgentStatus sends an agent status change event to the server.
	// Status values: "executing", "waiting", "idle".
	SendAgentStatus(podKey string, status string) error

	// SendUpgradeStatus sends an upgrade status event to the server.
	SendUpgradeStatus(event *runnerv1.UpgradeStatusEvent) error

	// SendLogUploadStatus sends a log upload status event to the server.
	SendLogUploadStatus(event *runnerv1.LogUploadStatusEvent) error

	// SendTokenUsage sends a token usage report to the server.
	SendTokenUsage(podKey string, models []*runnerv1.TokenModelUsage) error
}

// ConnectionMonitor defines methods for monitoring connection health.
// Use this interface when you need to observe queue pressure or connection metrics.
type ConnectionMonitor interface {
	// QueueLength returns the current send queue length.
	QueueLength() int

	// QueueCapacity returns the send queue capacity.
	QueueCapacity() int

	// QueueUsage returns the terminal queue usage ratio (0.0 to 1.0).
	// Used for monitoring queue pressure.
	QueueUsage() float64
}

// ConnectionConfig defines methods for connection configuration.
type ConnectionConfig interface {
	// SetOrgSlug sets the organization slug.
	SetOrgSlug(orgSlug string)

	// GetOrgSlug returns the organization slug.
	GetOrgSlug() string
}

// ProgressSender defines the minimal interface for sending pod initialization progress.
// Use this interface in components that only need to report progress (e.g., PodBuilder).
type ProgressSender interface {
	// SendPodInitProgress sends a pod initialization progress event to the server.
	SendPodInitProgress(podKey, phase string, progress int32, message string) error
}

// Connection defines the full interface for server communication.
// This interface composes all sub-interfaces for backward compatibility.
// New code should prefer using the specific sub-interfaces when possible.
type Connection interface {
	ConnectionLifecycle
	ConnectionSender
	ConnectionMonitor
	ConnectionConfig
}

// Ensure GRPCConnection implements Connection interface.
var _ Connection = (*GRPCConnection)(nil)

// Verify that Connection satisfies all sub-interfaces
var (
	_ ConnectionLifecycle = (Connection)(nil)
	_ ConnectionSender    = (Connection)(nil)
	_ ConnectionMonitor   = (Connection)(nil)
	_ ConnectionConfig    = (Connection)(nil)
	_ ProgressSender      = (Connection)(nil)
)
