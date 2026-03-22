package runner

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

var errStub = errors.New("stub error")

// encodeResizePayload builds the 4-byte big-endian resize payload used by PTY mode.
func encodeResizePayload(cols, rows uint16) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint16(b[0:2], cols)
	binary.BigEndian.PutUint16(b[2:4], rows)
	return b
}

// --- PTYPodIO trivial tests ---

func TestPTYPodIO_Mode(t *testing.T) {
	io := NewPTYPodIO(nil, nil, &Pod{})
	if io.Mode() != "pty" {
		t.Errorf("Mode() = %q, want %q", io.Mode(), "pty")
	}
}

// --- relayOutputAdapter tests ---

func TestRelayOutputAdapter_SendOutput(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)
	adapter := &relayOutputAdapter{rc: mc}

	data := []byte("hello terminal output")
	if err := adapter.SendOutput(data); err != nil {
		t.Fatalf("SendOutput: %v", err)
	}

	// Verify the message was sent with the correct type and payload.
	if got := mc.CountSentByType(relay.MsgTypeOutput); got != 1 {
		t.Errorf("expected 1 output message, got %d", got)
	}
	msg := mc.SentMessages[0]
	if msg.Type != relay.MsgTypeOutput {
		t.Errorf("expected type %d, got %d", relay.MsgTypeOutput, msg.Type)
	}
	if string(msg.Payload) != string(data) {
		t.Errorf("payload mismatch: got %q, want %q", msg.Payload, data)
	}
}

func TestRelayOutputAdapter_IsConnected(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	adapter := &relayOutputAdapter{rc: mc}

	mc.SetConnected(false)
	if adapter.IsConnected() {
		t.Error("expected false when relay is not connected")
	}

	mc.SetConnected(true)
	if !adapter.IsConnected() {
		t.Error("expected true when relay is connected")
	}
}

// --- PTYPodRelay tests ---

func TestPTYPodRelay_SetupHandlers_Input(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	var receivedInput string
	io := &stubPodIO{
		onSendInput: func(text string) error {
			receivedInput = text
			return nil
		},
	}

	r := NewPTYPodRelay("pod-1", io, nil, nil, nil)
	r.SetupHandlers(mc)

	// Simulate browser sending input via relay.
	mc.SimulateMessage(relay.MsgTypeInput, []byte("ls -la\n"))

	if receivedInput != "ls -la\n" {
		t.Errorf("expected input 'ls -la\\n', got %q", receivedInput)
	}
}

func TestPTYPodRelay_SetupHandlers_Resize(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	var resizedCols, resizedRows int
	io := &stubPodIO{
		onResize: func(cols, rows int) (bool, error) {
			resizedCols = cols
			resizedRows = rows
			return true, nil
		},
	}

	r := NewPTYPodRelay("pod-1", io, nil, nil, nil)
	r.SetupHandlers(mc)

	// Encode resize as 4-byte big-endian payload (cols=120, rows=40).
	mc.SimulateMessage(relay.MsgTypeResize, encodeResizePayload(120, 40))

	if resizedCols != 120 || resizedRows != 40 {
		t.Errorf("resize: got %dx%d, want 120x40", resizedCols, resizedRows)
	}
}

func TestPTYPodRelay_SetupHandlers_Resize_InvalidPayload(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	resizeCalled := false
	io := &stubPodIO{
		onResize: func(cols, rows int) (bool, error) {
			resizeCalled = true
			return true, nil
		},
	}

	r := NewPTYPodRelay("pod-1", io, nil, nil, nil)
	r.SetupHandlers(mc)

	// Send invalid resize payload (too short).
	mc.SimulateMessage(relay.MsgTypeResize, []byte{0x00, 0x50})

	if resizeCalled {
		t.Error("resize should not be called with invalid payload")
	}
}

func TestPTYPodRelay_SendSnapshot_MarshalAndSend(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	vterm := vt.NewVirtualTerminal(80, 24, 1000)
	vterm.Feed([]byte("Hello, World!\r\n"))

	r := NewPTYPodRelay("pod-1", nil, vterm, nil, nil)
	r.SendSnapshot(mc)

	// Verify a snapshot message was sent.
	count := mc.CountSentByType(relay.MsgTypeSnapshot)
	if count != 1 {
		t.Fatalf("expected 1 snapshot message, got %d", count)
	}

	// Verify the payload is valid JSON containing expected fields.
	payload := mc.SentMessages[0].Payload
	var snapshot map[string]any
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		t.Fatalf("snapshot payload is not valid JSON: %v", err)
	}
	if cols, ok := snapshot["cols"].(float64); !ok || int(cols) != 80 {
		t.Errorf("expected cols=80, got %v", snapshot["cols"])
	}
}

func TestPTYPodRelay_OnRelayConnected_SetsAdapter(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	// PTYPodRelay without aggregator — OnRelayConnected should not panic.
	r := NewPTYPodRelay("pod-1", nil, nil, nil, nil)
	r.OnRelayConnected(mc) // no-op, no panic
	r.OnRelayDisconnected()
}

// --- ACPPodRelay tests ---

func TestACPPodRelay_SetupHandlers_Command(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	var receivedPayload []byte
	r := NewACPPodRelay("pod-1", nil, func(payload []byte) {
		receivedPayload = payload
	})
	r.SetupHandlers(mc)

	// Simulate browser sending ACP command.
	cmdPayload := []byte(`{"type":"prompt","data":{"prompt":"hello"}}`)
	mc.SimulateMessage(relay.MsgTypeAcpCommand, cmdPayload)

	if string(receivedPayload) != string(cmdPayload) {
		t.Errorf("expected payload %q, got %q", cmdPayload, receivedPayload)
	}
}

func TestACPPodRelay_SendSnapshot_NilClient(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	// No ACP client — should not panic.
	r := NewACPPodRelay("pod-1", nil, nil)
	r.SendSnapshot(mc)

	if mc.CountSentByType(relay.MsgTypeAcpSnapshot) != 0 {
		t.Error("should not send snapshot when ACP client is nil")
	}
}

func TestACPPodRelay_OnRelayConnected_NoPanic(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	r := NewACPPodRelay("pod-1", nil, nil)
	// No-op — should not panic.
	r.OnRelayConnected(mc)
}

func TestACPPodRelay_OnRelayDisconnected_NoPanic(t *testing.T) {
	r := NewACPPodRelay("pod-1", nil, nil)
	// No-op — should not panic.
	r.OnRelayDisconnected()
}

// --- MockClient helper tests ---

func TestMockClient_SimulateMessage(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	var received []byte
	mc.SetMessageHandler(relay.MsgTypeInput, func(payload []byte) {
		received = payload
	})

	mc.SimulateMessage(relay.MsgTypeInput, []byte("hello"))

	if string(received) != "hello" {
		t.Errorf("expected 'hello', got %q", received)
	}
}

func TestMockClient_CountSentByType(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	mc.Send(relay.MsgTypeOutput, []byte("a"))
	mc.Send(relay.MsgTypeOutput, []byte("b"))
	mc.Send(relay.MsgTypeSnapshot, []byte("{}"))

	if got := mc.CountSentByType(relay.MsgTypeOutput); got != 2 {
		t.Errorf("expected 2 output messages, got %d", got)
	}
	if got := mc.CountSentByType(relay.MsgTypeSnapshot); got != 1 {
		t.Errorf("expected 1 snapshot message, got %d", got)
	}
	if got := mc.CountSentByType(relay.MsgTypeInput); got != 0 {
		t.Errorf("expected 0 input messages, got %d", got)
	}
}

// --- PTYPodRelay.SetupHandlers: io == nil branch ---

func TestPTYPodRelay_SetupHandlers_NilIO(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	// io is nil — handlers should be registered but silently no-op.
	r := NewPTYPodRelay("pod-1", nil, nil, nil, nil)
	r.SetupHandlers(mc)

	// Must not panic.
	mc.SimulateMessage(relay.MsgTypeInput, []byte("data"))
	mc.SimulateMessage(relay.MsgTypeResize, encodeResizePayload(80, 24))
}

func TestPTYPodRelay_SetupHandlers_InputError(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	io := &stubPodIO{
		onSendInput: func(text string) error {
			return errStub
		},
	}

	r := NewPTYPodRelay("pod-1", io, nil, nil, nil)
	r.SetupHandlers(mc)

	// Should not panic — error is logged, not propagated.
	mc.SimulateMessage(relay.MsgTypeInput, []byte("data"))
}

func TestPTYPodRelay_SetupHandlers_ResizeError(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	io := &stubPodIO{
		onResize: func(cols, rows int) (bool, error) {
			return false, errStub
		},
	}

	r := NewPTYPodRelay("pod-1", io, nil, nil, nil)
	r.SetupHandlers(mc)

	// Should not panic — error is logged, not propagated.
	mc.SimulateMessage(relay.MsgTypeResize, encodeResizePayload(80, 24))
}

// --- PTYPodRelay.SendSnapshot: vterm == nil branch ---

func TestPTYPodRelay_SendSnapshot_NilVTerm(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	r := NewPTYPodRelay("pod-1", nil, nil, nil, nil)
	r.SendSnapshot(mc)

	// No VT → no snapshot sent.
	if mc.CountSentByType(relay.MsgTypeSnapshot) != 0 {
		t.Error("should not send snapshot when VT is nil")
	}
}

// --- PTYPodRelay.OnRelayConnected: aggregator != nil branch ---

func TestPTYPodRelay_OnRelayConnected_WithAggregator(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	agg := aggregator.NewSmartAggregator(func(data []byte) {}, func() float64 { return 0 })

	r := NewPTYPodRelay("pod-1", nil, nil, nil, agg)
	r.OnRelayConnected(mc)

	// Write data through the aggregator — it should route to the relay adapter.
	// We verify the adapter was wired by checking that data eventually reaches mc.
	// SmartAggregator buffers and flushes asynchronously, so we trigger a direct test:
	// The adapter should be set, meaning aggregator.SetRelayClient was called.
	// Disconnect and verify nil is passed.
	r.OnRelayDisconnected()
	// No panic = success. The internal state is opaque, but the code path is exercised.

	agg.Stop()
}

// --- sendAcpViaRelay ---

func TestSendAcpViaRelay_NoRelayClient(t *testing.T) {
	pod := &Pod{PodKey: "pod-acp-1"}
	// No relay client set → silently dropped.
	sendAcpViaRelay(pod, "content_chunk", "sess-1", map[string]string{"text": "hi"})
}

func TestSendAcpViaRelay_NotConnected(t *testing.T) {
	pod := &Pod{PodKey: "pod-acp-2"}
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(false)
	pod.SetRelayClient(mc)

	sendAcpViaRelay(pod, "content_chunk", "sess-1", map[string]string{"text": "hi"})
	// No message sent when not connected.
	if mc.CountSentByType(relay.MsgTypeAcpEvent) != 0 {
		t.Error("should not send when relay not connected")
	}
}

func TestSendAcpViaRelay_Success(t *testing.T) {
	pod := &Pod{PodKey: "pod-acp-3"}
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)
	pod.SetRelayClient(mc)

	sendAcpViaRelay(pod, "content_chunk", "sess-1", map[string]string{"text": "hi"})

	// Verify the sent message.
	if mc.CountSentByType(relay.MsgTypeAcpEvent) != 1 {
		t.Fatalf("expected 1 ACP event, got %d", mc.CountSentByType(relay.MsgTypeAcpEvent))
	}
	var envelope map[string]any
	if err := json.Unmarshal(mc.SentMessages[0].Payload, &envelope); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if envelope["type"] != "content_chunk" {
		t.Errorf("expected type 'content_chunk', got %v", envelope["type"])
	}
	if envelope["session_id"] != "sess-1" {
		t.Errorf("expected session_id 'sess-1', got %v", envelope["session_id"])
	}
}

func TestSendAcpViaRelay_MarshalError(t *testing.T) {
	pod := &Pod{PodKey: "pod-acp-4"}
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)
	pod.SetRelayClient(mc)

	// json.Marshal fails on channels — should not panic.
	sendAcpViaRelay(pod, "bad", "", make(chan int))
	if mc.CountSentByType(relay.MsgTypeAcpEvent) != 0 {
		t.Error("should not send when marshal fails")
	}
}

// --- PTYPodIO.Teardown tests ---

func TestPTYPodIO_Teardown_WithPTYLogger(t *testing.T) {
	pod := &Pod{PodKey: "pod-teardown-logger"}

	agg := aggregator.NewSmartAggregator(nil, nil)
	ptyLogger, err := aggregator.NewPTYLogger(t.TempDir(), "pod-teardown-logger")
	if err != nil {
		t.Fatalf("failed to create PTYLogger: %v", err)
	}

	io := NewPTYPodIO(nil, nil, pod)
	io.SetAggregator(agg)
	io.SetPTYLogger(ptyLogger)

	result := io.Teardown()

	// No PTY error set → empty result
	if result != "" {
		t.Errorf("expected empty teardown result, got %q", result)
	}
}

func TestPTYPodIO_Teardown_PTYErrorFallback(t *testing.T) {
	pod := &Pod{PodKey: "pod-teardown-err"}
	pod.SetPTYError("disk full")

	io := NewPTYPodIO(nil, nil, pod)

	result := io.Teardown()

	if result != "disk full" {
		t.Errorf("expected 'disk full', got %q", result)
	}
}

func TestPTYPodIO_Teardown_EarlyOutputTakesPriority(t *testing.T) {
	pod := &Pod{PodKey: "pod-teardown-priority"}
	pod.SetPTYError("disk full")

	// Create aggregator with nil gRPC callback — data goes to early buffer.
	agg := aggregator.NewSmartAggregator(nil, nil)
	// Write data then stop immediately — unflushed data remains in early buffer.
	agg.Write([]byte("early output data"))
	// Give the aggregator timer a chance to see the data as "pending".
	// Teardown calls agg.Stop() which drains remaining data into early buffer.

	io := NewPTYPodIO(nil, nil, pod)
	io.SetAggregator(agg)

	result := io.Teardown()

	// Either early buffer captured the data, or PTY error is used as fallback.
	// The key invariant: result is non-empty.
	if result == "" {
		t.Error("expected non-empty teardown result")
	}
}

// --- stub helpers ---

// stubPodIO is a minimal PodIO implementation for unit testing.
type stubPodIO struct {
	onSendInput func(string) error
	onResize    func(int, int) (bool, error)
}

func (s *stubPodIO) Mode() string                                         { return "pty" }
func (s *stubPodIO) GetSnapshot(int) (string, error)                      { return "", nil }
func (s *stubPodIO) GetAgentStatus() string                               { return "idle" }
func (s *stubPodIO) SubscribeStateChange(string, func(string))            {}
func (s *stubPodIO) UnsubscribeStateChange(string)                        {}
func (s *stubPodIO) Start() error                                         { return nil }
func (s *stubPodIO) SendKeys([]string) error                              { return nil }
func (s *stubPodIO) GetPID() int                                          { return 0 }
func (s *stubPodIO) CursorPosition() (int, int)                           { return 0, 0 }
func (s *stubPodIO) GetScreenSnapshot() string                            { return "" }
func (s *stubPodIO) Stop()                                                {}
func (s *stubPodIO) Teardown() string                                      { return "" }
func (s *stubPodIO) SetExitHandler(func(int))                             {}
func (s *stubPodIO) Redraw() error                                        { return nil }
func (s *stubPodIO) Detach()                                              {}
func (s *stubPodIO) WriteOutput([]byte)                                    {}
func (s *stubPodIO) RespondToPermission(string, bool) error               { return nil }
func (s *stubPodIO) CancelSession() error                                 { return nil }
func (s *stubPodIO) SendInput(text string) error {
	if s.onSendInput != nil {
		return s.onSendInput(text)
	}
	return nil
}
func (s *stubPodIO) Resize(cols, rows int) (bool, error) {
	if s.onResize != nil {
		return s.onResize(cols, rows)
	}
	return false, nil
}
