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

// SimulatePodInput simulates server sending a pod_input message.
func (m *MockConnection) SimulatePodInput(req PodInputRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnPodInput(req)
	}
	return nil
}

// SimulateSubscribePod simulates server sending a subscribe_pod message.
func (m *MockConnection) SimulateSubscribePod(req SubscribePodRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnSubscribePod(req)
	}
	return nil
}

// SimulateUnsubscribePod simulates server sending an unsubscribe_pod message.
func (m *MockConnection) SimulateUnsubscribePod(req UnsubscribePodRequest) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnUnsubscribePod(req)
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

// SimulateSendPrompt simulates server sending a send_prompt message.
func (m *MockConnection) SimulateSendPrompt(cmd *runnerv1.SendPromptCommand) error {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnSendPrompt(cmd)
	}
	return nil
}
