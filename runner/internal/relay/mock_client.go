package relay

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// MockClient is a mock implementation of RelayClient for testing.
type MockClient struct {
	mu sync.RWMutex

	// Configuration
	relayURL string
	token    string

	// State
	connected   bool
	connectedAt int64

	// Tracking
	ConnectCalled     bool
	StartCalled       bool
	StopCalled        bool
	UpdateTokenCalls  []string
	SendSnapshotCalls int // Tracks SendSnapshot call count

	// Configurable behavior
	ConnectError error
	StartResult  bool
	StopDelay    time.Duration // Artificial delay in Stop() to simulate real behavior
	OnStopHook   func()        // Called during Stop() to simulate close handler side-effects

	// Handlers (stored but not used in mock)
	onInput        InputHandler
	onResize       ResizeHandler
	onClose        CloseHandler
	onImagePaste   ImagePasteHandler
	onReconnect    func()
	onTokenExpired func() string
}

// NewMockClient creates a new mock relay client for testing.
func NewMockClient(relayURL string) *MockClient {
	return &MockClient{
		relayURL:    relayURL,
		StartResult: true, // Default to successful start
	}
}

// Connect implements RelayClient.
func (m *MockClient) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectCalled = true
	if m.ConnectError != nil {
		return m.ConnectError
	}
	m.connected = true
	return nil
}

// Start implements RelayClient.
func (m *MockClient) Start() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartCalled = true
	return m.StartResult
}

// Stop implements RelayClient.
func (m *MockClient) Stop() {
	m.mu.Lock()
	delay := m.StopDelay
	hook := m.OnStopHook
	m.StopCalled = true
	m.connected = false
	m.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}
	if hook != nil {
		hook()
	}
}

// IsConnected implements RelayClient.
func (m *MockClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// GetRelayURL implements RelayClient.
func (m *MockClient) GetRelayURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.relayURL
}

// GetConnectedAt implements RelayClient.
func (m *MockClient) GetConnectedAt() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connectedAt
}

// UpdateToken implements RelayClient.
func (m *MockClient) UpdateToken(newToken string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = newToken
	m.UpdateTokenCalls = append(m.UpdateTokenCalls, newToken)
}

// SetInputHandler implements RelayClient.
func (m *MockClient) SetInputHandler(handler InputHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onInput = handler
}

// SetResizeHandler implements RelayClient.
func (m *MockClient) SetResizeHandler(handler ResizeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onResize = handler
}

// SetCloseHandler implements RelayClient.
func (m *MockClient) SetCloseHandler(handler CloseHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onClose = handler
}

// SetImagePasteHandler implements RelayClient.
func (m *MockClient) SetImagePasteHandler(handler ImagePasteHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onImagePaste = handler
}

// SetReconnectHandler implements RelayClient.
func (m *MockClient) SetReconnectHandler(handler func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onReconnect = handler
}

// SetTokenExpiredHandler implements RelayClient.
func (m *MockClient) SetTokenExpiredHandler(handler func() string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onTokenExpired = handler
}

// SendOutput implements RelayClient.
func (m *MockClient) SendOutput(data []byte) error {
	return nil
}

// SendSnapshot implements RelayClient.
func (m *MockClient) SendSnapshot(snapshot *vt.TerminalSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SendSnapshotCalls++
	return nil
}

// --- Test helpers ---

// SetConnected sets the connected state for testing.
func (m *MockClient) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// SetConnectedAt sets the connectedAt timestamp for testing.
func (m *MockClient) SetConnectedAt(ts int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectedAt = ts
}

// GetToken returns the current token for verification.
func (m *MockClient) GetToken() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.token
}

// Reset clears all tracking state.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectCalled = false
	m.StartCalled = false
	m.StopCalled = false
	m.UpdateTokenCalls = nil
	m.SendSnapshotCalls = 0
}

// Ensure MockClient implements RelayClient interface
var _ RelayClient = (*MockClient)(nil)
