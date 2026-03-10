package client

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

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
