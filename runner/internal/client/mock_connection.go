package client

import (
	"sync"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
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
		"branch_name":   branchName,
	}})
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

// NOTE: SendTerminalOutput removed - terminal output is exclusively streamed via Relay

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

// --- Test helper methods ---

// SimulateCreatePod simulates server sending a create_pod message.
// Uses Proto type directly for consistency with actual implementation.
func (m *MockConnection) SimulateCreatePod(cmd *runnerv1.CreatePodCommand) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnCreatePod(cmd)
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

// SimulateTerminalRedraw simulates server sending a terminal_redraw message.
func (m *MockConnection) SimulateTerminalRedraw(req TerminalRedrawRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnTerminalRedraw(req)
	}
	return nil
}

// SimulateSubscribeTerminal simulates server sending a subscribe_terminal message.
func (m *MockConnection) SimulateSubscribeTerminal(req SubscribeTerminalRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnSubscribeTerminal(req)
	}
	return nil
}

// SimulateUnsubscribeTerminal simulates server sending an unsubscribe_terminal message.
func (m *MockConnection) SimulateUnsubscribeTerminal(req UnsubscribeTerminalRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnUnsubscribeTerminal(req)
	}
	return nil
}

// SimulateQuerySandboxes simulates server sending a query_sandboxes message.
func (m *MockConnection) SimulateQuerySandboxes(req QuerySandboxesRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnQuerySandboxes(req)
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

// QueueUsage returns the mock queue usage (always 0 for testing).
func (m *MockConnection) QueueUsage() float64 {
	return 0.0
}

// SendOSCNotification records an OSC notification event.
func (m *MockConnection) SendOSCNotification(podKey, title, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, EventCall{
		Type: "osc_notification",
		Data: map[string]string{
			"pod_key": podKey,
			"title":   title,
			"body":    body,
		},
	})
	return nil
}

// SendOSCTitle records an OSC title change event.
func (m *MockConnection) SendOSCTitle(podKey, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, EventCall{
		Type: "osc_title",
		Data: map[string]string{
			"pod_key": podKey,
			"title":   title,
		},
	})
	return nil
}

// SendAgentStatus records an agent status change event.
func (m *MockConnection) SendAgentStatus(podKey string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{
		Type: MessageType("agent_status"),
		Data: map[string]string{
			"pod_key": podKey,
			"status":  status,
		},
	})
	return nil
}

// SendMessage records a raw RunnerMessage.
func (m *MockConnection) SendMessage(msg *runnerv1.RunnerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{
		Type: "raw_message",
		Data: msg,
	})
	return nil
}

// SimulateCreateAutopilot simulates server sending a create_autopilot message.
func (m *MockConnection) SimulateCreateAutopilot(cmd *runnerv1.CreateAutopilotCommand) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnCreateAutopilot(cmd)
	}
	return nil
}

// SimulateAutopilotControl simulates server sending an autopilot_control message.
func (m *MockConnection) SimulateAutopilotControl(cmd *runnerv1.AutopilotControlCommand) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnAutopilotControl(cmd)
	}
	return nil
}

// Ensure MockConnection implements Connection interface
var _ Connection = (*MockConnection)(nil)
