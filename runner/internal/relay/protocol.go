package relay

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// Message types for Relay protocol
// These must match relay/internal/protocol/message.go
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


// ResizeMessage represents a terminal resize request
type ResizeMessage struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
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
func EncodeSnapshot(snapshot *vt.TerminalSnapshot) ([]byte, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return EncodeMessage(MsgTypeSnapshot, payload), nil
}

// EncodeOutput encodes terminal output data
func EncodeOutput(data []byte) []byte {
	return EncodeMessage(MsgTypeOutput, data)
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
