package client

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// --- Additional tests for message router ---

func TestNewMessageRouter(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}

	router := NewMessageRouter(handler, sender)

	if router == nil {
		t.Fatal("NewMessageRouter returned nil")
	}
	if router.handler != handler {
		t.Error("handler not set correctly")
	}
	if router.eventSender != sender {
		t.Error("eventSender not set correctly")
	}
}

func TestMessageRouterRouteNilHandler(t *testing.T) {
	sender := &mockEventSender{}
	router := NewMessageRouter(nil, sender)

	// Should not panic with nil handler
	msg := ProtocolMessage{Type: MsgTypeCreatePod}
	router.Route(msg) // Should log and return without panic
}

func TestMessageRouterRouteUnknownType(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	// Unknown message type should be logged but not cause error
	msg := ProtocolMessage{Type: "unknown_type"}
	router.Route(msg) // Should log unknown type
}

func TestMessageRouterHandleListPods(t *testing.T) {
	handler := &mockHandler{
		pods: []PodInfo{
			{PodKey: "pod-1", Status: "running"},
			{PodKey: "pod-2", Status: "idle"},
		},
	}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	// Route list_pods message
	msg := ProtocolMessage{Type: MsgTypeListPods}
	router.Route(msg)

	// Verify event was sent
	sender.mu.Lock()
	events := sender.events
	sender.mu.Unlock()

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != MsgTypePodList {
		t.Errorf("event type: got %v, want %v", events[0].Type, MsgTypePodList)
	}

	// Verify pods data
	pods, ok := events[0].Data.([]PodInfo)
	if !ok {
		t.Fatalf("expected []PodInfo, got %T", events[0].Data)
	}
	if len(pods) != 2 {
		t.Errorf("expected 2 pods, got %d", len(pods))
	}
}

func TestMessageRouterHandleListPodsWithSendError(t *testing.T) {
	handler := &mockHandler{
		pods: []PodInfo{{PodKey: "pod-1"}},
	}
	sender := &mockEventSender{
		err: errors.New("send failed"),
	}
	router := NewMessageRouter(handler, sender)

	// Route list_pods message - should not panic on send error
	msg := ProtocolMessage{Type: MsgTypeListPods}
	router.Route(msg) // Should log error but not panic
}

func TestMessageRouterHandleCreatePod(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	reqData, _ := json.Marshal(CreatePodRequest{
		PodKey:        "test-pod",
		LaunchCommand: "claude",
		LaunchArgs:    []string{"--model", "opus"},
	})

	msg := ProtocolMessage{
		Type: MsgTypeCreatePod,
		Data: reqData,
	}
	router.Route(msg)

	handler.mu.Lock()
	defer handler.mu.Unlock()

	if !handler.createPodCalled {
		t.Error("OnCreatePod should have been called")
	}
	if handler.lastCreateReq.PodKey != "test-pod" {
		t.Errorf("PodKey: got %v, want test-pod", handler.lastCreateReq.PodKey)
	}
	if handler.lastCreateReq.LaunchCommand != "claude" {
		t.Errorf("LaunchCommand: got %v, want claude", handler.lastCreateReq.LaunchCommand)
	}
}

func TestMessageRouterHandleTerminatePod(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	reqData, _ := json.Marshal(TerminatePodRequest{
		PodKey: "pod-to-terminate",
	})

	msg := ProtocolMessage{
		Type: MsgTypeTerminatePod,
		Data: reqData,
	}
	router.Route(msg)

	handler.mu.Lock()
	defer handler.mu.Unlock()

	if !handler.terminatePodCalled {
		t.Error("OnTerminatePod should have been called")
	}
	if handler.lastTerminateReq.PodKey != "pod-to-terminate" {
		t.Errorf("PodKey: got %v, want pod-to-terminate", handler.lastTerminateReq.PodKey)
	}
}

func TestMessageRouterHandleTerminalInput(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	reqData, _ := json.Marshal(TerminalInputRequest{
		PodKey: "pod-1",
		Data:   "aGVsbG8=", // "hello" in base64
	})

	msg := ProtocolMessage{
		Type: MsgTypeTerminalInput,
		Data: reqData,
	}
	router.Route(msg)

	handler.mu.Lock()
	defer handler.mu.Unlock()

	if !handler.terminalInputCalled {
		t.Error("OnTerminalInput should have been called")
	}
	if handler.lastInputReq.PodKey != "pod-1" {
		t.Errorf("PodKey: got %v, want pod-1", handler.lastInputReq.PodKey)
	}
	if handler.lastInputReq.Data != "aGVsbG8=" {
		t.Errorf("Data: got %v, want aGVsbG8=", handler.lastInputReq.Data)
	}
}

func TestMessageRouterHandleTerminalResize(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	reqData, _ := json.Marshal(TerminalResizeRequest{
		PodKey: "pod-1",
		Cols:   120,
		Rows:   40,
	})

	msg := ProtocolMessage{
		Type: MsgTypeTerminalResize,
		Data: reqData,
	}
	router.Route(msg)

	handler.mu.Lock()
	defer handler.mu.Unlock()

	if !handler.terminalResizeCalled {
		t.Error("OnTerminalResize should have been called")
	}
	if handler.lastResizeReq.Cols != 120 {
		t.Errorf("Cols: got %v, want 120", handler.lastResizeReq.Cols)
	}
	if handler.lastResizeReq.Rows != 40 {
		t.Errorf("Rows: got %v, want 40", handler.lastResizeReq.Rows)
	}
}

func TestMessageRouterAllMessageTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType MessageType
		data    interface{}
		check   func(*mockHandler) bool
	}{
		{
			name:    "create_pod",
			msgType: MsgTypeCreatePod,
			data:    CreatePodRequest{PodKey: "p1"},
			check:   func(h *mockHandler) bool { return h.createPodCalled },
		},
		{
			name:    "terminate_pod",
			msgType: MsgTypeTerminatePod,
			data:    TerminatePodRequest{PodKey: "p1"},
			check:   func(h *mockHandler) bool { return h.terminatePodCalled },
		},
		{
			name:    "terminal_input",
			msgType: MsgTypeTerminalInput,
			data:    TerminalInputRequest{PodKey: "p1", Data: "test"},
			check:   func(h *mockHandler) bool { return h.terminalInputCalled },
		},
		{
			name:    "terminal_resize",
			msgType: MsgTypeTerminalResize,
			data:    TerminalResizeRequest{PodKey: "p1", Cols: 80, Rows: 24},
			check:   func(h *mockHandler) bool { return h.terminalResizeCalled },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockHandler{}
			sender := &mockEventSender{}
			router := NewMessageRouter(handler, sender)

			data, _ := json.Marshal(tt.data)
			msg := ProtocolMessage{
				Type: tt.msgType,
				Data: data,
			}
			router.Route(msg)

			handler.mu.Lock()
			defer handler.mu.Unlock()

			if !tt.check(handler) {
				t.Errorf("handler method not called for %s", tt.name)
			}
		})
	}
}

func TestMessageRouterHandlerErrors(t *testing.T) {
	tests := []struct {
		name        string
		msgType     MessageType
		data        interface{}
		handlerErr  error
		shouldPanic bool
	}{
		{
			name:       "create_pod with error",
			msgType:    MsgTypeCreatePod,
			data:       CreatePodRequest{PodKey: "p1"},
			handlerErr: context.DeadlineExceeded,
		},
		{
			name:       "terminate_pod with error",
			msgType:    MsgTypeTerminatePod,
			data:       TerminatePodRequest{PodKey: "p1"},
			handlerErr: context.Canceled,
		},
		{
			name:       "terminal_input with error",
			msgType:    MsgTypeTerminalInput,
			data:       TerminalInputRequest{PodKey: "p1"},
			handlerErr: errors.New("input error"),
		},
		{
			name:       "terminal_resize with error",
			msgType:    MsgTypeTerminalResize,
			data:       TerminalResizeRequest{PodKey: "p1", Cols: 80, Rows: 24},
			handlerErr: errors.New("resize error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockHandlerWithError{
				createError:    tt.handlerErr,
				terminateError: tt.handlerErr,
				inputError:     tt.handlerErr,
				resizeError:    tt.handlerErr,
			}
			sender := &mockEventSender{}
			router := NewMessageRouter(handler, sender)

			data, _ := json.Marshal(tt.data)
			msg := ProtocolMessage{
				Type: tt.msgType,
				Data: data,
			}

			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Route panicked: %v", r)
				}
			}()

			router.Route(msg)
		})
	}
}
