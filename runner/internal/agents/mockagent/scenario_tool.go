package mockagent

import (
	"encoding/json"
	"log/slog"
	"time"
)

// scenarioToolCallEdit emits a complete tool_call lifecycle without involving
// the permission system. The frontend's AcpToolCallCard should render the
// running spinner + animate-pulse on the in-progress state, then settle to
// the green ✓ icon when the result arrives.
func scenarioToolCallEdit(state *runtimeState, id int64, params json.RawMessage, _ *slog.Logger) error {
	prompt := extractPromptText(params)

	if err := emitContentChunk(state.writer, "Editing file for: "+prompt, "assistant"); err != nil {
		return err
	}
	const tcID = "tc-mock-edit-1"
	if err := emitToolCall(state.writer, tcID, "Edit", "in_progress"); err != nil {
		return err
	}
	// Give the UI a chance to render the in-progress state before the
	// completion arrives — regression for the AcpToolCallCard animate-pulse
	// breathing animation.
	time.Sleep(streamingChunkDelay)
	if err := emitToolCallUpdate(state.writer, tcID, "Edit", "completed", "Edited 1 line", ""); err != nil {
		return err
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}
