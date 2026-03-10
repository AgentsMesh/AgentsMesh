package client

import (
	"sync"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// mockHandler is a mock implementation of MessageHandler for testing.
type mockHandler struct {
	mu sync.Mutex

	createPodCalled      bool
	terminatePodCalled   bool
	terminalInputCalled  bool
	terminalResizeCalled bool
	terminalRedrawCalled bool

	lastCreateCmd    *runnerv1.CreatePodCommand
	lastTerminateReq TerminatePodRequest
	lastInputReq     TerminalInputRequest
	lastResizeReq    TerminalResizeRequest
	lastRedrawReq    TerminalRedrawRequest

	pods []PodInfo
}

func (h *mockHandler) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.createPodCalled = true
	h.lastCreateCmd = cmd
	return nil
}

func (h *mockHandler) OnTerminatePod(req TerminatePodRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.terminatePodCalled = true
	h.lastTerminateReq = req
	return nil
}

func (h *mockHandler) OnTerminalInput(req TerminalInputRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.terminalInputCalled = true
	h.lastInputReq = req
	return nil
}

func (h *mockHandler) OnTerminalResize(req TerminalResizeRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.terminalResizeCalled = true
	h.lastResizeReq = req
	return nil
}

func (h *mockHandler) OnTerminalRedraw(req TerminalRedrawRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.terminalRedrawCalled = true
	h.lastRedrawReq = req
	return nil
}

func (h *mockHandler) OnListPods() []PodInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pods
}

func (h *mockHandler) OnListRelayConnections() []RelayConnectionInfo {
	return nil
}

func (h *mockHandler) OnSubscribeTerminal(req SubscribeTerminalRequest) error {
	return nil
}

func (h *mockHandler) OnUnsubscribeTerminal(req UnsubscribeTerminalRequest) error {
	return nil
}

func (h *mockHandler) OnQuerySandboxes(req QuerySandboxesRequest) error {
	return nil
}

func (h *mockHandler) OnCreateAutopilot(cmd *runnerv1.CreateAutopilotCommand) error {
	return nil
}

func (h *mockHandler) OnAutopilotControl(cmd *runnerv1.AutopilotControlCommand) error {
	return nil
}

func (h *mockHandler) OnUpgradeRunner(cmd *runnerv1.UpgradeRunnerCommand) error {
	return nil
}

// mockHandlerWithError is a mock handler that can return errors.
type mockHandlerWithError struct {
	createError    error
	terminateError error
	inputError     error
	resizeError    error
	redrawError    error
}

func (h *mockHandlerWithError) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	return h.createError
}

func (h *mockHandlerWithError) OnTerminatePod(req TerminatePodRequest) error {
	return h.terminateError
}

func (h *mockHandlerWithError) OnTerminalInput(req TerminalInputRequest) error {
	return h.inputError
}

func (h *mockHandlerWithError) OnTerminalResize(req TerminalResizeRequest) error {
	return h.resizeError
}

func (h *mockHandlerWithError) OnTerminalRedraw(req TerminalRedrawRequest) error {
	return h.redrawError
}

func (h *mockHandlerWithError) OnListPods() []PodInfo {
	return nil
}

func (h *mockHandlerWithError) OnListRelayConnections() []RelayConnectionInfo {
	return nil
}

func (h *mockHandlerWithError) OnSubscribeTerminal(req SubscribeTerminalRequest) error {
	return nil
}

func (h *mockHandlerWithError) OnUnsubscribeTerminal(req UnsubscribeTerminalRequest) error {
	return nil
}

func (h *mockHandlerWithError) OnQuerySandboxes(req QuerySandboxesRequest) error {
	return nil
}

func (h *mockHandlerWithError) OnCreateAutopilot(cmd *runnerv1.CreateAutopilotCommand) error {
	return nil
}

func (h *mockHandlerWithError) OnAutopilotControl(cmd *runnerv1.AutopilotControlCommand) error {
	return nil
}

func (h *mockHandlerWithError) OnUpgradeRunner(cmd *runnerv1.UpgradeRunnerCommand) error {
	return nil
}

// mockEventSender is a mock implementation of EventSender for testing.
type mockEventSender struct {
	mu     sync.Mutex
	events []sentEvent
	err    error
}

type sentEvent struct {
	Type MessageType
	Data interface{}
}

func (s *mockEventSender) SendEvent(msgType MessageType, data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, sentEvent{Type: msgType, Data: data})
	return nil
}
