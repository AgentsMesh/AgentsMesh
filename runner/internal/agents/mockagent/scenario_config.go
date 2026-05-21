package mockagent

import (
	"encoding/json"
	"log/slog"
)

// scenarioConfigChangePlan is the canonical fixture for the control-plane
// round-trip: the scenario emits an initial content chunk acknowledging the
// pod is ready, then exits the turn. The interesting behavior comes later
// when the UI invokes Selector → relay → runner → session/control_request →
// mock acks → OnConfigChange fires → relay broadcasts configChanged →
// React selector reads new value. Tests assert end-to-end without the
// scenario emitting anything proactively.
//
// Naming: kept aligned with the intended Selector-driven flow even though
// the scenario itself is content-light — its value is in being a stable
// slot the mock binary's handleControlRequest path is tested against.
func scenarioConfigChangePlan(state *runtimeState, id int64, params json.RawMessage, _ *slog.Logger) error {
	prompt := extractPromptText(params)
	if err := emitContentChunk(state.writer, "Ready for mode switches (initial prompt: "+prompt+")", "assistant"); err != nil {
		return err
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}
