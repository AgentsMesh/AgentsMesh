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

type transport struct {
	callbacks acp.EventCallbacks
	logger    *slog.Logger

	stdin   io.Writer
	stdinMu sync.Mutex

	scanner *bufio.Scanner

	sessionID string
	sessionMu sync.RWMutex

	initCh chan struct{}

	toolCalls   map[int]*toolCallState
	toolCallsMu sync.Mutex

	hasStreamedText bool

	pendingInputs   map[string]json.RawMessage
	pendingInputsMu sync.Mutex

	outgoing *controlRequestTracker
}

func newTransport(callbacks acp.EventCallbacks, logger *slog.Logger) *transport {
	return &transport{
		callbacks:     callbacks,
		logger:        logger,
		toolCalls:     make(map[int]*toolCallState),
		pendingInputs: make(map[string]json.RawMessage),
		initCh:        make(chan struct{}),
		outgoing:      newControlRequestTracker(),
	}
}

func (t *transport) Initialize(_ context.Context, stdin io.Writer, stdout io.Reader, _ io.Reader) error {
	t.stdin = stdin
	t.scanner = bufio.NewScanner(stdout)
	t.scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	return nil
}

func (t *transport) Handshake(ctx context.Context) (string, error) {
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
		return "", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (t *transport) NewSession(_ string, _ map[string]any) (string, error) {
	t.sessionMu.RLock()
	defer t.sessionMu.RUnlock()
	return t.sessionID, nil
}

func (t *transport) SendPrompt(sessionID, prompt string) error {
	msg := userInput{
		Type: "user", SessionID: sessionID,
		Message: userInputMsg{Role: "user", Content: prompt},
	}
	if err := t.writeStdin(msg); err != nil {
		return fmt.Errorf("write user input: %w", err)
	}
	return nil
}

func (t *transport) RespondToPermission(requestID string, approved bool, updatedInput map[string]any) error {
	respData := controlResponseData{}
	if approved {
		respData.Behavior = "allow"
		if updatedInput != nil {
			if perms, ok := updatedInput["updatedPermissions"]; ok {
				b, _ := json.Marshal(perms)
				respData.UpdatedPermissions = json.RawMessage(b)
				cleaned := make(map[string]any, len(updatedInput)-1)
				for k, v := range updatedInput {
					if k != "updatedPermissions" {
						cleaned[k] = v
					}
				}
				updatedInput = cleaned
			}
		}
		if len(updatedInput) > 0 {
			b, _ := json.Marshal(updatedInput)
			respData.UpdatedInput = json.RawMessage(b)
		} else {
			t.pendingInputsMu.Lock()
			if input, ok := t.pendingInputs[requestID]; ok {
				respData.UpdatedInput = input
			}
			t.pendingInputsMu.Unlock()
		}
	} else {
		respData.Behavior = "deny"
		respData.Message = "Denied by user"
	}
	t.pendingInputsMu.Lock()
	delete(t.pendingInputs, requestID)
	t.pendingInputsMu.Unlock()

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

func (t *transport) CancelSession(_ string) error { return nil }

func (t *transport) SendControlRequest(_ string, subtype string, payload map[string]any) (map[string]any, error) {
	t.logger.Info("Sending control_request", "subtype", subtype)
	return t.outgoing.sendAndWait(subtype, payload, t.writeStdin)
}

func (t *transport) Close() {}
