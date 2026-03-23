package claude

import (
	"encoding/json"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func (t *Transport) handleUser(msg *message) {
	var user userMessage
	if err := json.Unmarshal(msg.Message, &user); err != nil {
		t.logger.Warn("failed to parse user message", "error", err)
		return
	}

	var blocks []struct {
		Type      string          `json:"type"`
		ToolUseID string          `json:"tool_use_id"`
		Content   json.RawMessage `json:"content"`
		IsError   bool            `json:"is_error"`
	}
	if err := json.Unmarshal(user.Content, &blocks); err != nil {
		return // plain text user content
	}

	t.sessionMu.RLock()
	sid := t.sessionID
	t.sessionMu.RUnlock()

	for _, block := range blocks {
		if block.Type == "tool_result" && t.callbacks.OnToolCallResult != nil {
			text := ExtractToolResultText(block.Content)
			errMsg := ""
			if block.IsError {
				errMsg = text
			}
			t.callbacks.OnToolCallResult(sid, acp.ToolCallResult{
				ToolCallID:   block.ToolUseID,
				Success:      !block.IsError,
				ResultText:   text,
				ErrorMessage: errMsg,
			})
		}
	}
}

// ExtractToolResultText extracts text from a tool_result content field,
// which can be either a JSON string or an array of content blocks.
func ExtractToolResultText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return string(raw)
}
