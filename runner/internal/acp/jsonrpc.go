package acp

import (
	"encoding/json"
	"sync/atomic"
)

// JSON-RPC 2.0 message types.

// JSONRPCRequest is a JSON-RPC 2.0 request (has method + id).
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response (has id, result or error).
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCNotification is a JSON-RPC 2.0 notification (has method, no id).
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// nextID is a monotonically increasing request ID generator.
var nextID atomic.Int64

// NextRequestID returns a new unique request ID.
func NextRequestID() int64 {
	return nextID.Add(1)
}

// JSONRPCMessage is a raw inbound message that can be a request,
// response, or notification. We determine the type by inspecting
// the "method" and "id" fields.
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// IsRequest returns true when the message has both method and id.
func (m *JSONRPCMessage) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// IsNotification returns true when the message has method but no id.
func (m *JSONRPCMessage) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// IsResponse returns true when the message has id but no method.
func (m *JSONRPCMessage) IsResponse() bool {
	return m.Method == "" && m.ID != nil
}

// GetID extracts the numeric ID from the raw JSON field.
func (m *JSONRPCMessage) GetID() (int64, bool) {
	if m.ID == nil {
		return 0, false
	}
	var id int64
	if err := json.Unmarshal(*m.ID, &id); err != nil {
		return 0, false
	}
	return id, true
}
