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
	io := NewPTYPodIO("test", &PTYComponents{}, PTYPodIODeps{})
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

	r := NewPTYPodRelay("pod-1", io, &PTYComponents{})
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
	io := &stubPodIOWithTerminal{
		onResize: func(cols, rows int) (bool, error) {
			resizedCols = cols
			resizedRows = rows
			return true, nil
		},
	}

	r := NewPTYPodRelay("pod-1", io, &PTYComponents{})
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
	io := &stubPodIOWithTerminal{
		onResize: func(cols, rows int) (bool, error) {
			resizeCalled = true
			return true, nil
		},
	}

	r := NewPTYPodRelay("pod-1", io, &PTYComponents{})
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

	r := NewPTYPodRelay("pod-1", nil, &PTYComponents{VirtualTerminal: vterm})
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
	r := NewPTYPodRelay("pod-1", nil, &PTYComponents{})
	r.OnRelayConnected(mc) // no-op, no panic
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
	r := NewPTYPodRelay("pod-1", nil, &PTYComponents{})
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

	r := NewPTYPodRelay("pod-1", io, &PTYComponents{})
	r.SetupHandlers(mc)

	// Should not panic — error is logged, not propagated.
	mc.SimulateMessage(relay.MsgTypeInput, []byte("data"))
}

func TestPTYPodRelay_SetupHandlers_ResizeError(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")

	io := &stubPodIOWithTerminal{
		onResize: func(cols, rows int) (bool, error) {
			return false, errStub
		},
	}

	r := NewPTYPodRelay("pod-1", io, &PTYComponents{})
	r.SetupHandlers(mc)

	// Should not panic — error is logged, not propagated.
	mc.SimulateMessage(relay.MsgTypeResize, encodeResizePayload(80, 24))
}

// --- PTYPodRelay.SendSnapshot: vterm == nil branch ---

func TestPTYPodRelay_SendSnapshot_NilVTerm(t *testing.T) {
	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	r := NewPTYPodRelay("pod-1", nil, &PTYComponents{})
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

	agg := aggregator.NewSmartAggregator(func() float64 { return 0 })

	r := NewPTYPodRelay("pod-1", nil, &PTYComponents{Aggregator: agg})
	r.OnRelayConnected(mc)

	r.OnRelayDisconnected()

	agg.Stop()
}

// --- stub helpers ---

// stubPodIO is a minimal PodIO implementation for unit testing.
type stubPodIO struct {
	onSendInput func(string) error
}

func (s *stubPodIO) Mode() string                              { return "pty" }
func (s *stubPodIO) GetSnapshot(int) (string, error)           { return "", nil }
func (s *stubPodIO) GetAgentStatus() string                    { return "idle" }
func (s *stubPodIO) SubscribeStateChange(string, func(string)) {}
func (s *stubPodIO) UnsubscribeStateChange(string)             {}
func (s *stubPodIO) GetPID() int                               { return 0 }
func (s *stubPodIO) Stop()                                     {}
func (s *stubPodIO) Teardown() string                          { return "" }
func (s *stubPodIO) SetExitHandler(func(int))                  {}
func (s *stubPodIO) Detach()                                   {}
func (s *stubPodIO) Start() error                              { return nil }
func (s *stubPodIO) SetIOErrorHandler(func(error))             {}
func (s *stubPodIO) SendInput(text string) error {
	if s.onSendInput != nil {
		return s.onSendInput(text)
	}
	return nil
}

// stubPodIOWithTerminal extends stubPodIO with TerminalAccess for resize tests.
type stubPodIOWithTerminal struct {
	stubPodIO
	onResize func(int, int) (bool, error)
}

func (s *stubPodIOWithTerminal) SendKeys([]string) error    { return nil }
func (s *stubPodIOWithTerminal) CursorPosition() (int, int) { return 0, 0 }
func (s *stubPodIOWithTerminal) GetScreenSnapshot() string  { return "" }
func (s *stubPodIOWithTerminal) Redraw() error              { return nil }
func (s *stubPodIOWithTerminal) WriteOutput([]byte)         {}
func (s *stubPodIOWithTerminal) Resize(cols, rows int) (bool, error) {
	if s.onResize != nil {
		return s.onResize(cols, rows)
	}
	return false, nil
}
