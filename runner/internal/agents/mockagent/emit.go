package mockagent

import (
	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// emit_* helpers compose the ACP session/update payloads consumed by the
// runner's handler and dispatched to the web via relay. Each helper is
// intentionally a thin one-liner over acp.Writer so scenarios stay readable.
//
// All notifications carry sessionId=mockSessionID so consumers can correlate
// against the session/new response; the runner-side router does not enforce
// this but Claude-style transports do.

func emitContentChunk(w *acp.Writer, text, role string) error {
	updateType := "agent_message_chunk"
	if role == "user" {
		updateType = "user_message_chunk"
	}
	return w.WriteNotification("session/update", map[string]any{
		"sessionId": mockSessionID,
		"update": map[string]any{
			"sessionUpdate": updateType,
			"content":       map[string]any{"type": "text", "text": text},
		},
	})
}

func emitThinkingChunk(w *acp.Writer, text string) error {
	return w.WriteNotification("session/update", map[string]any{
		"sessionId": mockSessionID,
		"update": map[string]any{
			"sessionUpdate": "agent_thought_chunk",
			"content":       map[string]any{"type": "text", "text": text},
		},
	})
}

func emitToolCall(w *acp.Writer, id, title, status string) error {
	return w.WriteNotification("session/update", map[string]any{
		"sessionId": mockSessionID,
		"update": map[string]any{
			"sessionUpdate": "tool_call",
			"toolCallId":    id,
			"title":         title,
			"status":        status,
		},
	})
}

func emitToolCallUpdate(w *acp.Writer, id, title, status, resultText, errorMessage string) error {
	update := map[string]any{
		"sessionUpdate": "tool_call_update",
		"toolCallId":    id,
		"title":         title,
		"status":        status,
	}
	if resultText != "" {
		update["resultText"] = resultText
	}
	if errorMessage != "" {
		update["errorMessage"] = errorMessage
	}
	return w.WriteNotification("session/update", map[string]any{
		"sessionId": mockSessionID,
		"update":    update,
	})
}

// emitPermissionRequest sends an out-of-band JSON-RPC *request* (not a
// notification) per the ACP spec: the runner expects a control flow where
// permission decisions echo back through session/request_permission.
//
// Because the binary doesn't have a request tracker for inbound responses
// (the runner is the consumer, not us), we use WriteRequestWithID with a
// fixed id so a test scenario can predict & respond to it. The returned
// id is the same one passed in (helper convenience).
func emitPermissionRequest(w *acp.Writer, requestID int64, toolCallID, toolTitle string) (int64, error) {
	err := w.WriteRequestWithID(requestID, "session/request_permission", map[string]any{
		"sessionId": mockSessionID,
		"toolCall": map[string]any{
			"toolCallId": toolCallID,
			"title":      toolTitle,
		},
		"options": []map[string]string{
			{"optionId": "allow_once", "name": "Allow", "kind": "allow_once"},
			{"optionId": "reject_once", "name": "Deny", "kind": "reject_once"},
		},
	})
	return requestID, err
}
