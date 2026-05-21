package mockagent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// scenarioFailAfter1s acknowledges the prompt and then exits with code 1
// after a short delay, simulating an agent CLI crash mid-session. The
// runner's OnExit callback should fire, the pod transitions to stopped,
// and the UI must not be wedged in "processing" indefinitely.
func scenarioFailAfter1s(state *runtimeState, id int64, params json.RawMessage, logger *slog.Logger) error {
	prompt := extractPromptText(params)
	_ = emitContentChunk(state.writer, "Will crash soon: "+prompt, "assistant")
	_ = state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
	time.Sleep(1 * time.Second)
	logger.Info("mock simulating crash via os.Exit(1)")
	os.Exit(1)
	return nil // unreachable
}

// scenarioMalformedJSON writes a malformed JSON line directly to stdout
// followed by a valid response — exercises the reader's tolerance for
// non-JSON output (Claude CLI occasionally interleaves human-readable
// noise on stdout). The valid follow-up message proves the runtime
// recovers without losing the session.
func scenarioMalformedJSON(state *runtimeState, id int64, params json.RawMessage, _ *slog.Logger) error {
	prompt := extractPromptText(params)
	// Bypass the writer to emit raw garbage — the writer would refuse a
	// non-JSON payload, but the reader on the runner side must already
	// be defensive (see runner/internal/acp/reader.go line 33).
	if _, err := fmt.Fprintln(os.Stdout, "this is not json"); err != nil {
		return err
	}
	if err := emitContentChunk(state.writer, "recovered: "+prompt, "assistant"); err != nil {
		return err
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}

// scenarioToolCallFailed emits a tool_call that completes with status=failed
// and a populated errorMessage. The UI's AcpToolCallCard should render the
// red ✗ icon (success=false branch in ToolStatusIcon) and surface the error
// message when expanded.
func scenarioToolCallFailed(state *runtimeState, id int64, params json.RawMessage, _ *slog.Logger) error {
	prompt := extractPromptText(params)
	if err := emitContentChunk(state.writer, "Trying to edit: "+prompt, "assistant"); err != nil {
		return err
	}
	const tcID = "tc-mock-edit-fail-1"
	if err := emitToolCall(state.writer, tcID, "Edit", "in_progress"); err != nil {
		return err
	}
	time.Sleep(streamingChunkDelay)
	if err := emitToolCallUpdate(state.writer, tcID, "Edit", "failed", "", "file not found"); err != nil {
		return err
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}

// scenarioLogWarnings emits both a warn-level and an error-level log entry
// before completing the turn. The frontend's AcpActivityStream surfaces
// warn/error logs through LogEntry components (info-level is suppressed by
// design — see acpEventDispatcher.ts log case).
func scenarioLogWarnings(state *runtimeState, id int64, params json.RawMessage, logger *slog.Logger) error {
	prompt := extractPromptText(params)
	// stderr lines flow through OnLog in the runner; we synthesize them by
	// using fmt.Fprintln to os.Stderr so the message_handler_acp wrapper
	// observes a real log event with level="stderr". For deterministic
	// level tagging, the runner accepts info/warn/error in OnLog payload.
	fmt.Fprintln(os.Stderr, "warn: degraded connection to upstream")
	fmt.Fprintln(os.Stderr, "error: skipping file due to permission denied")
	logger.Info("mock emitted synthetic warn+error logs")
	if err := emitContentChunk(state.writer, "Completed with warnings: "+prompt, "assistant"); err != nil {
		return err
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}
