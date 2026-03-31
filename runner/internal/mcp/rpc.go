package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// call makes an RPC call and waits for response
func (s *Server) call(ctx context.Context, method string, params interface{}) (*Response, error) {
	s.mu.Lock()
	s.requestID++
	id := s.requestID
	respChan := make(chan *Response, 1)
	s.pending[id] = respChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	// Build request
	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}

	// Send request
	if err := s.send(&req); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-respChan:
		if !ok {
			return nil, fmt.Errorf("server closed")
		}
		return resp, nil
	}
}

// notify sends a notification (no response expected)
func (s *Server) notify(method string, params interface{}) error {
	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	req := Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	return s.send(&req)
}

// send writes a request to the server
func (s *Server) send(req *Request) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("server not running")
	}

	if s.stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// MCP uses newline-delimited JSON
	data = append(data, '\n')

	if _, err := s.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

// readResponses reads responses from the server
func (s *Server) readResponses() {
	defer s.readerDone.Done()
	decoder := json.NewDecoder(s.stdout)

	for {
		var resp Response
		if err := decoder.Decode(&resp); err != nil {
			// Exit on any read error (EOF, closed pipe, etc.)
			// JSON syntax errors from partial reads also indicate the stream is done.
			return
		}

		// Route response to waiting caller — remove from pending under lock
		// before sending, so Stop() won't close a channel we're about to use.
		s.mu.Lock()
		ch, ok := s.pending[resp.ID]
		if ok {
			delete(s.pending, resp.ID)
		}
		s.mu.Unlock()
		if ok {
			ch <- &resp
		}
	}
}
