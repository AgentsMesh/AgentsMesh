package claude

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func (t *transport) handleStreamEvent(msg *message) {
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
	default:
		t.logger.Debug("unhandled stream_event type", "type", evt.Type)
	}
}

func (t *transport) handleContentBlockStart(index int, raw json.RawMessage) {
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

func (t *transport) handleContentBlockDelta(index int, raw json.RawMessage) {
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
		t.hasStreamedText = true
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

func (t *transport) handleContentBlockStop(index int) {
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
