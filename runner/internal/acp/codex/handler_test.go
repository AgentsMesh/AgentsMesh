package codex

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestHandler_AgentMessageDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/agentMessage/delta", agentMessageDelta{Text: "Hello "})
	writeNotification(f.PW, "item/agentMessage/delta", agentMessageDelta{Text: "world!"})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(f.Chunks))
	}
	if f.Chunks[0].Text != "Hello " || f.Chunks[0].Role != "assistant" {
		t.Errorf("chunk[0] = %+v", f.Chunks[0])
	}
}

func TestHandler_ThinkingDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/thinking/delta", thinkingDelta{Text: "hmm"})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ThinkingTexts) != 1 || f.ThinkingTexts[0] != "hmm" {
		t.Errorf("thinking = %v", f.ThinkingTexts)
	}
}

func TestHandler_PlanDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/plan/delta", planDelta{
		Step: struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		}{Title: "Read files", Status: "in_progress"},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.PlanSteps) != 1 {
		t.Fatalf("expected 1 plan step, got %d", len(f.PlanSteps))
	}
	if f.PlanSteps[0].Title != "Read files" || f.PlanSteps[0].Status != "in_progress" {
		t.Errorf("step = %+v", f.PlanSteps[0])
	}
}

func TestHandler_ToolCallStarted(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/toolCall/started", toolCallStarted{
		ToolCallID: "tc1", ToolName: "Read", ArgumentsJSON: `{"file":"main.go"}`,
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ToolUpdates) != 1 {
		t.Fatalf("expected 1 tool update, got %d", len(f.ToolUpdates))
	}
	if f.ToolUpdates[0].Status != "running" || f.ToolUpdates[0].ToolName != "Read" {
		t.Errorf("update = %+v", f.ToolUpdates[0])
	}
}

func TestHandler_CommandExecution(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/commandExecution/started", commandExecutionStarted{
		ToolCallID: "ce1", Command: "ls -la",
	})
	writeNotification(f.PW, "item/commandExecution/completed", commandExecutionCompleted{
		ToolCallID: "ce1", ExitCode: 0, Output: "file list",
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ToolUpdates) != 1 {
		t.Fatalf("expected 1 tool update, got %d", len(f.ToolUpdates))
	}
	if f.ToolUpdates[0].ToolName != "shell" || f.ToolUpdates[0].Status != "running" {
		t.Errorf("update = %+v", f.ToolUpdates[0])
	}
	if len(f.ToolResults) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(f.ToolResults))
	}
	if !f.ToolResults[0].Success || f.ToolResults[0].ResultText != "file list" {
		t.Errorf("result = %+v", f.ToolResults[0])
	}
}

func TestHandler_CommandExecution_Failure(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/commandExecution/completed", commandExecutionCompleted{
		ToolCallID: "ce2", ExitCode: 1, Output: "error",
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ToolResults) != 1 || f.ToolResults[0].Success {
		t.Errorf("expected failure, got %+v", f.ToolResults)
	}
}

func TestHandler_ItemCompleted_ToolCall(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/completed", itemCompleted{
		Type: "tool_call", ToolCallID: "tc1", ToolName: "Write",
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ToolUpdates) != 1 || f.ToolUpdates[0].Status != "completed" {
		t.Errorf("updates = %+v", f.ToolUpdates)
	}
}

func TestHandler_ItemCompleted_NonToolCall(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/completed", itemCompleted{Type: "agent_message"})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ToolUpdates) != 0 {
		t.Errorf("expected 0 tool updates, got %d", len(f.ToolUpdates))
	}
}

func TestHandler_TurnCompleted(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "turn/completed", map[string]any{})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.StateChanges) != 1 || f.StateChanges[0] != acp.StateIdle {
		t.Errorf("states = %v", f.StateChanges)
	}
}

func TestHandler_ApprovalRequired(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "serverRequest/approvalRequired", approvalRequest{
		RequestID: "r1", Type: "command_execution", Description: "run rm -rf",
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.StateChanges) != 1 || f.StateChanges[0] != acp.StateWaitingPermission {
		t.Errorf("states = %v", f.StateChanges)
	}
	if len(f.PermissionReqs) != 1 || f.PermissionReqs[0].RequestID != "r1" {
		t.Errorf("perms = %+v", f.PermissionReqs)
	}
}

func TestHandler_UnknownNotification(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "some/future/method", map[string]any{})
	f.Drain()
	// Should not panic or produce callbacks
}

func TestHandler_InvalidParams(t *testing.T) {
	f := newFixture()
	defer f.Close()
	// Send notifications with invalid params — should log warn but not panic
	for _, method := range []string{
		"item/agentMessage/delta",
		"item/thinking/delta",
		"item/plan/delta",
		"item/toolCall/started",
		"item/commandExecution/started",
		"item/commandExecution/completed",
		"item/completed",
		"serverRequest/approvalRequired",
	} {
		writeNotification(f.PW, method, "invalid")
	}
	f.Drain()
}
