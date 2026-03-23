package claude

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func (t *Transport) handleSystem(msg *message) {
	if msg.Subtype == "init" {
		t.sessionMu.Lock()
		t.sessionID = msg.SessionID
		t.sessionMu.Unlock()

		t.logger.Info("Claude session initialized", "session_id", msg.SessionID)
		// session_id is set here; initCh is closed by handleControlResponse.
	}
}

// handleControlResponse handles the control_response message from Claude.
// The initialize response closes initCh to unblock Handshake().
func (t *Transport) handleControlResponse(msg *message) {
	select {
	case <-t.initCh:
		// already closed (duplicate response — defensive)
	default:
		close(t.initCh)
	}
}

func (t *Transport) handleStreamEvent(msg *message) {
	var evt streamEvent
	if err := json.Unmarshal(msg.Event, &evt); err != nil {
		t.logger.Warn("failed to parse stream_event", "error", err)
		return
	}

	switch evt.Type {
	case "content_block_start":
		t.handleContentBlockStart(evt.Index, evt.ContentBlock)
	case "content_block_delta":
		t.handleContentBlockDelta(evt.Index, evt.Delta)
	case "content_block_stop":
		t.handleContentBlockStop(evt.Index)
	case "message_start", "message_delta", "message_stop":
		// Handled at the assistant/result message level
	default:
		t.logger.Debug("unhandled stream_event type", "type", evt.Type)
	}
}

func (t *Transport) handleContentBlockStart(index int, raw json.RawMessage) {
	var block contentBlock
	if err := json.Unmarshal(raw, &block); err != nil {
		t.logger.Warn("failed to parse content_block_start", "error", err)
		return
	}

	if block.Type == "tool_use" {
		t.toolCallsMu.Lock()
		t.toolCalls[index] = &toolCallState{
			ID:   block.ID,
			Name: block.Name,
		}
		t.toolCallsMu.Unlock()

		t.sessionMu.RLock()
		sid := t.sessionID
		t.sessionMu.RUnlock()

		if t.callbacks.OnToolCallUpdate != nil {
			t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
				ToolCallID: block.ID,
				ToolName:   block.Name,
				Status:     "running",
			})
		}
	}
}

func (t *Transport) handleContentBlockDelta(index int, raw json.RawMessage) {
	var d delta
	if err := json.Unmarshal(raw, &d); err != nil {
		t.logger.Warn("failed to parse content_block_delta", "error", err)
		return
	}

	t.sessionMu.RLock()
	sid := t.sessionID
	t.sessionMu.RUnlock()

	switch d.Type {
	case "text_delta":
		if t.callbacks.OnContentChunk != nil {
			t.callbacks.OnContentChunk(sid, acp.ContentChunk{
				Text: d.Text,
				Role: "assistant",
			})
		}
	case "thinking_delta":
		if t.callbacks.OnThinkingUpdate != nil {
			t.callbacks.OnThinkingUpdate(sid, acp.ThinkingUpdate{Text: d.Text})
		}
	case "input_json_delta":
		t.toolCallsMu.Lock()
		if tc, ok := t.toolCalls[index]; ok {
			tc.InputJSON += d.PartialJSON
		}
		t.toolCallsMu.Unlock()
	}
}

func (t *Transport) handleContentBlockStop(index int) {
	t.toolCallsMu.Lock()
	tc, ok := t.toolCalls[index]
	if ok {
		delete(t.toolCalls, index)
	}
	t.toolCallsMu.Unlock()

	if ok && t.callbacks.OnToolCallUpdate != nil {
		t.sessionMu.RLock()
		sid := t.sessionID
		t.sessionMu.RUnlock()

		t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
			ToolCallID:    tc.ID,
			ToolName:      tc.Name,
			Status:        "completed",
			ArgumentsJSON: tc.InputJSON,
		})
	}
}

func (t *Transport) handleControlRequest(msg *message) {
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

func (t *Transport) handleCanUseTool(requestID string, req *controlRequestPayload) {
	t.sessionMu.RLock()
	sid := t.sessionID
	t.sessionMu.RUnlock()

	argsJSON := string(req.Input)

	if t.callbacks.OnStateChange != nil {
		t.callbacks.OnStateChange(acp.StateWaitingPermission)
	}
	if t.callbacks.OnPermissionRequest != nil {
		t.callbacks.OnPermissionRequest(acp.PermissionRequest{
			SessionID:     sid,
			RequestID:     requestID,
			ToolName:      req.ToolName,
			ArgumentsJSON: argsJSON,
			Description:   fmt.Sprintf("Tool: %s", req.ToolName),
		})
	}
}

func (t *Transport) handleResult(msg *message) {
	if msg.Subtype == "success" || msg.Subtype == "" {
		if t.callbacks.OnStateChange != nil {
			t.callbacks.OnStateChange(acp.StateIdle)
		}
	} else {
		// error_* subtypes
		if t.callbacks.OnLog != nil {
			t.callbacks.OnLog("error", fmt.Sprintf("Claude result error: subtype=%s result=%s", msg.Subtype, msg.Result))
		}
		if t.callbacks.OnStateChange != nil {
			t.callbacks.OnStateChange(acp.StateIdle)
		}
	}
}
