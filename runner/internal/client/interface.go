package client

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
	SendPodCreated(podKey string, pid int32) error

	// SendPodTerminated sends a pod_terminated event to the server.
	SendPodTerminated(podKey string, exitCode int32, errorMsg string) error

	// SendTerminalOutput sends terminal output to the server.
	// Non-blocking: drops silently if buffer is full (TUI frames are expendable).
	SendTerminalOutput(podKey string, data []byte) error

	// SendPtyResized sends a PTY resize event to the server.
	SendPtyResized(podKey string, cols, rows int32) error

	// SendError sends an error event to the server.
	SendError(podKey, code, message string) error

	// SendPodInitProgress sends a pod initialization progress event to the server.
	SendPodInitProgress(podKey, phase string, progress int32, message string) error

	// SendRequestRelayToken sends a request for a new relay token to the server.
	// This is called when the relay connection fails due to token expiration.
	SendRequestRelayToken(podKey, sessionID, relayURL string) error

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
