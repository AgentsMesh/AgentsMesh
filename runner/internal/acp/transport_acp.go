package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
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

	// supportsControlRequest captures whether the agent advertised the
	// `agentsmeshExtensions.controlRequest` capability in its initialize
	// response. When false (codex/gemini/opencode + any other standard
	// ACP agent), SendControlRequest fails fast with ErrControlNotSupported
	// instead of waiting on a 10-second JSON-RPC timeout.
	supportsControlRequest bool

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

	t.supportsControlRequest = parseControlRequestCapability(resp.Result)
	t.logger.Info("ACP initialize succeeded",
		"supports_control_request", t.supportsControlRequest)
	return "", nil // ACP doesn't auto-discover session IDs
}

// parseControlRequestCapability reads the agentsmeshExtensions block from an
// initialize response. Agents that don't ship this extension simply omit it,
// which (correctly) leaves the runner in "no control_request" mode.
func parseControlRequestCapability(raw json.RawMessage) bool {
	var body struct {
		AgentsmeshExtensions struct {
			ControlRequest bool `json:"controlRequest"`
		} `json:"agentsmeshExtensions"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}
	return body.AgentsmeshExtensions.ControlRequest
}

// NewSession sends a session/new RPC and returns the session ID.
func (t *ACPTransport) NewSession(cwd string, mcpServers map[string]any) (string, error) {
	// Both cwd and mcpServers are required by the ACP spec.
	var servers []map[string]any
	// Convert map format {name: {type,url}} to ACP array format [{name,type,url}].
	for name, cfg := range mcpServers {
		entry := map[string]any{"name": name}
		if m, ok := cfg.(map[string]any); ok {
			for k, v := range m {
				entry[k] = v
			}
		}
		servers = append(servers, entry)
	}
	if servers == nil {
		servers = []map[string]any{}
	}
	params := map[string]any{
		"cwd":        cwd,
		"mcpServers": servers,
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
		SessionID string `json:"sessionId"`
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
		"sessionId": sessionID,
		"prompt": []map[string]any{
			{"type": "text", "text": prompt},
		},
	}

	pr, err := t.tracker.SendRequest("session/prompt", params)
	if err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	// Wait for the RPC response in the background;
	// actual content arrives via session/update notifications.
	// When the response arrives, transition to idle (standard ACP has no session/complete).
	go func() {
		resp, err := t.tracker.WaitResponse(pr, 5*time.Minute)
		if err != nil {
			t.logger.Error("prompt response error", "error", err)
		} else if resp.Error != nil {
			t.logger.Error("prompt error",
				"code", resp.Error.Code, "message", resp.Error.Message)
		}
		// Prompt response means the agent finished this turn.
		if t.handler.callbacks.OnStateChange != nil {
			t.handler.callbacks.OnStateChange(StateIdle)
		}
	}()

	return nil
}

// RespondToPermission sends a JSON-RPC response to a session/request_permission request.
// The requestID is the string-encoded JSON-RPC request id.
func (t *ACPTransport) RespondToPermission(requestID string, approved bool, _ map[string]any) error {
	rpcID, err := strconv.ParseInt(requestID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid permission request ID %q: %w", requestID, err)
	}

	// Select the correct optionId from the agent-provided options.
	optionID := t.handler.SelectOptionID(requestID, approved)
	result := map[string]any{
		"outcome": map[string]any{
			"outcome":  "selected",
			"optionId": optionID,
		},
	}

	return t.tracker.Writer.WriteResponse(rpcID, result, nil)
}

// CancelSession sends a session/cancel notification.
func (t *ACPTransport) CancelSession(sessionID string) error {
	params := map[string]any{
		"sessionId": sessionID,
	}
	return t.tracker.Writer.WriteNotification("session/cancel", params)
}

// SendControlRequest issues a `session/control_request` JSON-RPC and waits
// for the agent's response. This is an AgentsMesh extension to the standard
// ACP protocol — agents that don't implement it (codex, gemini, opencode)
// return method_not_found, which surfaces here as an error so callers can
// degrade gracefully. The mock agent (e2e-mock-agent) and any future agent
// that opts into control-plane round-trips should accept this method.
//
// Capability check up-front: if the agent didn't advertise
// agentsmeshExtensions.controlRequest in its initialize response, we
// short-circuit to ErrControlNotSupported instead of waiting on the
// 10-second JSON-RPC timeout. This keeps the Selector responsive on agents
// that simply don't implement the extension.
//
// Payload schema: { sessionId, subtype, params? }. The subtype is the
// runner-side action name (e.g. "set_permission_mode", "set_model"),
// kept stable so agents can dispatch by literal match.
func (t *ACPTransport) SendControlRequest(sessionID string, subtype string, payload map[string]any) (map[string]any, error) {
	if !t.supportsControlRequest {
		return nil, ErrControlNotSupported
	}
	params := map[string]any{
		"sessionId": sessionID,
		"subtype":   subtype,
	}
	if len(payload) > 0 {
		params["params"] = payload
	}
	pr, err := t.tracker.SendRequest("session/control_request", params)
	if err != nil {
		return nil, fmt.Errorf("write control_request: %w", err)
	}
	resp, err := t.tracker.WaitResponse(pr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("wait control_request response: %w", err)
	}
	if resp.Error != nil {
		// method_not_found is the canonical "agent does not implement this"
		// signal. ErrControlNotSupported lets ACPClient.SetPermissionMode
		// distinguish "agent rejected" from "agent crashed/timed out".
		if resp.Error.Code == ErrCodeMethodNotFound {
			return nil, ErrControlNotSupported
		}
		return nil, fmt.Errorf("control_request %s: code=%d msg=%s",
			subtype, resp.Error.Code, resp.Error.Message)
	}
	if len(resp.Result) == 0 {
		return map[string]any{}, nil
	}
	var result map[string]any
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse control_request result: %w", err)
	}
	return result, nil
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
		// Standard ACP: agent sends session/request_permission as a request.
		if msg.Method == "session/request_permission" {
			id, _ := msg.GetID()
			t.handler.HandlePermissionRequest(id, msg.Params)
		} else {
			t.tracker.RejectRequest(msg)
		}
	}
}
