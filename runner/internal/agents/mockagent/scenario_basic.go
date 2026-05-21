package mockagent

import (
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// scenarioEcho echoes the prompt back as a single contentChunk, then completes
// the turn. Mirrors PTY mode's `got: <line>` semantics so test assertions can
// remain symmetric across modes.
func scenarioEcho(state *runtimeState, id int64, params json.RawMessage, _ *slog.Logger) error {
	prompt := extractPromptText(params)
	if err := emitContentChunk(state.writer, "echo: "+prompt, "assistant"); err != nil {
		return err
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}

// streamingChunkDelay is the gap between chunks in scenarios that test the
// frontend's streaming/in-progress render path. The default (80ms) is short
// enough that tests don't slow down but long enough that React render cycles
// observe intermediate state. Override via E2E_MOCK_CHUNK_DELAY_MS so timing-
// sensitive specs (slow CI, fast local) can dial it without recompiling.
var streamingChunkDelay = resolveChunkDelay()

func resolveChunkDelay() time.Duration {
	const fallback = 80 * time.Millisecond
	raw := os.Getenv("E2E_MOCK_CHUNK_DELAY_MS")
	if raw == "" {
		return fallback
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return fallback
	}
	return time.Duration(ms) * time.Millisecond
}

// scenarioStreaming3 emits the prompt response as three separate chunks
// (so consumers see assistant `complete=false` → `complete=true` transition).
// Canonical regression for the AcpActivityStream StreamingCaret and
// markLastMessageComplete pipeline.
func scenarioStreaming3(state *runtimeState, id int64, params json.RawMessage, _ *slog.Logger) error {
	prompt := extractPromptText(params)
	chunks := []string{"streaming: ", prompt + " ", "(done)"}
	for i, c := range chunks {
		if err := emitContentChunk(state.writer, c, "assistant"); err != nil {
			return err
		}
		if i < len(chunks)-1 {
			time.Sleep(streamingChunkDelay)
		}
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}

// scenarioThinkingThenAnswer emits a thinking chunk first (verifies the
// AcpActivityStream ThinkingIndicator + animate-spin spinner is rendered),
// then a content chunk and end-of-turn.
func scenarioThinkingThenAnswer(state *runtimeState, id int64, params json.RawMessage, _ *slog.Logger) error {
	prompt := extractPromptText(params)
	if err := emitThinkingChunk(state.writer, "Let me analyze the prompt: "+prompt); err != nil {
		return err
	}
	time.Sleep(streamingChunkDelay)
	if err := emitContentChunk(state.writer, "Answer to: "+prompt, "assistant"); err != nil {
		return err
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}
