package claude

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func (t *transport) handleSystem(msg *message) {
	switch msg.Subtype {
	case "init":
		t.sessionMu.Lock()
		t.sessionID = msg.SessionID
		t.sessionMu.Unlock()

		t.logger.Info("Claude session initialized", "session_id", msg.SessionID)
	case "api_retry":
		if t.callbacks.OnLog != nil {
			t.callbacks.OnLog("warn", "Claude API retry in progress")
		}
	}
}

func (t *transport) handleControlResponse(msg *message) {
	if t.resolveOutgoingControlResponse(msg) {
		return
	}
	select {
	case <-t.initCh:
	default:
		close(t.initCh)
	}
}

func (t *transport) handleControlRequest(msg *message) {
	if msg.RequestID == "" {
		t.logger.Warn("control_request missing request_id")
		return
	}

	var req controlRequestPayload
	if err := json.Unmarshal(msg.Request, &req); err != nil {
		t.logger.Warn("failed to parse control_request", "error", err)
		return
	}

	switch req.Subtype {
	case "can_use_tool":
		t.handleCanUseTool(msg.RequestID, &req)
	default:
		t.logger.Debug("unhandled control_request subtype", "subtype", req.Subtype)
	}
}

func (t *transport) handleCanUseTool(requestID string, req *controlRequestPayload) {
	t.sessionMu.RLock()
	sid := t.sessionID
	t.sessionMu.RUnlock()

	t.pendingInputsMu.Lock()
	t.pendingInputs[requestID] = req.Input
	t.pendingInputsMu.Unlock()

	argsJSON := string(req.Input)

	desc := req.Description
	if desc == "" {
		desc = fmt.Sprintf("Tool: %s", req.ToolName)
	}

	if t.callbacks.OnStateChange != nil {
		t.callbacks.OnStateChange(acp.StateWaitingPermission)
	}
	if t.callbacks.OnPermissionRequest != nil {
		t.callbacks.OnPermissionRequest(acp.PermissionRequest{
			SessionID:     sid,
			RequestID:     requestID,
			ToolName:      req.ToolName,
			ArgumentsJSON: argsJSON,
			Description:   desc,
		})
	}
}

func (t *transport) handleAssistant(msg *message) {
	if len(msg.Message) == 0 {
		return
	}

	var assistantMsg struct {
		Content []contentBlock `json:"content"`
	}
	if err := json.Unmarshal(msg.Message, &assistantMsg); err != nil {
		t.logger.Debug("failed to parse assistant message", "error", err)
		return
	}

	t.sessionMu.RLock()
	sid := t.sessionID
	t.sessionMu.RUnlock()

	for _, block := range assistantMsg.Content {
		switch block.Type {
		case "text":
			if t.hasStreamedText {
				continue
			}
			if block.Text != "" && t.callbacks.OnContentChunk != nil {
				t.callbacks.OnContentChunk(sid, acp.ContentChunk{
					Text: block.Text,
					Role: "assistant",
				})
			}
		case "thinking":
			if block.Text != "" && t.callbacks.OnThinkingUpdate != nil {
				t.callbacks.OnThinkingUpdate(sid, acp.ThinkingUpdate{Text: block.Text})
			}
		case "tool_use":
			if t.callbacks.OnToolCallUpdate != nil {
				argsJSON := ""
				if block.Input != nil {
					if b, err := json.Marshal(block.Input); err == nil {
						argsJSON = string(b)
					}
				}
				t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
					ToolCallID:    block.ID,
					ToolName:      block.Name,
					Status:        "completed",
					ArgumentsJSON: argsJSON,
				})
			}
		}
	}
}

func (t *transport) handleResult(msg *message) {
	t.hasStreamedText = false

	if msg.Subtype == "success" || msg.Subtype == "" {
		if t.callbacks.OnStateChange != nil {
			t.callbacks.OnStateChange(acp.StateIdle)
		}
	} else {
		if t.callbacks.OnLog != nil {
			t.callbacks.OnLog("error", fmt.Sprintf("Claude result error: subtype=%s result=%s", msg.Subtype, msg.Result))
		}
		if msg.Result != "" && t.callbacks.OnContentChunk != nil {
			t.sessionMu.RLock()
			sid := t.sessionID
			t.sessionMu.RUnlock()
			t.callbacks.OnContentChunk(sid, acp.ContentChunk{
				Text: msg.Result,
				Role: "assistant",
			})
		}
		if t.callbacks.OnStateChange != nil {
			t.callbacks.OnStateChange(acp.StateIdle)
		}
	}
}
