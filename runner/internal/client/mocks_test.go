package client

import (
	"sync"
)

// mockHandler is a mock implementation of MessageHandler for testing.
type mockHandler struct {
	mu sync.Mutex

	createPodCalled      bool
	terminatePodCalled   bool
	terminalInputCalled  bool
	terminalResizeCalled bool

	lastCreateReq    CreatePodRequest
	lastTerminateReq TerminatePodRequest
	lastInputReq     TerminalInputRequest
	lastResizeReq    TerminalResizeRequest

	pods []PodInfo
}

func (h *mockHandler) OnCreatePod(req CreatePodRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.createPodCalled = true
	h.lastCreateReq = req
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

func (h *mockHandler) OnListPods() []PodInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pods
}

// mockHandlerWithError is a mock handler that can return errors.
type mockHandlerWithError struct {
	createError    error
	terminateError error
	inputError     error
	resizeError    error
}

func (h *mockHandlerWithError) OnCreatePod(req CreatePodRequest) error {
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

func (h *mockHandlerWithError) OnListPods() []PodInfo {
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
