package acp

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLoopalState_BgTaskLifecycle(t *testing.T) {
	l := newLoopalState()
	l.apply("bgTask.spawned", json.RawMessage(`{"id":"bg1","description":"npm test","created_at_unix_ms":1}`))
	l.apply("bgTask.output", json.RawMessage(`{"id":"bg1","output_delta":"line1\n"}`))
	l.apply("bgTask.output", json.RawMessage(`{"id":"bg1","output_delta":"line2\n"}`))
	l.apply("bgTask.completed", json.RawMessage(`{"id":"bg1","status":"Completed","exit_code":0,"output":"done"}`))

	var out struct {
		Type    string `json:"type"`
		BgTasks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Output string `json:"output"`
		} `json:"bg_tasks"`
	}
	if err := json.Unmarshal(l.snapshot(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != "loopal.snapshot" {
		t.Errorf("type = %q", out.Type)
	}
	if len(out.BgTasks) != 1 || out.BgTasks[0].Status != "Completed" || out.BgTasks[0].Output != "done" {
		t.Errorf("bg_tasks = %+v", out.BgTasks)
	}
}

func TestLoopalState_SpawnedIdempotent(t *testing.T) {
	l := newLoopalState()
	ev := json.RawMessage(`{"id":"bg1","description":"x","created_at_unix_ms":1}`)
	l.apply("bgTask.spawned", ev)
	l.apply("bgTask.spawned", ev)
	var out struct {
		BgTasks []json.RawMessage `json:"bg_tasks"`
	}
	_ = json.Unmarshal(l.snapshot(), &out)
	if len(out.BgTasks) != 1 {
		t.Errorf("expected 1 bg task, got %d", len(out.BgTasks))
	}
}

func TestLoopalState_CronsFullReplace(t *testing.T) {
	l := newLoopalState()
	l.apply("crons", json.RawMessage(`{"crons":[{"id":"c1","cron_expr":"0 9 * * *"}]}`))
	var out struct {
		Crons []struct {
			ID string `json:"id"`
		} `json:"crons"`
	}
	if err := json.Unmarshal(l.snapshot(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Crons) != 1 || out.Crons[0].ID != "c1" {
		t.Errorf("crons = %+v", out.Crons)
	}
}

func TestLoopalState_EmptySnapshotNil(t *testing.T) {
	if newLoopalState().snapshot() != nil {
		t.Error("empty state should snapshot to nil")
	}
}

func TestLoopalState_TopologyAccumulateIdempotent(t *testing.T) {
	l := newLoopalState()
	ev := json.RawMessage(`{"spawn":{"name":"worker","agent_id":"a1","parent":{"agent":"main"},"model":"opus"}}`)
	l.apply("topology.spawn", ev)
	l.apply("topology.spawn", ev)
	var out struct {
		Topology []struct {
			Name   string `json:"name"`
			Parent string `json:"parent"`
			Model  string `json:"model"`
		} `json:"topology"`
	}
	if err := json.Unmarshal(l.snapshot(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Topology) != 1 {
		t.Fatalf("topology len = %d, want 1 (idempotent by agent_id)", len(out.Topology))
	}
	if out.Topology[0].Name != "worker" || out.Topology[0].Parent != "main" || out.Topology[0].Model != "opus" {
		t.Errorf("node = %+v", out.Topology[0])
	}
}

func TestLoopalState_OutputCapped(t *testing.T) {
	l := newLoopalState()
	l.apply("bgTask.spawned", json.RawMessage(`{"id":"bg1","description":"x","created_at_unix_ms":1}`))
	big := make([]byte, 100*1024)
	for i := range big {
		big[i] = 'x'
	}
	ev, _ := json.Marshal(map[string]string{"id": "bg1", "output_delta": string(big)})
	l.apply("bgTask.output", ev)
	if got := len(l.bgTasks["bg1"].Output); got > maxBgOutput {
		t.Errorf("output len = %d, want <= %d", got, maxBgOutput)
	}
}

func TestLoopalState_CompletedOutputCapped(t *testing.T) {
	l := newLoopalState()
	l.apply("bgTask.spawned", json.RawMessage(`{"id":"bg1","description":"x","created_at_unix_ms":1}`))
	big := make([]byte, 100*1024)
	for i := range big {
		big[i] = 'x'
	}
	ev, _ := json.Marshal(map[string]any{"id": "bg1", "status": "Completed", "output": string(big)})
	l.apply("bgTask.completed", ev)
	if got := len(l.bgTasks["bg1"].Output); got > maxBgOutput {
		t.Errorf("completed output len = %d, want <= %d", got, maxBgOutput)
	}
}

func TestLoopalState_Goal(t *testing.T) {
	l := newLoopalState()
	l.apply("goal", json.RawMessage(`{"goal":{"goal_id":"g1","objective":"ship","status":"active"}}`))
	var out struct {
		Goal struct {
			Objective string `json:"objective"`
			Status    string `json:"status"`
		} `json:"goal"`
	}
	if err := json.Unmarshal(l.snapshot(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Goal.Objective != "ship" || out.Goal.Status != "active" {
		t.Errorf("goal = %+v", out.Goal)
	}
}

func TestLoopalState_OutOfOrderUpsert(t *testing.T) {
	// output/completed arriving before spawned (relay reorder) must not be
	// dropped — the task is upserted so its data still reaches the snapshot.
	l := newLoopalState()
	l.apply("bgTask.output", json.RawMessage(`{"id":"bg9","output_delta":"early\n"}`))
	l.apply("bgTask.completed", json.RawMessage(`{"id":"bg9","status":"Completed","exit_code":0}`))
	t9 := l.bgTasks["bg9"]
	if t9 == nil {
		t.Fatal("out-of-order bg task was dropped")
	}
	if t9.Status != "Completed" || t9.Output != "early\n" {
		t.Errorf("bg9 = %+v, want status=Completed output=early", t9)
	}
}

func TestLoopalState_CompletedWithoutStatusPreservesRunning(t *testing.T) {
	// A completed event omitting status must not blank the prior status.
	l := newLoopalState()
	l.apply("bgTask.spawned", json.RawMessage(`{"id":"bg1","description":"x","created_at_unix_ms":1}`))
	l.apply("bgTask.completed", json.RawMessage(`{"id":"bg1","exit_code":0,"output":"done"}`))
	if got := l.bgTasks["bg1"].Status; got != "Running" {
		t.Errorf("status = %q, want Running (preserved, not blanked)", got)
	}
}

// TestLoopalState_FoldsCanonicalSequence pins the full fold of the canonical
// _loopal/* sequence (testdata/loopal_panel_signals.json). Its Rust mirror is
// loopal_dispatch_tests::folds_canonical_sequence — this Go snapshot cache and
// the Rust core reducer MUST fold identically (SSOT boundary), so both assert
// the same end state. A drift in either reducer turns one of these red.
func TestLoopalState_FoldsCanonicalSequence(t *testing.T) {
	l := newLoopalState()
	for _, s := range []struct{ kind, data string }{
		{"bgTask.spawned", `{"id":"bg1","description":"npm test","created_at_unix_ms":1717000000000}`},
		{"bgTask.output", `{"id":"bg1","output_delta":"running...\n"}`},
		{"bgTask.completed", `{"id":"bg1","status":"Completed","exit_code":0,"output":"done"}`},
		{"crons", `{"crons":[{"id":"c1","cron_expr":"0 9 * * *","prompt":"daily","recurring":true,"durable":true}]}`},
		{"tasks", `{"tasks":[{"id":"t1","subject":"build","status":"in_progress","blocked_by":[]}]}`},
		{"topology.spawn", `{"spawn":{"name":"worker","agent_id":"a1","parent":{"agent":"main"},"model":"opus"}}`},
		{"mcp", `{"servers":[{"name":"fs","status":"connected","tool_count":3}]}`},
		{"goal", `{"goal":{"goal_id":"g1","objective":"ship it","status":"active"}}`},
		{"mode", `{"mode":"plan"}`},
		{"thinking", `{"thinking":"{\"type\":\"effort\",\"level\":\"high\"}"}`},
		{"model", `{"model":"claude-opus-4-7"}`},
	} {
		l.apply(s.kind, json.RawMessage(s.data))
	}

	if bg := l.bgTasks["bg1"]; bg == nil || bg.Status != "Completed" || bg.Output != "done" {
		t.Errorf("bg1 = %+v, want Completed/done", bg)
	}
	if l.mode != "plan" || l.thinking != `{"type":"effort","level":"high"}` || l.model != "claude-opus-4-7" {
		t.Errorf("session params = %q / %q / %q", l.mode, l.thinking, l.model)
	}
	if len(l.topology) != 1 || l.topology[0].Name != "worker" || l.topology[0].AgentID != "a1" ||
		l.topology[0].Parent != "main" || l.topology[0].Model != "opus" {
		t.Errorf("topology = %+v", l.topology)
	}
	if !bytes.Contains(l.crons, []byte(`"c1"`)) || !bytes.Contains(l.tasks, []byte(`"t1"`)) ||
		!bytes.Contains(l.mcp, []byte(`"fs"`)) || !bytes.Contains(l.goal, []byte("ship it")) {
		t.Errorf("crons/tasks/mcp/goal content drift: crons=%s tasks=%s mcp=%s goal=%s",
			l.crons, l.tasks, l.mcp, l.goal)
	}
	if l.snapshot() == nil {
		t.Error("snapshot must be non-nil after the canonical sequence")
	}
}

func TestLoopalState_CompletedWithoutExitCodePreservesPrior(t *testing.T) {
	// A re-delivered completed lacking exit_code must not blank a prior code.
	l := newLoopalState()
	l.apply("bgTask.spawned", json.RawMessage(`{"id":"bg1","description":"x","created_at_unix_ms":1}`))
	l.apply("bgTask.completed", json.RawMessage(`{"id":"bg1","status":"Failed","exit_code":1}`))
	l.apply("bgTask.completed", json.RawMessage(`{"id":"bg1","status":"Failed"}`))
	if ec := l.bgTasks["bg1"].ExitCode; ec == nil || *ec != 1 {
		t.Errorf("exit_code = %v, want preserved 1", ec)
	}
}

func TestLoopalState_FoldsEdgeCaseContract(t *testing.T) {
	// Pins the subtle fold behaviors most likely to drift between this Go cache
	// and the Rust core reducer: out-of-order bgTask upsert (output before
	// spawned) and present-key preserve (an absent-key crons/mode event must NOT
	// wipe). Mirror: loopal_fold_contract_tests::folds_edge_case_contract.
	l := newLoopalState()
	for _, s := range []struct{ kind, data string }{
		{"bgTask.output", `{"id":"bgX","output_delta":"early\n"}`},
		{"bgTask.spawned", `{"id":"bgX","description":"task X","created_at_unix_ms":5}`},
		{"bgTask.completed", `{"id":"bgX","status":"Completed","exit_code":0}`},
		{"crons", `{"crons":[{"id":"c1","cron_expr":"* * * * *","prompt":"p","recurring":true,"durable":false}]}`},
		{"crons", `{}`},
		{"mode", `{"mode":"plan"}`},
		{"mode", `{}`},
		{"mode", `{"mode":""}`},
	} {
		l.apply(s.kind, json.RawMessage(s.data))
	}
	if bg := l.bgTasks["bgX"]; bg == nil || bg.Status != "Completed" || bg.Output != "early\n" || bg.Description != "task X" {
		t.Errorf("out-of-order upsert drift: %+v", bg)
	}
	if !bytes.Contains(l.crons, []byte(`"c1"`)) {
		t.Errorf("absent-key crons wiped (present-key drift): crons=%s", l.crons)
	}
	if l.mode != "plan" {
		t.Errorf("absent-key/empty-string mode wiped (present-key drift): mode=%q", l.mode)
	}
}
