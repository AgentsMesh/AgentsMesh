package relay

import (
	"errors"
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
	MsgTypeResync             = 0x0A // Browser → Runner, request full state resync
	MsgTypeAcpEvent           = 0x0B // Runner → Browser, ACP session event
	MsgTypeAcpCommand         = 0x0C // Browser → Runner, ACP command
	MsgTypeAcpSnapshot        = 0x0D // Runner → Browser, ACP session snapshot
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

// EncodePing encodes a ping message
func EncodePing() []byte {
	return EncodeMessage(MsgTypePing, nil)
}

// EncodePong encodes a pong message
func EncodePong() []byte {
	return EncodeMessage(MsgTypePong, nil)
}
