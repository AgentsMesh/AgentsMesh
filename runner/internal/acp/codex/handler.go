package codex

import (
	"encoding/json"

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
		if t.callbacks.OnStateChange != nil {
			t.callbacks.OnStateChange(acp.StateIdle)
		}

	case "item/agentMessage/delta":
		var d agentMessageDelta
		if err := json.Unmarshal(params, &d); err != nil {
			t.logger.Warn("failed to parse agentMessage/delta", "error", err)
			return
		}
		if t.callbacks.OnContentChunk != nil {
			t.callbacks.OnContentChunk(sid, acp.ContentChunk{Text: d.Text, Role: "assistant"})
		}

	case "item/thinking/delta":
		var d thinkingDelta
		if err := json.Unmarshal(params, &d); err != nil {
			t.logger.Warn("failed to parse thinking/delta", "error", err)
			return
		}
		if t.callbacks.OnThinkingUpdate != nil {
			t.callbacks.OnThinkingUpdate(sid, acp.ThinkingUpdate{Text: d.Text})
		}

	case "item/plan/delta":
		var d planDelta
		if err := json.Unmarshal(params, &d); err != nil {
			t.logger.Warn("failed to parse plan/delta", "error", err)
			return
		}
		if t.callbacks.OnPlanUpdate != nil {
			t.callbacks.OnPlanUpdate(sid, acp.PlanUpdate{
				Steps: []acp.PlanStep{{Title: d.Step.Title, Status: d.Step.Status}},
			})
		}

	case "item/toolCall/started":
		var tc toolCallStarted
		if err := json.Unmarshal(params, &tc); err != nil {
			t.logger.Warn("failed to parse toolCall/started", "error", err)
			return
		}
		if t.callbacks.OnToolCallUpdate != nil {
			t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
				ToolCallID: tc.ToolCallID, ToolName: tc.ToolName,
				Status: "running", ArgumentsJSON: tc.ArgumentsJSON,
			})
		}

	case "item/commandExecution/started":
		var ce commandExecutionStarted
		if err := json.Unmarshal(params, &ce); err != nil {
			t.logger.Warn("failed to parse commandExecution/started", "error", err)
			return
		}
		if t.callbacks.OnToolCallUpdate != nil {
			t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
				ToolCallID: ce.ToolCallID, ToolName: "shell", Status: "running",
			})
		}

	case "item/commandExecution/completed":
		var ce commandExecutionCompleted
		if err := json.Unmarshal(params, &ce); err != nil {
			t.logger.Warn("failed to parse commandExecution/completed", "error", err)
			return
		}
		if t.callbacks.OnToolCallResult != nil {
			t.callbacks.OnToolCallResult(sid, acp.ToolCallResult{
				ToolCallID: ce.ToolCallID, ToolName: "shell",
				Success: ce.ExitCode == 0, ResultText: ce.Output,
			})
		}

	case "item/completed":
		var ic itemCompleted
		if err := json.Unmarshal(params, &ic); err != nil {
			t.logger.Warn("failed to parse item/completed", "error", err)
			return
		}
		if ic.Type == "tool_call" && t.callbacks.OnToolCallUpdate != nil {
			t.callbacks.OnToolCallUpdate(sid, acp.ToolCallUpdate{
				ToolCallID: ic.ToolCallID, ToolName: ic.ToolName, Status: "completed",
			})
		}

	case "serverRequest/approvalRequired":
		var req approvalRequest
		if err := json.Unmarshal(params, &req); err != nil {
			t.logger.Warn("failed to parse approvalRequired", "error", err)
			return
		}
		if t.callbacks.OnStateChange != nil {
			t.callbacks.OnStateChange(acp.StateWaitingPermission)
		}
		if t.callbacks.OnPermissionRequest != nil {
			t.callbacks.OnPermissionRequest(acp.PermissionRequest{
				SessionID:   sid,
				RequestID:   req.RequestID,
				ToolName:    req.Type,
				Description: req.Description,
			})
		}

	default:
		t.logger.Debug("unhandled codex notification", "method", method)
	}
}
