package codex

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestHandler_AgentMessageDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/agentMessage/delta", agentMessageDelta{Delta: "Hello "})
	writeNotification(f.PW, "item/agentMessage/delta", agentMessageDelta{Delta: "world!"})
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

func TestHandler_ReasoningDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/reasoning/summaryTextDelta", reasoningDelta{Delta: "hmm"})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ThinkingTexts) != 1 || f.ThinkingTexts[0] != "hmm" {
		t.Errorf("thinking = %v", f.ThinkingTexts)
	}
}

func TestHandler_ReasoningTextDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/reasoning/textDelta", reasoningDelta{Delta: "deep thought"})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ThinkingTexts) != 1 || f.ThinkingTexts[0] != "deep thought" {
		t.Errorf("thinking = %v", f.ThinkingTexts)
	}
}

func TestHandler_PlanDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/plan/delta", planDelta{Delta: "Step 1: Read files"})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.Chunks) != 1 {
		t.Fatalf("expected 1 plan chunk, got %d", len(f.Chunks))
	}
	if f.Chunks[0].Text != "Step 1: Read files" || f.Chunks[0].Role != "plan" {
		t.Errorf("chunk = %+v", f.Chunks[0])
	}
}

func TestHandler_ItemStarted_ToolCall(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/started", map[string]any{
		"item": map[string]any{"id": "tc1", "type": "toolCall", "toolName": "Read"},
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

func TestHandler_ItemStarted_CommandExecution(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/started", map[string]any{
		"item": map[string]any{"id": "ce1", "type": "commandExecution"},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ToolUpdates) != 1 || f.ToolUpdates[0].ToolName != "shell" {
		t.Errorf("updates = %+v", f.ToolUpdates)
	}
}

func TestHandler_ItemCompleted_CommandExecution(t *testing.T) {
	f := newFixture()
	defer f.Close()
	exitCode := 0
	writeNotification(f.PW, "item/completed", map[string]any{
		"item": map[string]any{
			"id": "ce1", "type": "commandExecution",
			"exitCode": exitCode, "aggregatedOutput": "file list",
		},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ToolResults) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(f.ToolResults))
	}
	if !f.ToolResults[0].Success || f.ToolResults[0].ResultText != "file list" {
		t.Errorf("result = %+v", f.ToolResults[0])
	}
}

func TestHandler_ItemCompleted_CommandExecution_Failure(t *testing.T) {
	f := newFixture()
	defer f.Close()
	exitCode := 1
	writeNotification(f.PW, "item/completed", map[string]any{
		"item": map[string]any{
			"id": "ce2", "type": "commandExecution",
			"exitCode": exitCode, "aggregatedOutput": "error",
		},
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
	writeNotification(f.PW, "item/completed", map[string]any{
		"item": map[string]any{"id": "tc1", "type": "toolCall", "toolName": "Write"},
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
	writeNotification(f.PW, "item/completed", map[string]any{
		"item": map[string]any{"id": "x", "type": "agentMessage"},
	})
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

func TestHandler_TurnCompleted_Failed(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "turn/completed", map[string]any{
		"turn": map[string]any{
			"status": "failed",
			"error":  map[string]any{"message": "API error"},
		},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.StateChanges) != 1 || f.StateChanges[0] != acp.StateIdle {
		t.Errorf("states = %v", f.StateChanges)
	}
}

func TestHandler_UnknownNotification(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "some/future/method", map[string]any{})
	f.Drain()
}

func TestHandler_InvalidParams(t *testing.T) {
	f := newFixture()
	defer f.Close()
	for _, method := range []string{
		"item/agentMessage/delta",
		"item/reasoning/summaryTextDelta",
		"item/plan/delta",
		"item/started",
		"item/completed",
	} {
		writeNotification(f.PW, method, "invalid")
	}
	f.Drain()
}
