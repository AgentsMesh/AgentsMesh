package protocol

import (
	"encoding/binary"
	"testing"
)

func TestEncodeDecodeMessage(t *testing.T) {
	tests := []struct {
		msgType byte
		payload []byte
	}{
		{MsgTypeOutput, nil}, {MsgTypeOutput, []byte("hello")}, {MsgTypeOutput, make([]byte, 1024)},
		{MsgTypeSnapshot, []byte("{}")}, {MsgTypeInput, []byte("test")}, {MsgTypeResize, []byte{0, 80, 0, 24}},
		{MsgTypePing, nil}, {MsgTypePong, nil},
		{MsgTypeRunnerDisconnected, nil}, {MsgTypeRunnerReconnected, nil},
	}
	for _, tt := range tests {
		encoded := EncodeMessage(tt.msgType, tt.payload)
		if len(encoded) != 1+len(tt.payload) || encoded[0] != tt.msgType {
			t.Errorf("encode failed for type %d", tt.msgType)
		}
		msg, err := DecodeMessage(encoded)
		if err != nil || msg.Type != tt.msgType || len(msg.Payload) != len(tt.payload) {
			t.Errorf("decode failed for type %d", tt.msgType)
		}
	}
}

func TestDecodeMessage_EmptyData(t *testing.T) {
	if _, err := DecodeMessage([]byte{}); err != ErrEmptyMessage {
		t.Error("expected ErrEmptyMessage")
	}
	if _, err := DecodeMessage(nil); err != ErrEmptyMessage {
		t.Error("expected ErrEmptyMessage for nil")
	}
}

func TestEncodeOutputInput(t *testing.T) {
	data := []byte("test data")
	if out := EncodeOutput(data); out[0] != MsgTypeOutput {
		t.Error("EncodeOutput type wrong")
	}
	if in := EncodeInput(data); in[0] != MsgTypeInput {
		t.Error("EncodeInput type wrong")
	}
}

func TestEncodeDecodeResize(t *testing.T) {
	tests := []struct{ cols, rows uint16 }{{80, 24}, {120, 40}, {0, 0}, {65535, 65535}}
	for _, tt := range tests {
		encoded := EncodeResize(tt.cols, tt.rows)
		if encoded[0] != MsgTypeResize {
			t.Error("EncodeResize type wrong")
		}
		msg, _ := DecodeMessage(encoded)
		cols, rows, err := DecodeResize(msg.Payload)
		if err != nil || cols != tt.cols || rows != tt.rows {
			t.Errorf("resize failed: %d,%d", tt.cols, tt.rows)
		}
	}
}

func TestDecodeResize_InvalidPayload(t *testing.T) {
	for _, p := range [][]byte{{}, {0}, {0, 80}, {0, 80, 0}} {
		if _, _, err := DecodeResize(p); err != ErrInvalidMessage {
			t.Error("expected ErrInvalidMessage")
		}
	}
}

func TestEncodePingPong(t *testing.T) {
	if p := EncodePing(); p[0] != MsgTypePing || len(p) != 1 {
		t.Error("EncodePing wrong")
	}
	if p := EncodePong(); p[0] != MsgTypePong || len(p) != 1 {
		t.Error("EncodePong wrong")
	}
}

func TestEncodeRunnerDisconnectedReconnected(t *testing.T) {
	if d := EncodeRunnerDisconnected(); d[0] != MsgTypeRunnerDisconnected || len(d) != 1 {
		t.Error("EncodeRunnerDisconnected wrong")
	}
	if r := EncodeRunnerReconnected(); r[0] != MsgTypeRunnerReconnected || len(r) != 1 {
		t.Error("EncodeRunnerReconnected wrong")
	}
}

func TestMessageConstants(t *testing.T) {
	expected := map[byte]byte{
		MsgTypeSnapshot: 0x01, MsgTypeOutput: 0x02, MsgTypeInput: 0x03, MsgTypeResize: 0x04,
		MsgTypePing: 0x05, MsgTypePong: 0x06, MsgTypeControl: 0x07,
		MsgTypeRunnerDisconnected: 0x08, MsgTypeRunnerReconnected: 0x09,
		MsgTypeResync:   0x0A,
		MsgTypeAcpEvent: 0x0B, MsgTypeAcpCommand: 0x0C, MsgTypeAcpSnapshot: 0x0D,
	}
	for got, want := range expected {
		if got != want {
			t.Errorf("message type: expected 0x%02x, got 0x%02x", want, got)
		}
	}
}

func TestResizeMessageBigEndian(t *testing.T) {
	cols, rows := uint16(0x1234), uint16(0x5678)
	payload := EncodeResize(cols, rows)[1:]
	if binary.BigEndian.Uint16(payload[0:2]) != cols || binary.BigEndian.Uint16(payload[2:4]) != rows {
		t.Error("resize not in big-endian")
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrInvalidMessage.Error() != "invalid message format" {
		t.Error("ErrInvalidMessage message wrong")
	}
	if ErrEmptyMessage.Error() != "empty message" {
		t.Error("ErrEmptyMessage message wrong")
	}
}
