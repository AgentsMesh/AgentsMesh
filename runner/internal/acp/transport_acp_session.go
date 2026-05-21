package acp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

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
