package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
)

// Message types for Relay protocol
const (
	MsgTypeSnapshot           = 0x01 // Complete terminal snapshot
	MsgTypeOutput             = 0x02 // Terminal output data
	MsgTypeInput              = 0x03 // User input
	MsgTypeResize             = 0x04 // Terminal resize
	MsgTypePing               = 0x05 // Heartbeat ping
	MsgTypePong               = 0x06 // Heartbeat pong
	MsgTypeControl            = 0x07 // Control request (for input control)
	MsgTypeRunnerDisconnected = 0x08 // Runner disconnected notification
	MsgTypeRunnerReconnected  = 0x09 // Runner reconnected notification
	MsgTypeResync             = 0x0A // Resync request (browser → runner, request full state)

	// ACP (Agent Communication Protocol) message types
	MsgTypeAcpEvent    = 0x0B // ACP event (ephemeral, not buffered)
	MsgTypeAcpCommand  = 0x0C // ACP command (browser → runner)
	MsgTypeAcpSnapshot = 0x0D // ACP snapshot (complete session state)
)

var (
	ErrInvalidMessage = errors.New("invalid message format")
	ErrEmptyMessage   = errors.New("empty message")
)

// Message represents a protocol message
type Message struct {
	Type    byte
	Payload []byte
}

// TerminalSnapshot represents a complete terminal state
type TerminalSnapshot struct {
	Cols              uint16   `json:"cols"`
	Rows              uint16   `json:"rows"`
	Lines             []string `json:"lines"`               // Plain text lines (kept for compatibility)
	SerializedContent string   `json:"serialized_content"`  // ANSI-escaped serialized content for xterm.js
	CursorX           int      `json:"cursor_x"`
	CursorY           int      `json:"cursor_y"`
	CursorVisible     bool     `json:"cursor_visible"`
	IsAltScreen       bool     `json:"is_alt_screen"` // Whether in alternate screen mode (TUI apps)
}

// ResizeMessage represents a terminal resize request
type ResizeMessage struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// ControlRequest represents an input control request
type ControlRequest struct {
	Action     string `json:"action"` // "request", "release", "query"
	BrowserID  string `json:"browser_id"`
	Controller string `json:"controller,omitempty"` // Current controller (in response)
}

// EncodeMessage encodes a message with type prefix
// Format: [1 byte type][payload]
func EncodeMessage(msgType byte, payload []byte) []byte {
	result := make([]byte, 1+len(payload))
	result[0] = msgType
	copy(result[1:], payload)
	return result
}

// DecodeMessage decodes a message from wire format
func DecodeMessage(data []byte) (*Message, error) {
	if len(data) < 1 {
		return nil, ErrEmptyMessage
	}
	return &Message{
		Type:    data[0],
		Payload: data[1:],
	}, nil
}

// EncodeSnapshot encodes a terminal snapshot
func EncodeSnapshot(snapshot *TerminalSnapshot) ([]byte, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return EncodeMessage(MsgTypeSnapshot, payload), nil
}

// DecodeSnapshot decodes a terminal snapshot from message payload
func DecodeSnapshot(payload []byte) (*TerminalSnapshot, error) {
	var snapshot TerminalSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// EncodeOutput encodes terminal output data
func EncodeOutput(data []byte) []byte {
	return EncodeMessage(MsgTypeOutput, data)
}

// EncodeInput encodes user input data
func EncodeInput(data []byte) []byte {
	return EncodeMessage(MsgTypeInput, data)
}

// EncodeResize encodes a resize message
func EncodeResize(cols, rows uint16) []byte {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint16(payload[0:2], cols)
	binary.BigEndian.PutUint16(payload[2:4], rows)
	return EncodeMessage(MsgTypeResize, payload)
}

// DecodeResize decodes a resize message from payload
func DecodeResize(payload []byte) (cols, rows uint16, err error) {
	if len(payload) < 4 {
		return 0, 0, ErrInvalidMessage
	}
	cols = binary.BigEndian.Uint16(payload[0:2])
	rows = binary.BigEndian.Uint16(payload[2:4])
	return cols, rows, nil
}

// EncodePing encodes a ping message
func EncodePing() []byte {
	return EncodeMessage(MsgTypePing, nil)
}

// EncodePong encodes a pong message
func EncodePong() []byte {
	return EncodeMessage(MsgTypePong, nil)
}

// EncodeControlRequest encodes a control request
func EncodeControlRequest(req *ControlRequest) ([]byte, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return EncodeMessage(MsgTypeControl, payload), nil
}

// DecodeControlRequest decodes a control request from payload
func DecodeControlRequest(payload []byte) (*ControlRequest, error) {
	var req ControlRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// EncodeRunnerDisconnected encodes a runner disconnected notification
func EncodeRunnerDisconnected() []byte {
	return EncodeMessage(MsgTypeRunnerDisconnected, nil)
}

// EncodeRunnerReconnected encodes a runner reconnected notification
func EncodeRunnerReconnected() []byte {
	return EncodeMessage(MsgTypeRunnerReconnected, nil)
}
