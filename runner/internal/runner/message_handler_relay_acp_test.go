package runner

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// mockPodIO records calls to SendInput, RespondToPermission, CancelSession, and new control methods.
// Implements both PodIO and SessionAccess for ACP relay command tests.
type mockPodIO struct {
	mu           sync.Mutex
	inputs       []string
	permResps    []permResp
	cancelled    bool
	interrupted  bool
	permMode     string
	model        string
	controlReqs  []controlReq
	cancelErr    error
	sendErr      error
	permErr      error
	interruptErr error
	permModeErr  error
	modelErr     error
	controlErr   error
	controlNil   bool
}

type permResp struct {
	id           string
	approved     bool
	updatedInput map[string]any
}

type controlReq struct {
	subtype string
	payload map[string]any
}

func (m *mockPodIO) Mode() string                              { return "acp" }
func (m *mockPodIO) GetSnapshot(int) (string, error)           { return "", nil }
func (m *mockPodIO) GetAgentStatus() string                    { return "idle" }
func (m *mockPodIO) SubscribeStateChange(string, func(string)) {}
func (m *mockPodIO) UnsubscribeStateChange(string)             {}
func (m *mockPodIO) GetPID() int                               { return 0 }
func (m *mockPodIO) Stop()                                     {}
func (m *mockPodIO) Teardown() string                          { return "" }
func (m *mockPodIO) SetExitHandler(func(int))                  {}
func (m *mockPodIO) Detach()                                   {}
func (m *mockPodIO) Start() error                              { return nil }
func (m *mockPodIO) SetIOErrorHandler(func(error))             {}

func (m *mockPodIO) SendInput(text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputs = append(m.inputs, text)
	return m.sendErr
}

func (m *mockPodIO) RespondToPermission(id string, approved bool, updatedInput map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.permResps = append(m.permResps, permResp{id, approved, updatedInput})
	return m.permErr
}

func (m *mockPodIO) CancelSession() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelled = true
	return m.cancelErr
}

func (m *mockPodIO) NotifyStateChange(string) {
	// No-op for tests.
}

func (m *mockPodIO) Interrupt() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interrupted = true
	return m.interruptErr
}

func (m *mockPodIO) SetPermissionMode(mode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.permMode = mode
	return m.permModeErr
}

func (m *mockPodIO) SetModel(model string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.model = model
	return m.modelErr
}

func (m *mockPodIO) SendControlRequest(subtype string, payload map[string]any) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.controlReqs = append(m.controlReqs, controlReq{subtype, payload})
	if m.controlNil {
		return nil, m.controlErr
	}
	return map[string]any{"ok": true}, m.controlErr
}

// newTestHandler creates a minimal RunnerMessageHandler for relay tests.
func newTestHandler() *RunnerMessageHandler {
	return &RunnerMessageHandler{}
}

func TestHandleAcpRelayCommand_Prompt(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	pod := &Pod{PodKey: "test-pod", IO: mock}
	pod.Relay = NewACPPodRelay("test-pod", nil, nil, nil)
	pod.SetRelayClient(mc)

	payload, _ := json.Marshal(map[string]any{
		"type":   "prompt",
		"prompt": "hello world",
	})
	h.handleAcpRelayCommand(pod, payload)

	// Verify echo contentChunk was sent via relay
	if mc.CountSentByType(relay.MsgTypeAcpEvent) != 1 {
		t.Errorf("expected 1 relay event (echo), got %d",
			mc.CountSentByType(relay.MsgTypeAcpEvent))
	}

	// Verify SendInput was called
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.inputs) != 1 || mock.inputs[0] != "hello world" {
		t.Errorf("inputs = %v, want [hello world]", mock.inputs)
	}
}

func TestHandleAcpRelayCommand_PermissionResponse(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{
		"type":      "permission_response",
		"requestId": "42",
		"approved":  true,
	})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.permResps) != 1 {
		t.Fatalf("expected 1 permission response, got %d", len(mock.permResps))
	}
	if mock.permResps[0].id != "42" || !mock.permResps[0].approved {
		t.Errorf("perm = %+v, want {42 true}", mock.permResps[0])
	}
}

func TestHandleAcpRelayCommand_PermissionResponse_Decline(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{
		"type":      "permission_response",
		"requestId": "99",
		"approved":  false,
	})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.permResps) != 1 {
		t.Fatalf("expected 1 permission response, got %d", len(mock.permResps))
	}
	if mock.permResps[0].id != "99" || mock.permResps[0].approved {
		t.Errorf("perm = %+v, want {99 false}", mock.permResps[0])
	}
}

func TestHandleAcpRelayCommand_PermissionResponse_WithUpdatedInput(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{
		"type":      "permission_response",
		"requestId": "ask-1",
		"approved":  true,
		"updatedInput": map[string]any{
			"answers": map[string]any{"Which DB?": "PostgreSQL"},
		},
	})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.permResps) != 1 {
		t.Fatalf("expected 1 permission response, got %d", len(mock.permResps))
	}
	resp := mock.permResps[0]
	if resp.id != "ask-1" || !resp.approved {
		t.Errorf("perm = {id:%s approved:%v}, want {ask-1 true}", resp.id, resp.approved)
	}
	if resp.updatedInput == nil {
		t.Fatal("updatedInput should not be nil")
	}
	answers, ok := resp.updatedInput["answers"].(map[string]any)
	if !ok {
		t.Fatal("answers should be map")
	}
	if answers["Which DB?"] != "PostgreSQL" {
		t.Errorf("answer = %v, want PostgreSQL", answers["Which DB?"])
	}
}

func TestHandleAcpRelayCommand_PermissionResponse_WithUpdatedPermissions(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{
		"type":      "permission_response",
		"requestId": "perm-always",
		"approved":  true,
		"updatedInput": map[string]any{
			"updatedPermissions": []map[string]any{
				{"type": "addRules", "destination": "session", "rules": []map[string]any{
					{"tool": "Edit", "permission": "allow"},
				}},
			},
		},
	})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.permResps) != 1 {
		t.Fatalf("expected 1 permission response, got %d", len(mock.permResps))
	}
	resp := mock.permResps[0]
	if resp.updatedInput == nil {
		t.Fatal("updatedInput should contain updatedPermissions")
	}
	if _, ok := resp.updatedInput["updatedPermissions"]; !ok {
		t.Error("updatedPermissions should be present in updatedInput")
	}
}

func TestHandleAcpRelayCommand_Cancel(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{"type": "cancel"})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !mock.cancelled {
		t.Error("CancelSession was not called")
	}
}

func TestHandleAcpRelayCommand_UnknownType(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{"type": "unknown_cmd"})
	h.handleAcpRelayCommand(pod, payload)

	// No methods should have been called
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.inputs) != 0 || len(mock.permResps) != 0 || mock.cancelled {
		t.Error("expected no-op for unknown command type")
	}
}

func TestHandleAcpRelayCommand_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	// Should not panic on invalid JSON
	h.handleAcpRelayCommand(pod, []byte("not json"))

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.inputs) != 0 || len(mock.permResps) != 0 || mock.cancelled {
		t.Error("expected no-op for invalid JSON")
	}
}

func TestHandleAcpRelayCommand_NilIO(t *testing.T) {
	h := newTestHandler()
	pod := &Pod{PodKey: "test-pod", IO: nil}

	payload, _ := json.Marshal(map[string]any{
		"type":   "prompt",
		"prompt": "hello",
	})
	// Should not panic when IO is nil
	h.handleAcpRelayCommand(pod, payload)
}

func TestHandleAcpRelayCommand_Interrupt(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{"type": "interrupt"})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !mock.interrupted {
		t.Error("Interrupt was not called")
	}
}

func TestHandleAcpRelayCommand_SetPermissionMode(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{
		"type": "set_permission_mode",
		"mode": "acceptEdits",
	})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.permMode != "acceptEdits" {
		t.Errorf("permMode = %q, want %q", mock.permMode, "acceptEdits")
	}
}

func TestHandleAcpRelayCommand_SetModel(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	pod := &Pod{PodKey: "test-pod", IO: mock}

	payload, _ := json.Marshal(map[string]any{
		"type":  "set_model",
		"model": "claude-opus-4",
	})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.model != "claude-opus-4" {
		t.Errorf("model = %q, want %q", mock.model, "claude-opus-4")
	}
}

func TestHandleAcpRelayCommand_GenericControlRequest(t *testing.T) {
	h := newTestHandler()
	mock := &mockPodIO{}
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	pod := &Pod{PodKey: "test-pod", IO: mock}
	pod.Relay = NewACPPodRelay("test-pod", nil, nil, nil)
	pod.SetRelayClient(mc)

	payload, _ := json.Marshal(map[string]any{
		"type":    "control_request",
		"subtype": "mcp_status",
		"payload": map[string]any{},
	})
	h.handleAcpRelayCommand(pod, payload)

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.controlReqs) != 1 {
		t.Fatalf("expected 1 control_request, got %d", len(mock.controlReqs))
	}
	if mock.controlReqs[0].subtype != "mcp_status" {
		t.Errorf("subtype = %q, want %q", mock.controlReqs[0].subtype, "mcp_status")
	}
}
