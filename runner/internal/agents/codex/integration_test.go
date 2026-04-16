package codex

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestCodexTransport_FullPromptFlow(t *testing.T) {
	f := newFixture()
	defer f.Close()

	writeNotification(f.PW, "item/started", map[string]any{
		"item": map[string]any{"id": "ce-1", "type": "commandExecution"},
	})

	writeNotification(f.PW, "item/agentMessage/delta", agentMessageDelta{
		ItemID: "msg-1", Delta: "Here is the output",
	})

	writeNotification(f.PW, "item/completed", map[string]any{
		"item": map[string]any{
			"id": "ce-1", "type": "commandExecution",
			"exitCode": 0, "aggregatedOutput": "ls output",
		},
	})

	writeNotification(f.PW, "turn/completed", map[string]any{
		"turn": map[string]any{"status": "completed"},
	})

	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.ToolUpdates) < 1 {
		t.Fatalf("expected at least 1 tool update, got %d", len(f.ToolUpdates))
	}
	if f.ToolUpdates[0].ToolName != "shell" || f.ToolUpdates[0].Status != "running" {
		t.Errorf("tool update[0] = %+v, want shell/running", f.ToolUpdates[0])
	}

	if len(f.Chunks) != 1 {
		t.Fatalf("expected 1 content chunk, got %d", len(f.Chunks))
	}
	if f.Chunks[0].Text != "Here is the output" || f.Chunks[0].Role != "assistant" {
		t.Errorf("chunk = %+v", f.Chunks[0])
	}

	if len(f.ToolResults) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(f.ToolResults))
	}
	if !f.ToolResults[0].Success || f.ToolResults[0].ResultText != "ls output" {
		t.Errorf("result = %+v", f.ToolResults[0])
	}

	if len(f.StateChanges) != 1 || f.StateChanges[0] != acp.StateIdle {
		t.Errorf("state changes = %v, want [idle]", f.StateChanges)
	}
}

func TestCodexTransport_ToolCallFlow(t *testing.T) {
	f := newFixture()
	defer f.Close()

	writeNotification(f.PW, "item/started", map[string]any{
		"item": map[string]any{"id": "tc-1", "type": "toolCall", "toolName": "read_file"},
	})
	writeNotification(f.PW, "item/completed", map[string]any{
		"item": map[string]any{"id": "tc-1", "type": "toolCall", "toolName": "read_file"},
	})
	f.Drain()

	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.ToolUpdates) != 2 {
		t.Fatalf("expected 2 tool updates, got %d", len(f.ToolUpdates))
	}
	if f.ToolUpdates[0].Status != "running" || f.ToolUpdates[0].ToolName != "read_file" {
		t.Errorf("update[0] = %+v, want running/read_file", f.ToolUpdates[0])
	}
	if f.ToolUpdates[1].Status != "completed" || f.ToolUpdates[1].ToolName != "read_file" {
		t.Errorf("update[1] = %+v, want completed/read_file", f.ToolUpdates[1])
	}
}

func TestCodexTransport_ReasoningFlow(t *testing.T) {
	f := newFixture()
	defer f.Close()

	writeNotification(f.PW, "item/reasoning/summaryTextDelta", reasoningDelta{
		ItemID: "r-1", Delta: "Let me think...",
	})
	writeNotification(f.PW, "item/reasoning/textDelta", reasoningDelta{
		ItemID: "r-1", Delta: "deeper analysis",
	})
	f.Drain()

	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.ThinkingTexts) != 2 {
		t.Fatalf("expected 2 thinking updates, got %d", len(f.ThinkingTexts))
	}
	if f.ThinkingTexts[0] != "Let me think..." {
		t.Errorf("thinking[0] = %q", f.ThinkingTexts[0])
	}
	if f.ThinkingTexts[1] != "deeper analysis" {
		t.Errorf("thinking[1] = %q", f.ThinkingTexts[1])
	}
}

func TestCodexTransport_MixedFlow(t *testing.T) {
	f := newFixture()
	defer f.Close()

	f.transport.sessionMu.Lock()
	f.transport.sessionID = "thread-mixed"
	f.transport.sessionMu.Unlock()

	writeNotification(f.PW, "item/reasoning/summaryTextDelta", reasoningDelta{Delta: "plan"})
	writeNotification(f.PW, "item/started", map[string]any{
		"item": map[string]any{"id": "tc-a", "type": "toolCall", "toolName": "Bash"},
	})
	writeNotification(f.PW, "item/completed", map[string]any{
		"item": map[string]any{"id": "tc-a", "type": "toolCall", "toolName": "Bash"},
	})
	writeNotification(f.PW, "item/agentMessage/delta", agentMessageDelta{Delta: "Done!"})
	writeNotification(f.PW, "item/plan/delta", planDelta{Delta: "Step 2: verify"})
	writeNotification(f.PW, "turn/completed", map[string]any{
		"turn": map[string]any{"status": "completed"},
	})
	f.Drain()

	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.ThinkingTexts) != 1 || f.ThinkingTexts[0] != "plan" {
		t.Errorf("thinking = %v", f.ThinkingTexts)
	}
	if len(f.ToolUpdates) != 2 {
		t.Errorf("tool updates = %d, want 2", len(f.ToolUpdates))
	}
	if len(f.Chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(f.Chunks))
	}
	if f.Chunks[0].Role != "assistant" || f.Chunks[1].Role != "plan" {
		t.Errorf("chunk roles = [%q, %q]", f.Chunks[0].Role, f.Chunks[1].Role)
	}
	if len(f.StateChanges) != 1 || f.StateChanges[0] != acp.StateIdle {
		t.Errorf("state changes = %v", f.StateChanges)
	}
}

func TestCodexTransport_TurnFailed_WithLog(t *testing.T) {
	f := newFixture()
	defer f.Close()

	writeNotification(f.PW, "turn/completed", map[string]any{
		"turn": map[string]any{
			"status": "failed",
			"error":  map[string]any{"message": "rate limit exceeded"},
		},
	})
	f.Drain()

	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.StateChanges) != 1 || f.StateChanges[0] != acp.StateIdle {
		t.Errorf("state changes = %v", f.StateChanges)
	}
	if len(f.LogMessages) != 1 {
		t.Fatalf("expected 1 log message, got %d", len(f.LogMessages))
	}
	if f.LogMessages[0] != "error:turn failed: rate limit exceeded" {
		t.Errorf("log = %q", f.LogMessages[0])
	}
}
