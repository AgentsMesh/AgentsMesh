package codex

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func (t *Transport) getSessionID() string {
	t.sessionMu.RLock()
	defer t.sessionMu.RUnlock()
	return t.sessionID
}

func (t *Transport) handleNotification(method string, params json.RawMessage) {
	sid := t.getSessionID()

	switch method {
	case "turn/completed":
		t.handleTurnCompleted(params)

	case "item/agentMessage/delta":
		var d agentMessageDelta
		if err := json.Unmarshal(params, &d); err != nil {
			t.logger.Warn("failed to parse agentMessage/delta", "error", err)
			return
		}
		if t.callbacks.OnContentChunk != nil {
			t.callbacks.OnContentChunk(sid, acp.ContentChunk{Text: d.Delta, Role: "assistant"})
		}

	case "item/reasoning/summaryTextDelta", "item/reasoning/textDelta":
		var d reasoningDelta
		if err := json.Unmarshal(params, &d); err != nil {
			t.logger.Warn("failed to parse reasoning delta", "error", err)
			return
		}
		if t.callbacks.OnThinkingUpdate != nil {
			t.callbacks.OnThinkingUpdate(sid, acp.ThinkingUpdate{Text: d.Delta})
		}

	case "item/plan/delta":
		var d planDelta
		if err := json.Unmarshal(params, &d); err != nil {
			t.logger.Warn("failed to parse plan/delta", "error", err)
			return
		}
		if t.callbacks.OnContentChunk != nil {
			t.callbacks.OnContentChunk(sid, acp.ContentChunk{Text: d.Delta, Role: "plan"})
		}

	case "item/started":
		t.handleItemStarted(sid, params)

	case "item/completed":
		t.handleItemCompleted(sid, params)

	default:
		t.logger.Debug("unhandled codex notification", "method", method)
	}
}

// handleItemStarted dispatches item/started by item.type.
func (t *Transport) handleItemStarted(sid string, params json.RawMessage) {
	var is itemStartedParams
	if err := json.Unmarshal(params, &is); err != nil {
		t.logger.Warn("failed to parse item/started", "error", err)
		return
	}
	switch is.Item.Type {
	case "toolCall":
		if t.callbacks.OnToolCallUpdate != nil {
			t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
				ToolCallID: is.Item.ID, ToolName: is.Item.ToolName, Status: "running",
			})
		}
	case "commandExecution":
		if t.callbacks.OnToolCallUpdate != nil {
			t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
				ToolCallID: is.Item.ID, ToolName: "shell", Status: "running",
			})
		}
	case "fileChange":
		if t.callbacks.OnToolCallUpdate != nil {
			t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
				ToolCallID: is.Item.ID, ToolName: "fileChange", Status: "running",
			})
		}
	}
}

// handleItemCompleted dispatches item/completed by item.type.
func (t *Transport) handleItemCompleted(sid string, params json.RawMessage) {
	var ic itemCompletedParams
	if err := json.Unmarshal(params, &ic); err != nil {
		t.logger.Warn("failed to parse item/completed", "error", err)
		return
	}
	switch ic.Item.Type {
	case "toolCall":
		if t.callbacks.OnToolCallUpdate != nil {
			t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
				ToolCallID: ic.Item.ID, ToolName: ic.Item.ToolName, Status: "completed",
			})
		}
	case "commandExecution":
		exitCode := 0
		if ic.Item.ExitCode != nil {
			exitCode = *ic.Item.ExitCode
		}
		if t.callbacks.OnToolCallResult != nil {
			t.callbacks.OnToolCallResult(sid, acp.ToolCallResult{
				ToolCallID: ic.Item.ID, ToolName: "shell",
				Success: exitCode == 0, ResultText: ic.Item.AggregatedOutput,
			})
		}
	case "fileChange":
		success := ic.Item.Status == "" || ic.Item.Status == "completed"
		if t.callbacks.OnToolCallResult != nil {
			t.callbacks.OnToolCallResult(sid, acp.ToolCallResult{
				ToolCallID: ic.Item.ID, ToolName: "fileChange",
				Success: success, ResultText: ic.Item.FilePath,
			})
		}
		// agentMessage: content already delivered via item/agentMessage/delta — no action needed.
	}
}

func (t *Transport) handleTurnCompleted(params json.RawMessage) {
	var tc turnCompletedParams
	if err := json.Unmarshal(params, &tc); err != nil {
		if t.callbacks.OnStateChange != nil {
			t.callbacks.OnStateChange(acp.StateIdle)
		}
		return
	}
	if tc.Turn.Status == "failed" && t.callbacks.OnLog != nil {
		msg := "turn failed"
		if tc.Turn.Error != nil {
			msg = "turn failed: " + tc.Turn.Error.Message
		}
		t.callbacks.OnLog("error", msg)
	}
	if t.callbacks.OnStateChange != nil {
		t.callbacks.OnStateChange(acp.StateIdle)
	}
}

func (t *Transport) handleApprovalRequest(rpcID int64, params json.RawMessage) {
	var req approvalRequestParams
	if err := json.Unmarshal(params, &req); err != nil {
		t.logger.Warn("failed to parse approval request", "error", err)
		return
	}

	if t.callbacks.OnStateChange != nil {
		t.callbacks.OnStateChange(acp.StateWaitingPermission)
	}

	description := req.Description
	toolName := "command"
	if req.Path != "" {
		toolName = "fileChange"
		if description == "" {
			description = req.Path
		}
	}
	if req.Command != "" && description == "" {
		description = req.Command
	}

	if t.callbacks.OnPermissionRequest != nil {
		t.callbacks.OnPermissionRequest(acp.PermissionRequest{
			SessionID:   t.getSessionID(),
			RequestID:   fmt.Sprintf("%d", rpcID),
			ToolName:    toolName,
			Description: description,
		})
	}
}
