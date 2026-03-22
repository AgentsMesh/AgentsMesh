package acp

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Writer sends JSON-RPC messages as newline-delimited JSON.
// All writes are serialized via a mutex.
type Writer struct {
	w  io.Writer
	mu sync.Mutex
}

// NewWriter creates a Writer that sends to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteRequest sends a JSON-RPC request and returns the assigned ID.
func (w *Writer) WriteRequest(method string, params any) (int64, error) {
	id := NextRequestID()
	return id, w.WriteRequestWithID(id, method, params)
}

// WriteRequestWithID sends a JSON-RPC request with a pre-assigned ID.
// Use this when the caller needs to register a response handler before
// the request is written (to avoid response-before-register races).
func (w *Writer) WriteRequestWithID(id int64, method string, params any) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
	}
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		req.Params = data
	}
	return w.writeJSON(req)
}

// WriteNotification sends a JSON-RPC notification (no ID, no response expected).
func (w *Writer) WriteNotification(method string, params any) error {
	notif := JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
	}
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		notif.Params = data
	}
	return w.writeJSON(notif)
}

// WriteResponse sends a JSON-RPC response for the given request ID.
func (w *Writer) WriteResponse(id int64, result any, rpcErr *JSONRPCError) error {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      &id,
		Error:   rpcErr,
	}
	if result != nil && rpcErr == nil {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		resp.Result = data
	}
	return w.writeJSON(resp)
}

func (w *Writer) writeJSON(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	data = append(data, '\n')

	_, err = w.w.Write(data)
	return err
}
