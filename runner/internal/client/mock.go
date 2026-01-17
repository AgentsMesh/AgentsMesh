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
func (m *MockConnection) SendPodCreated(podKey string, pid int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MsgTypePodCreated, Data: map[string]interface{}{"pod_key": podKey, "pid": pid}})
	return nil
}

// SendPodTerminated implements Connection.
func (m *MockConnection) SendPodTerminated(podKey string, exitCode int32, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MsgTypePodTerminated, Data: map[string]interface{}{"pod_key": podKey, "exit_code": exitCode, "error": errorMsg}})
	return nil
}

// SendTerminalOutput implements Connection.
func (m *MockConnection) SendTerminalOutput(podKey string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MsgTypeTerminalOutput, Data: map[string]interface{}{"pod_key": podKey, "data": data}})
	return nil
}

// SendPtyResized implements Connection.
func (m *MockConnection) SendPtyResized(podKey string, cols, rows int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MsgTypePtyResized, Data: map[string]interface{}{"pod_key": podKey, "cols": cols, "rows": rows}})
	return nil
}

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

// --- Test helper methods ---

// SimulateCreatePod simulates server sending a create_pod message.
func (m *MockConnection) SimulateCreatePod(req CreatePodRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnCreatePod(req)
	}
	return nil
}

// SimulateTerminatePod simulates server sending a terminate_pod message.
func (m *MockConnection) SimulateTerminatePod(req TerminatePodRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnTerminatePod(req)
	}
	return nil
}

// SimulateTerminalInput simulates server sending a terminal_input message.
func (m *MockConnection) SimulateTerminalInput(req TerminalInputRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnTerminalInput(req)
	}
	return nil
}

// SimulateTerminalResize simulates server sending a terminal_resize message.
func (m *MockConnection) SimulateTerminalResize(req TerminalResizeRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnTerminalResize(req)
	}
	return nil
}

// GetPods returns pods from handler (if available).
func (m *MockConnection) GetPods() []PodInfo {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnListPods()
	}
	return nil
}

// GetEvents returns captured events (thread-safe).
func (m *MockConnection) GetEvents() []EventCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]EventCall, len(m.Events))
	copy(result, m.Events)
	return result
}


// IsStarted returns whether Start was called.
func (m *MockConnection) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

// IsStopped returns whether Stop was called.
func (m *MockConnection) IsStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

// Reset clears all captured calls.
func (m *MockConnection) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = nil
	m.started = false
	m.stopped = false
}

// Ensure MockConnection implements Connection interface
var _ Connection = (*MockConnection)(nil)
