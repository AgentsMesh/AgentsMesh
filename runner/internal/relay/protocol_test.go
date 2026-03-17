package relay

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func TestEncodeDecodeMessage(t *testing.T) {
	tests := []struct {
		name     string
		msgType  byte
		payload  []byte
	}{
		{"output", MsgTypeOutput, []byte("hello")},
		{"ping", MsgTypePing, nil},
		{"pong", MsgTypePong, nil},
		{"empty_payload", MsgTypeOutput, []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeMessage(tt.msgType, tt.payload)
			msg, err := DecodeMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeMessage: %v", err)
			}
			if msg.Type != tt.msgType {
				t.Errorf("type: got %d, want %d", msg.Type, tt.msgType)
			}
			if string(msg.Payload) != string(tt.payload) {
				t.Errorf("payload mismatch")
			}
		})
	}
}

func TestDecodeMessageEmpty(t *testing.T) {
	_, err := DecodeMessage(nil)
	if err != ErrEmptyMessage {
		t.Errorf("expected ErrEmptyMessage, got: %v", err)
	}
	_, err = DecodeMessage([]byte{})
	if err != ErrEmptyMessage {
		t.Errorf("expected ErrEmptyMessage, got: %v", err)
	}
}

func TestEncodeOutput(t *testing.T) {
	data := []byte("test output")
	encoded := EncodeOutput(data)
	if encoded[0] != MsgTypeOutput {
		t.Errorf("type: got %d, want %d", encoded[0], MsgTypeOutput)
	}
	if string(encoded[1:]) != string(data) {
		t.Error("payload mismatch")
	}
}

func TestEncodeDecodeResize(t *testing.T) {
	encoded := EncodeResize(120, 40)
	if encoded[0] != MsgTypeResize {
		t.Errorf("type: got %d, want %d", encoded[0], MsgTypeResize)
	}
	cols, rows, err := DecodeResize(encoded[1:])
	if err != nil {
		t.Fatalf("DecodeResize: %v", err)
	}
	if cols != 120 || rows != 40 {
		t.Errorf("got %dx%d, want 120x40", cols, rows)
	}
}

func TestDecodeResizeInvalid(t *testing.T) {
	_, _, err := DecodeResize([]byte{1, 2})
	if err != ErrInvalidMessage {
		t.Errorf("expected ErrInvalidMessage, got: %v", err)
	}
}

func TestEncodePingPong(t *testing.T) {
	ping := EncodePing()
	if ping[0] != MsgTypePing || len(ping) != 1 {
		t.Errorf("ping: got type=%d len=%d", ping[0], len(ping))
	}
	pong := EncodePong()
	if pong[0] != MsgTypePong || len(pong) != 1 {
		t.Errorf("pong: got type=%d len=%d", pong[0], len(pong))
	}
}

func TestEncodeSnapshot(t *testing.T) {
	snapshot := &vt.TerminalSnapshot{
		Cols:              80,
		Rows:              24,
		Lines:             []string{"line1", "line2"},
		SerializedContent: "test content",
		CursorX:           0,
		CursorY:           0,
		CursorVisible:     true,
		IsAltScreen:       false,
	}
	encoded, err := EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatalf("EncodeSnapshot: %v", err)
	}
	if encoded[0] != MsgTypeSnapshot {
		t.Errorf("type: got %d, want %d", encoded[0], MsgTypeSnapshot)
	}
}

func TestMessageConstants(t *testing.T) {
	// Verify constants match expected values
	if MsgTypeSnapshot != 0x01 {
		t.Error("MsgTypeSnapshot")
	}
	if MsgTypeOutput != 0x02 {
		t.Error("MsgTypeOutput")
	}
	if MsgTypeInput != 0x03 {
		t.Error("MsgTypeInput")
	}
	if MsgTypeResize != 0x04 {
		t.Error("MsgTypeResize")
	}
	if MsgTypePing != 0x05 {
		t.Error("MsgTypePing")
	}
	if MsgTypePong != 0x06 {
		t.Error("MsgTypePong")
	}
	if MsgTypeControl != 0x07 {
		t.Error("MsgTypeControl")
	}
	if MsgTypeRunnerDisconnected != 0x08 {
		t.Error("MsgTypeRunnerDisconnected")
	}
	if MsgTypeRunnerReconnected != 0x09 {
		t.Error("MsgTypeRunnerReconnected")
	}
}
