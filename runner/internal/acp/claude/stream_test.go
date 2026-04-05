package claude

import (
	"testing"
	"time"
)

func TestTransport_TextDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 0,
		"delta": map[string]any{"type": "text_delta", "text": "Hello "},
	}})
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 0,
		"delta": map[string]any{"type": "text_delta", "text": "world!"},
	}})
	f.Drain()
	_, chunks, _, _ := f.Snap()
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Text != "Hello " || chunks[1].Text != "world!" {
		t.Errorf("chunks = %v", chunks)
	}
	if chunks[0].Role != "assistant" {
		t.Errorf("role = %q", chunks[0].Role)
	}
}

func TestTransport_ToolUseStreaming(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_start", "index": 1,
		"content_block": map[string]any{"type": "tool_use", "id": "t1", "name": "Read"},
	}})
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 1,
		"delta": map[string]any{"type": "input_json_delta", "partial_json": `{"file":`},
	}})
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 1,
		"delta": map[string]any{"type": "input_json_delta", "partial_json": `"main.go"}`},
	}})
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_stop", "index": 1,
	}})
	f.Drain()
	_, _, tools, _ := f.Snap()
	if len(tools) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(tools))
	}
	if tools[0].Status != "running" || tools[0].ToolName != "Read" {
		t.Errorf("first = %+v", tools[0])
	}
	if tools[1].Status != "completed" || tools[1].ArgumentsJSON != `{"file":"main.go"}` {
		t.Errorf("second = %+v", tools[1])
	}
}

func TestTransport_ThinkingDelta(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 0,
		"delta": map[string]any{"type": "thinking_delta", "text": "hmm"},
	}})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.ThinkingTexts) != 1 || f.ThinkingTexts[0] != "hmm" {
		t.Errorf("thinking = %v", f.ThinkingTexts)
	}
}

func TestTransport_ContentBlockStop_NonToolUse(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_stop", "index": 0,
	}})
	f.Drain()
	_, _, tools, _ := f.Snap()
	if len(tools) != 0 {
		t.Errorf("expected 0 tool updates, got %d", len(tools))
	}
}

func TestTransport_AssistantDeduplicatedWithStream(t *testing.T) {
	// When stream_event text_delta has already delivered content,
	// the final assistant message's text blocks are skipped (deduplication).
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 0,
		"delta": map[string]any{"type": "text_delta", "text": "Hello"},
	}})
	writeLine(f.PW, map[string]any{"type": "assistant", "message": map[string]any{
		"role": "assistant", "content": []map[string]any{{"type": "text", "text": "Hello"}},
	}})
	f.Drain()
	_, chunks, tools, _ := f.Snap()
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk (from stream_event only), got %d", len(chunks))
	}
	if len(tools) != 0 {
		t.Errorf("expected 0 tool updates, got %d", len(tools))
	}
}

func TestTransport_MultiTurnConversation(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "system", "subtype": "init", "session_id": "s1"})
	time.Sleep(50 * time.Millisecond)
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 0,
		"delta": map[string]any{"type": "text_delta", "text": "Turn 1"},
	}})
	writeLine(f.PW, map[string]any{"type": "result", "subtype": "success"})
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 0,
		"delta": map[string]any{"type": "text_delta", "text": "Turn 2"},
	}})
	writeLine(f.PW, map[string]any{"type": "result", "subtype": "success"})
	f.Drain()
	states, chunks, _, _ := f.Snap()
	if len(states) != 2 {
		t.Errorf("expected 2 state changes, got %d: %v", len(states), states)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
}

func TestTransport_InputJsonDelta_UnknownIndex(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 99,
		"delta": map[string]any{"type": "input_json_delta", "partial_json": `{"x":1}`},
	}})
	f.Drain()
	f.Transport.toolCallsMu.Lock()
	n := len(f.Transport.toolCalls)
	f.Transport.toolCallsMu.Unlock()
	if n != 0 {
		t.Errorf("expected 0 tracked tool calls, got %d", n)
	}
}
