package relay

// RelayClient defines the interface for a relay WebSocket client.
// This is a generic message pipe — zero protocol knowledge.
// All mode-specific encoding/decoding lives in the consumer (PodRelay).
type RelayClient interface {
	// Connection lifecycle
	Connect() error
	Start() bool
	Stop()
	IsConnected() bool

	// Configuration
	GetRelayURL() string
	GetConnectedAt() int64
	UpdateToken(newToken string)

	// Generic message I/O (replaces all mode-specific methods)
	Send(msgType byte, payload []byte) error
	SetMessageHandler(msgType byte, handler func(payload []byte))

	// Lifecycle callbacks
	SetCloseHandler(handler CloseHandler)
	SetReconnectHandler(handler func())
	SetTokenExpiredHandler(handler func() string)
}

// Ensure Client implements RelayClient interface
var _ RelayClient = (*Client)(nil)
