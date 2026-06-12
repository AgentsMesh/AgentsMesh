package mockagent

import (
	"encoding/json"
	"log/slog"
)

// scenarioLoopalPanels emits the full Loopal control-panel `_loopal/*` signal
// set on prompt so the browser console (/[org]/loopal/[podKey]) renders every
// panel end-to-end (browser ↔ relay ↔ runner ↔ mock agent). Field shapes mirror
// the shared golden fixture (runner/internal/runner/testdata/
// loopal_panel_signals.json) and Loopal crates/loopal-acp/src/translate/panel.rs.
func scenarioLoopalPanels(state *runtimeState, id int64, _ json.RawMessage, _ *slog.Logger) error {
	signals := []struct {
		kind string
		data map[string]any
	}{
		// bg1 stays Running so the dock's bg tab renders it — BgShellSection
		// filters to Running (mirrors loopal's render_bg_tasks); a Completed
		// shell would be hidden and the tab would collapse to count 0. The
		// bgTask.completed reducer path is covered by loopal_snapshot_test.go.
		{"bgTask.spawned", map[string]any{"id": "bg1", "description": "npm test", "created_at_unix_ms": 1717000000000}},
		{"bgTask.output", map[string]any{"id": "bg1", "output_delta": "running...\n"}},
		{"crons", map[string]any{"crons": []map[string]any{
			{"id": "c1", "cron_expr": "0 9 * * *", "prompt": "daily", "recurring": true, "durable": true},
		}}},
		{"tasks", map[string]any{"tasks": []map[string]any{
			{"id": "t1", "subject": "build", "status": "in_progress", "blocked_by": []string{}},
		}}},
		{"topology.spawn", map[string]any{"spawn": map[string]any{
			"name": "worker", "agent_id": "a1", "parent": map[string]any{"agent": "main"}, "model": "opus",
		}}},
		{"mcp", map[string]any{"servers": []map[string]any{
			{"name": "fs", "status": "connected", "tool_count": 3},
		}}},
		{"goal", map[string]any{"goal": map[string]any{
			"goal_id": "g1", "objective": "ship it", "status": "active",
		}}},
		{"mode", map[string]any{"mode": "plan"}},
		{"thinking", map[string]any{"thinking": `{"type":"effort","level":"high"}`}},
		{"model", map[string]any{"model": "claude-opus-4-7"}},
	}
	for _, s := range signals {
		if err := state.writer.WriteNotification("_loopal/"+s.kind, map[string]any{
			"sessionId": mockSessionID,
			"data":      s.data,
		}); err != nil {
			return err
		}
	}
	return state.writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}
