package acp

import (
	"sync"
	"testing"
	"time"
)

// TestACPClient_AllUpdateTypes verifies that all 6 session/update types
// are correctly dispatched to the appropriate callbacks.
func TestACPClient_AllUpdateTypes(t *testing.T) {
	var mu sync.Mutex
	var contentChunks []ContentChunk
	var thinkingUpdates []ThinkingUpdate
	var toolCallUpdates []ToolCallUpdate
	var toolCallResults []ToolCallResult
	var planUpdates []PlanUpdate

	client := startMockClientWithMode(t, mockModeMultiUpdate, EventCallbacks{
		OnContentChunk: func(_ string, chunk ContentChunk) {
			mu.Lock()
			contentChunks = append(contentChunks, chunk)
			mu.Unlock()
		},
		OnThinkingUpdate: func(_ string, update ThinkingUpdate) {
			mu.Lock()
			thinkingUpdates = append(thinkingUpdates, update)
			mu.Unlock()
		},
		OnToolCallUpdate: func(_ string, update ToolCallUpdate) {
			mu.Lock()
			toolCallUpdates = append(toolCallUpdates, update)
			mu.Unlock()
		},
		OnToolCallResult: func(_ string, result ToolCallResult) {
			mu.Lock()
			toolCallResults = append(toolCallResults, result)
			mu.Unlock()
		},
		OnPlanUpdate: func(_ string, update PlanUpdate) {
			mu.Lock()
			planUpdates = append(planUpdates, update)
			mu.Unlock()
		},
	})
	defer client.Stop()

	if err := client.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := client.SendPrompt("trigger all updates"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}

	// Wait for all callbacks to fire (agent sends 6 updates before response).
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		allReady := len(contentChunks) >= 2 &&
			len(thinkingUpdates) >= 1 &&
			len(toolCallUpdates) >= 2 &&
			len(toolCallResults) >= 1 &&
			len(planUpdates) >= 1
		mu.Unlock()
		if allReady {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timeout: chunks=%d think=%d tool=%d result=%d plan=%d",
				len(contentChunks), len(thinkingUpdates),
				len(toolCallUpdates), len(toolCallResults), len(planUpdates))
			mu.Unlock()
		case <-time.After(50 * time.Millisecond):
		}
	}

	mu.Lock()
	defer mu.Unlock()

	verifyContentChunks(t, contentChunks)
	verifyThinkingUpdates(t, thinkingUpdates)
	verifyToolCallUpdates(t, toolCallUpdates)
	verifyToolCallResults(t, toolCallResults)
	verifyPlanUpdates(t, planUpdates)
}

func verifyContentChunks(t *testing.T, chunks []ContentChunk) {
	t.Helper()
	// agent_message_chunk → role="assistant", user_message_chunk → role="user"
	var gotAssistant, gotUser bool
	for _, c := range chunks {
		if c.Role == "assistant" && c.Text == "thinking..." {
			gotAssistant = true
		}
		if c.Role == "user" && c.Text == "echoed user message" {
			gotUser = true
		}
	}
	if !gotAssistant {
		t.Error("missing assistant content chunk with text 'thinking...'")
	}
	if !gotUser {
		t.Error("missing user content chunk with text 'echoed user message'")
	}
}

func verifyThinkingUpdates(t *testing.T, updates []ThinkingUpdate) {
	t.Helper()
	if updates[0].Text != "let me think" {
		t.Errorf("thinking text = %q, want 'let me think'", updates[0].Text)
	}
}

func verifyToolCallUpdates(t *testing.T, updates []ToolCallUpdate) {
	t.Helper()
	// tool_call (pending → normalized to running) + tool_call_update (completed)
	var gotRunning, gotCompleted bool
	for _, u := range updates {
		if u.ToolCallID != "tc-001" {
			t.Errorf("unexpected toolCallId: %s", u.ToolCallID)
		}
		if u.Status == "running" {
			gotRunning = true
		}
		if u.Status == "completed" {
			gotCompleted = true
		}
	}
	if !gotRunning {
		t.Error("missing tool_call with status 'running' (normalized from pending)")
	}
	if !gotCompleted {
		t.Error("missing tool_call_update with status 'completed'")
	}
}

func verifyToolCallResults(t *testing.T, results []ToolCallResult) {
	t.Helper()
	if results[0].ToolCallID != "tc-001" {
		t.Errorf("result toolCallId = %s", results[0].ToolCallID)
	}
	if !results[0].Success {
		t.Error("expected tool call result success=true")
	}
}

func verifyPlanUpdates(t *testing.T, updates []PlanUpdate) {
	t.Helper()
	if len(updates[0].Steps) != 2 {
		t.Fatalf("plan steps = %d, want 2", len(updates[0].Steps))
	}
	if updates[0].Steps[0].Title != "Step 1" || updates[0].Steps[0].Status != "completed" {
		t.Errorf("step 0: title=%q status=%q", updates[0].Steps[0].Title, updates[0].Steps[0].Status)
	}
	if updates[0].Steps[1].Title != "Step 2" || updates[0].Steps[1].Status != "in_progress" {
		t.Errorf("step 1: title=%q status=%q", updates[0].Steps[1].Title, updates[0].Steps[1].Status)
	}
}
