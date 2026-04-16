package claude

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestHandleAssistant_ThinkingBlock(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant", "message": map[string]any{
		"content": []map[string]any{
			{"type": "thinking", "text": "let me consider this"},
		},
	}})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ThinkingTexts) != 1 || f.ThinkingTexts[0] != "let me consider this" {
		t.Errorf("thinking = %v, want [let me consider this]", f.ThinkingTexts)
	}
}

func TestHandleAssistant_ThinkingBlockEmpty(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant", "message": map[string]any{
		"content": []map[string]any{
			{"type": "thinking", "text": ""},
		},
	}})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ThinkingTexts) != 0 {
		t.Errorf("expected 0 thinking updates for empty text, got %d", len(f.ThinkingTexts))
	}
}

func TestHandleAssistant_ToolUseBlock(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant", "message": map[string]any{
		"content": []map[string]any{
			{"type": "tool_use", "id": "tool1", "name": "Read", "input": map[string]any{"file": "x.go"}},
		},
	}})
	f.Drain()
	_, _, tools, _ := f.Snap()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool update, got %d", len(tools))
	}
	if tools[0].ToolCallID != "tool1" || tools[0].ToolName != "Read" || tools[0].Status != "completed" {
		t.Errorf("tool update = %+v", tools[0])
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(tools[0].ArgumentsJSON), &args); err != nil {
		t.Fatalf("parse args: %v", err)
	}
	if args["file"] != "x.go" {
		t.Errorf("args = %v", args)
	}
}

func TestHandleAssistant_ToolUseBlock_NilInput(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant", "message": map[string]any{
		"content": []map[string]any{
			{"type": "tool_use", "id": "tool2", "name": "Write"},
		},
	}})
	f.Drain()
	_, _, tools, _ := f.Snap()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool update, got %d", len(tools))
	}
	if tools[0].ArgumentsJSON != "" {
		t.Errorf("expected empty args for nil input, got %q", tools[0].ArgumentsJSON)
	}
}

func TestHandleAssistant_EmptyMessage(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant"})
	f.Drain()
	_, chunks, tools, _ := f.Snap()
	if len(chunks) != 0 || len(tools) != 0 {
		t.Errorf("expected no events for empty message, chunks=%d tools=%d", len(chunks), len(tools))
	}
}

func TestHandleAssistant_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant", "message": "not an object"})
	f.Drain()
	_, chunks, tools, _ := f.Snap()
	if len(chunks) != 0 || len(tools) != 0 {
		t.Errorf("expected no events for parse error, chunks=%d tools=%d", len(chunks), len(tools))
	}
}

func TestHandleAssistant_TextBlockEmpty(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant", "message": map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": ""},
		},
	}})
	f.Drain()
	_, chunks, _, _ := f.Snap()
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text block, got %d", len(chunks))
	}
}

func TestHandleAssistant_MixedBlocks(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant", "message": map[string]any{
		"content": []map[string]any{
			{"type": "thinking", "text": "reasoning"},
			{"type": "text", "text": "answer"},
			{"type": "tool_use", "id": "t1", "name": "Bash", "input": map[string]any{"cmd": "ls"}},
		},
	}})
	f.Drain()
	_, chunks, tools, _ := f.Snap()
	f.mu.Lock()
	thinking := f.ThinkingTexts
	f.mu.Unlock()
	if len(thinking) != 1 || thinking[0] != "reasoning" {
		t.Errorf("thinking = %v", thinking)
	}
	if len(chunks) != 1 || chunks[0].Text != "answer" {
		t.Errorf("chunks = %v", chunks)
	}
	if len(tools) != 1 || tools[0].ToolName != "Bash" {
		t.Errorf("tools = %v", tools)
	}
}

func TestHandleAssistant_NilCallbacks(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	msg := &message{
		Type: "assistant",
		Message: json.RawMessage(`{"content":[
			{"type":"text","text":"hello"},
			{"type":"thinking","text":"hmm"},
			{"type":"tool_use","id":"t1","name":"Read","input":{"f":"x"}}
		]}`),
	}
	tr.handleAssistant(msg)
}

func TestHandleResult_ErrorWithContent(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{
		"type":    "result",
		"subtype": "error_max_turns",
		"result":  "Turn limit reached",
	})
	f.Drain()
	_, chunks, _, _ := f.Snap()
	if len(chunks) != 1 || chunks[0].Text != "Turn limit reached" {
		t.Errorf("expected error content chunk, got %v", chunks)
	}
}

func TestHandleResult_ErrorEmptyResult(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{
		"type":    "result",
		"subtype": "error_unknown",
	})
	f.Drain()
	states, chunks, _, _ := f.Snap()
	if len(states) != 1 || states[0] != acp.StateIdle {
		t.Errorf("states = %v", states)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty result, got %d", len(chunks))
	}
}

func TestHandleResult_EmptySubtype(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "result"})
	f.Drain()
	states, _, _, _ := f.Snap()
	if len(states) != 1 || states[0] != acp.StateIdle {
		t.Errorf("expected idle state for empty subtype, got %v", states)
	}
}

func TestHandleResult_NilCallbacks(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.handleResult(&message{Type: "result", Subtype: "success"})
	tr.handleResult(&message{Type: "result", Subtype: "error", Result: "fail"})
}
