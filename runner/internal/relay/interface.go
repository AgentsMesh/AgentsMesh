package relay

import (
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// RelayClient defines the interface for a relay WebSocket client.
// This interface enables dependency injection and easier testing.
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

	// Handler registration
	SetInputHandler(handler InputHandler)
	SetResizeHandler(handler ResizeHandler)
	SetCloseHandler(handler CloseHandler)
	SetImagePasteHandler(handler ImagePasteHandler)
	SetReconnectHandler(handler func())
	SetTokenExpiredHandler(handler func() string)

	// Data transmission
	SendOutput(data []byte) error
	SendSnapshot(snapshot *vt.TerminalSnapshot) error
}

// Ensure Client implements RelayClient interface
var _ RelayClient = (*Client)(nil)
