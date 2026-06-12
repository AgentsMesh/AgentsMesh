package acp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// Internal helpers are in request_tracker.go (shared with Codex transport).
// Session-level methods (NewSession / SendPrompt / RespondToPermission /
// CancelSession) live in transport_acp_session.go.
// SendControlRequest + capability negotiation live in transport_acp_control.go.

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

	// supportsControlRequest captures whether the agent advertised the
	// `agentsmeshExtensions.controlRequest` capability in its initialize
	// response. When false (codex/gemini/opencode + any other standard
	// ACP agent), SendControlRequest fails fast with ErrControlNotSupported
	// instead of waiting on a 10-second JSON-RPC timeout.
	supportsControlRequest bool

	// supportedPermissionModes captures agentsmeshExtensions.permissionModes from
	// the initialize response — the wire values this agent accepts for
	// set_permission_mode. Empty means the agent didn't advertise (frontend falls
	// back to the Claude default set).
	supportedPermissionModes []string

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
		"protocolVersion": 1,
		"clientInfo": map[string]any{
			"name":    "agentsmesh-runner",
			"version": "1.0.0",
		},
		"clientCapabilities": map[string]any{},
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

	t.supportsControlRequest, t.supportedPermissionModes = parseAgentsmeshExtensions(resp.Result)
	t.logger.Info("ACP initialize succeeded",
		"supports_control_request", t.supportsControlRequest,
		"permission_modes", t.supportedPermissionModes)
	return "", nil // ACP doesn't auto-discover session IDs
}

// NewSession / SendPrompt / RespondToPermission / CancelSession live in
// transport_acp_session.go.
// SendControlRequest lives in transport_acp_control.go.

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

// SupportedPermissionModes returns the permission-mode wire values advertised in
// the initialize response (nil if none).
func (t *ACPTransport) SupportedPermissionModes() []string {
	return t.supportedPermissionModes
}

func (t *ACPTransport) dispatchMessage(msg *JSONRPCMessage) {
	switch {
	case msg.IsResponse():
		t.tracker.HandleResponse(msg)
	case msg.IsNotification():
		t.handler.HandleNotification(msg.Method, msg.Params)
	case msg.IsRequest():
		// Standard ACP: agent sends session/request_permission as a request.
		if msg.Method == "session/request_permission" {
			id, _ := msg.GetID()
			t.handler.HandlePermissionRequest(id, msg.Params)
		} else {
			t.tracker.RejectRequest(msg)
		}
	}
}
