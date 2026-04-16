package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// Transport implements acp.Transport for the Codex CLI app-server
// JSON-RPC 2.0 protocol (launched via `codex app-server`).
type transport struct {
	tracker   *acp.RequestTracker
	reader    *acp.Reader
	callbacks acp.EventCallbacks

	sessionID string
	sessionMu sync.RWMutex

	ctx    context.Context
	logger *slog.Logger
}

func newTransport(callbacks acp.EventCallbacks, logger *slog.Logger) *transport {
	return &transport{
		callbacks: callbacks,
		logger:    logger,
	}
}

func (t *transport) Initialize(ctx context.Context, stdin io.Writer, stdout io.Reader, _ io.Reader) error {
	t.ctx = ctx
	writer := acp.NewWriter(stdin)
	t.reader = acp.NewReader(stdout, t.logger)
	t.tracker = acp.NewRequestTracker(writer, t.logger, func() <-chan struct{} { return ctx.Done() })
	return nil
}

func (t *transport) Handshake(_ context.Context) (string, error) {
	params := map[string]any{
		"clientInfo": map[string]any{
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

	if err := t.tracker.Writer.WriteNotification("initialized", nil); err != nil {
		return "", fmt.Errorf("write initialized: %w", err)
	}

	t.logger.Info("Codex initialize succeeded")
	return "", nil
}

func (t *transport) NewSession(_ string, mcpServers map[string]any) (string, error) {
	params := map[string]any{}
	if mcpServers != nil {
		params["mcpServers"] = mcpServers
	}

	pr, err := t.tracker.SendRequest("thread/start", params)
	if err != nil {
		return "", fmt.Errorf("write thread/start: %w", err)
	}

	resp, err := t.tracker.WaitResponse(pr, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("wait thread/start response: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("thread/start error: code=%d msg=%s",
			resp.Error.Code, resp.Error.Message)
	}

	var result threadStartResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("parse thread/start result: %w", err)
	}

	t.sessionMu.Lock()
	t.sessionID = result.Thread.ID
	t.sessionMu.Unlock()

	return result.Thread.ID, nil
}

func (t *transport) SendPrompt(sessionID, prompt string) error {
	params := turnStartParams{
		ThreadID: sessionID,
		Input: []turnInput{{
			Type: "text",
			Text: prompt,
		}},
	}

	pr, err := t.tracker.SendRequest("turn/start", params)
	if err != nil {
		return fmt.Errorf("write turn/start: %w", err)
	}

	go func() {
		resp, err := t.tracker.WaitResponse(pr, 5*time.Minute)
		if err != nil {
			t.logger.Error("turn/start response error", "error", err)
		} else if resp.Error != nil {
			t.logger.Error("turn/start error",
				"code", resp.Error.Code, "message", resp.Error.Message)
		}
	}()

	return nil
}

func (t *transport) RespondToPermission(requestID string, approved bool, _ map[string]any) error {
	rpcID, err := strconv.ParseInt(requestID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse request ID %q: %w", requestID, err)
	}
	decision := "decline"
	if approved {
		decision = "accept"
	}
	result := map[string]any{"decision": decision}
	return t.tracker.Writer.WriteResponse(rpcID, result, nil)
}

func (t *transport) CancelSession(sessionID string) error {
	params := turnInterruptParams{ThreadID: sessionID}
	pr, err := t.tracker.SendRequest("turn/interrupt", params)
	if err != nil {
		return fmt.Errorf("write turn/interrupt: %w", err)
	}
	go func() {
		t.tracker.WaitResponse(pr, 10*time.Second)
	}()
	return nil
}

func (t *transport) SendControlRequest(_ string, _ string, _ map[string]any) (map[string]any, error) {
	return nil, acp.ErrControlNotSupported
}

func (t *transport) ReadLoop(ctx context.Context) {
	for {
		msg, err := t.reader.ReadMessage()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				t.logger.Error("codex read error", "error", err)
				return
			}
		}
		t.dispatchMessage(msg)
	}
}

func (t *transport) Close() {}
