package client

import (
	"sync"
)

// MockConnection is a mock implementation of Connection for testing.
type MockConnection struct {
	mu sync.Mutex

	// Handler
	handler MessageHandler

	// Captured calls for verification
	Events []EventCall

	// Configurable responses
	ConnectErr error
	SendErr    error

	// State
	started bool
	stopped bool
}

// EventCall represents a captured event call
type EventCall struct {
	Type MessageType
	Data interface{}
}

// NewMockConnection creates a new mock connection for testing.
func NewMockConnection() *MockConnection {
	return &MockConnection{}
}

// SetHandler implements Connection.
func (m *MockConnection) SetHandler(handler MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
}

// Connect implements Connection.
func (m *MockConnection) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ConnectErr != nil {
		return m.ConnectErr
	}
	return nil
}

// Start implements Connection.
func (m *MockConnection) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
}

// Stop implements Connection.
func (m *MockConnection) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
}

// QueueLength implements Connection.
func (m *MockConnection) QueueLength() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Events)
}

// QueueCapacity implements Connection.
func (m *MockConnection) QueueCapacity() int {
	return 100
}

// SetOrgSlug implements Connection.
func (m *MockConnection) SetOrgSlug(orgSlug string) {
	// No-op for mock
}

// GetOrgSlug implements Connection.
func (m *MockConnection) GetOrgSlug() string {
	return ""
}

// SendPodCreated implements Connection.
func (m *MockConnection) SendPodCreated(podKey string, pid int32, sandboxPath, branchName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MsgTypePodCreated, Data: map[string]interface{}{
		"pod_key":      podKey,
		"pid":          pid,
		"sandbox_path": sandboxPath,
		"branch_name":  branchName,
	}})
	return nil
}

// SendPodTerminated implements Connection.
func (m *MockConnection) SendPodTerminated(podKey string, exitCode int32, errorMsg string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MsgTypePodTerminated, Data: map[string]interface{}{"pod_key": podKey, "exit_code": exitCode, "error": errorMsg, "status": status}})
	return nil
}

// NOTE: SendTerminalOutput removed - output is exclusively streamed via Relay

// SendError implements Connection.
func (m *MockConnection) SendError(podKey, code, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MessageType("error"), Data: map[string]interface{}{"pod_key": podKey, "code": code, "message": message}})
	return nil
}

// SendPodInitProgress implements Connection.
func (m *MockConnection) SendPodInitProgress(podKey, phase string, progress int32, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MessageType("pod_init_progress"), Data: map[string]interface{}{"pod_key": podKey, "phase": phase, "progress": progress, "message": message}})
	return nil
}

// SendRequestRelayToken implements Connection.
// Note: SessionID has been removed - channels are now identified by PodKey only
func (m *MockConnection) SendRequestRelayToken(podKey, relayURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MessageType("request_relay_token"), Data: map[string]interface{}{"pod_key": podKey, "relay_url": relayURL}})
	return nil
}

// SendSandboxesStatus implements Connection.
func (m *MockConnection) SendSandboxesStatus(requestID string, results []*SandboxStatusInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MessageType("sandboxes_status"), Data: map[string]interface{}{"request_id": requestID, "results": results}})
	return nil
}

// Additional Send methods are in mock_connection_send.go

// Ensure MockConnection implements Connection interface
var _ Connection = (*MockConnection)(nil)
