package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// Internal helpers are in request_tracker.go (shared with Codex transport).

func init() {
	RegisterTransport(TransportTypeACP, func(cb EventCallbacks, l *slog.Logger) Transport {
		return NewACPTransport(cb, l)
	})
}

// ACPTransport implements the Transport interface for the standard
// ACP JSON-RPC 2.0 protocol (used by Gemini CLI --acp, OpenCode acp).
type ACPTransport struct {
	tracker *RequestTracker
	reader  *Reader
	handler *Handler

	ctx    context.Context
	logger *slog.Logger
}

// NewACPTransport creates a new JSON-RPC 2.0 transport.
func NewACPTransport(callbacks EventCallbacks, logger *slog.Logger) *ACPTransport {
	return &ACPTransport{
		handler: NewHandler(callbacks, logger),
		logger:  logger,
	}
}

// Initialize wires the transport's I/O pipes.
func (t *ACPTransport) Initialize(ctx context.Context, stdin io.Writer, stdout io.Reader, _ io.Reader) error {
	t.ctx = ctx
	writer := NewWriter(stdin)
	t.reader = NewReader(stdout, t.logger)
	t.tracker = NewRequestTracker(writer, t.logger, func() <-chan struct{} { return ctx.Done() })
	return nil
}

// Handshake performs the ACP JSON-RPC initialize handshake.
// Must be called after ReadLoop is started.
func (t *ACPTransport) Handshake(_ context.Context) (string, error) {
	params := map[string]any{
		"protocol_version": "2025-01-01",
		"client_info": map[string]any{
			"name":    "agentsmesh-runner",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{
			"permissions": true,
		},
	}

	pr, err := t.tracker.SendRequest("initialize", params)
	if err != nil {
		return "", fmt.Errorf("write initialize: %w", err)
	}

	resp, err := t.tracker.WaitResponse(pr, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("wait initialize response: %w", err)
	}

	if resp.Error != nil {
		return "", fmt.Errorf("initialize error: code=%d msg=%s",
			resp.Error.Code, resp.Error.Message)
	}

	t.logger.Info("ACP initialize succeeded")
	return "", nil // ACP doesn't auto-discover session IDs
}

// NewSession sends a session/new RPC and returns the session ID.
func (t *ACPTransport) NewSession(mcpServers map[string]any) (string, error) {
	params := map[string]any{}
	if mcpServers != nil {
		params["mcp_servers"] = mcpServers
	}

	pr, err := t.tracker.SendRequest("session/new", params)
	if err != nil {
		return "", fmt.Errorf("write session/new: %w", err)
	}

	resp, err := t.tracker.WaitResponse(pr, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("wait session/new response: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("session/new error: code=%d msg=%s",
			resp.Error.Code, resp.Error.Message)
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("parse session/new result: %w", err)
	}

	return result.SessionID, nil
}

// SendPrompt sends a session/prompt RPC request. Non-blocking: the actual
// content arrives via session/update notifications in ReadLoop.
func (t *ACPTransport) SendPrompt(sessionID, prompt string) error {
	params := map[string]any{
		"session_id": sessionID,
		"prompt":     prompt,
	}

	pr, err := t.tracker.SendRequest("session/prompt", params)
	if err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	// Wait for the RPC response in the background;
	// actual content arrives via session/update notifications.
	go func() {
		resp, err := t.tracker.WaitResponse(pr, 5*time.Minute)
		if err != nil {
			t.logger.Error("prompt response error", "error", err)
		} else if resp.Error != nil {
			t.logger.Error("prompt error",
				"code", resp.Error.Code, "message", resp.Error.Message)
		}
	}()

	return nil
}

// RespondToPermission sends a permission/response notification.
func (t *ACPTransport) RespondToPermission(requestID string, approved bool) error {
	params := map[string]any{
		"request_id": requestID,
		"approved":   approved,
	}
	return t.tracker.Writer.WriteNotification("permission/response", params)
}

// CancelSession sends a session/cancel notification.
func (t *ACPTransport) CancelSession(sessionID string) error {
	params := map[string]any{
		"session_id": sessionID,
	}
	return t.tracker.Writer.WriteNotification("session/cancel", params)
}

// ReadLoop reads JSON-RPC messages from stdout and dispatches them.
func (t *ACPTransport) ReadLoop(ctx context.Context) {
	for {
		msg, err := t.reader.ReadMessage()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				t.logger.Error("read error", "error", err)
				return
			}
		}
		t.dispatchMessage(msg)
	}
}

// Close is a no-op for ACPTransport (resources owned by ACPClient).
func (t *ACPTransport) Close() {}

func (t *ACPTransport) dispatchMessage(msg *JSONRPCMessage) {
	switch {
	case msg.IsResponse():
		t.tracker.HandleResponse(msg)
	case msg.IsNotification():
		t.handler.HandleNotification(msg.Method, msg.Params)
	case msg.IsRequest():
		t.tracker.RejectRequest(msg)
	}
}
