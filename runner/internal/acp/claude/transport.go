package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func init() {
	acp.RegisterTransport(acp.TransportTypeClaudeStream, func(cb acp.EventCallbacks, l *slog.Logger) acp.Transport {
		return NewTransport(cb, l)
	})
}

// Transport implements the acp.Transport interface for
// Claude Code's stream-json NDJSON protocol.
type Transport struct {
	callbacks acp.EventCallbacks
	logger    *slog.Logger

	stdin   io.Writer
	stdinMu sync.Mutex // protects stdin writes
	scanner *bufio.Scanner

	// Session ID discovered asynchronously (system subtype=init message).
	sessionID string
	sessionMu sync.RWMutex

	// initCh is closed when the control_response (initialize) arrives.
	// Handshake() blocks on this channel.
	initCh chan struct{}

	// toolCalls tracks in-progress streaming tool_use blocks by index.
	toolCalls   map[int]*toolCallState
	toolCallsMu sync.Mutex

	// hasStreamedText is set when text_delta events are received in the current turn.
	// When true, the final assistant message's text blocks are skipped (already delivered).
	hasStreamedText bool

	// pendingInputs stores original tool input by requestID for permission responses.
	pendingInputs   map[string]json.RawMessage
	pendingInputsMu sync.Mutex
}

// NewTransport creates a new Claude stream-json transport.
func NewTransport(callbacks acp.EventCallbacks, logger *slog.Logger) *Transport {
	return &Transport{
		callbacks:     callbacks,
		logger:        logger,
		toolCalls:     make(map[int]*toolCallState),
		pendingInputs: make(map[string]json.RawMessage),
		initCh:        make(chan struct{}),
	}
}

// Initialize sets up I/O. Claude doesn't have a synchronous handshake.
func (t *Transport) Initialize(_ context.Context, stdin io.Writer, stdout io.Reader, _ io.Reader) error {
	t.stdin = stdin
	t.scanner = bufio.NewScanner(stdout)
	t.scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	return nil
}

// writeStdin marshals v as NDJSON and writes it to stdin.
func (t *Transport) writeStdin(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	t.stdinMu.Lock()
	defer t.stdinMu.Unlock()
	_, err = t.stdin.Write(data)
	return err
}

// Handshake sends a control_request initialize and blocks until the
// control_response arrives. The session_id is discovered later when the
// first user message triggers a system/init — Handshake returns "" here.
func (t *Transport) Handshake(ctx context.Context) (string, error) {
	if t.stdin != nil {
		msg := controlInitMessage{
			Type: "control_request", RequestID: "init_1",
			Request: controlInitPayload{Subtype: "initialize"},
		}
		if err := t.writeStdin(msg); err != nil {
			return "", fmt.Errorf("write control_request initialize: %w", err)
		}
	}

	select {
	case <-t.initCh:
		return "", nil // session_id set asynchronously by system/init
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// NewSession is a no-op for Claude: the session is auto-created on startup.
func (t *Transport) NewSession(_ string, _ map[string]any) (string, error) {
	t.sessionMu.RLock()
	defer t.sessionMu.RUnlock()
	return t.sessionID, nil
}

// SendPrompt writes a user message to Claude's stdin as NDJSON.
func (t *Transport) SendPrompt(sessionID, prompt string) error {
	msg := userInput{
		Type: "user", SessionID: sessionID,
		Message: userInputMsg{Role: "user", Content: prompt},
	}
	if err := t.writeStdin(msg); err != nil {
		return fmt.Errorf("write user input: %w", err)
	}
	return nil
}

// RespondToPermission sends a control_response to approve or deny a tool request.
func (t *Transport) RespondToPermission(requestID string, approved bool) error {
	respData := controlResponseData{}
	if approved {
		respData.Behavior = "allow"
		t.pendingInputsMu.Lock()
		if input, ok := t.pendingInputs[requestID]; ok {
			respData.UpdatedInput = input
			delete(t.pendingInputs, requestID)
		}
		t.pendingInputsMu.Unlock()
	} else {
		respData.Behavior = "deny"
		respData.Message = "Denied by user"
		t.pendingInputsMu.Lock()
		delete(t.pendingInputs, requestID)
		t.pendingInputsMu.Unlock()
	}
	msg := controlResponseMessage{
		Type: "control_response",
		Response: controlResponsePayload{
			Subtype: "success", RequestID: requestID,
			Response: respData,
		},
	}
	if err := t.writeStdin(msg); err != nil {
		return fmt.Errorf("write control response: %w", err)
	}
	return nil
}

// CancelSession is a no-op — the caller should kill the subprocess instead.
func (t *Transport) CancelSession(_ string) error {
	return nil
}

// ReadLoop reads NDJSON lines from Claude's stdout and dispatches events.
func (t *Transport) ReadLoop(ctx context.Context) {
	for t.scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := t.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg message
		if err := json.Unmarshal(line, &msg); err != nil {
			t.logger.Warn("failed to parse claude message", "error", err, "line", string(line))
			continue
		}

		t.handleMessage(&msg)
	}

	if err := t.scanner.Err(); err != nil {
		select {
		case <-ctx.Done():
		default:
			t.logger.Error("claude stdout read error", "error", err)
		}
	}
}

// Close is a no-op (resources owned by ACPClient).
func (t *Transport) Close() {}

// --- message dispatch ---

func (t *Transport) handleMessage(msg *message) {
	switch msg.Type {
	case "system":
		t.handleSystem(msg)
	case "stream_event":
		t.handleStreamEvent(msg)
	case "assistant":
		t.handleAssistant(msg)
	case "user":
		t.handleUser(msg)
	case "result":
		t.handleResult(msg)
	case "control_request":
		t.handleControlRequest(msg)
	case "control_response":
		t.handleControlResponse(msg)
	default:
		t.logger.Debug("unhandled claude message type", "type", msg.Type)
	}
}
