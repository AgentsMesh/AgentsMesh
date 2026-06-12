use serde_json::json;

use crate::loopal_dispatch::dispatch_event;
use crate::loopal_session::LoopalSessionManager;

// Cross-reducer fold contract. The Go snapshot cache (runner
// acp::TestLoopalState_FoldsCanonicalSequence) and this Rust core reducer MUST
// fold the canonical _loopal/* sequence (runner
// testdata/loopal_panel_signals.json) to the same end state — both pin it here,
// so a drift in either reducer turns one of the two tests red.
#[test]
fn folds_canonical_sequence() {
    let mut mgr = LoopalSessionManager::new();
    let seq: &[(&str, serde_json::Value)] = &[
        ("loopal.bgTask.spawned", json!({"id":"bg1","description":"npm test","created_at_unix_ms":1_717_000_000_000u64})),
        ("loopal.bgTask.output", json!({"id":"bg1","output_delta":"running...\n"})),
        ("loopal.bgTask.completed", json!({"id":"bg1","status":"Completed","exit_code":0,"output":"done"})),
        ("loopal.crons", json!({"crons":[{"id":"c1","cron_expr":"0 9 * * *","prompt":"daily","recurring":true,"durable":true}]})),
        ("loopal.tasks", json!({"tasks":[{"id":"t1","subject":"build","status":"in_progress","blocked_by":[]}]})),
        ("loopal.topology.spawn", json!({"spawn":{"name":"worker","agent_id":"a1","parent":{"agent":"main"},"model":"opus"}})),
        ("loopal.mcp", json!({"servers":[{"name":"fs","status":"connected","tool_count":3}]})),
        ("loopal.goal", json!({"goal":{"goal_id":"g1","objective":"ship it","status":"active"}})),
        ("loopal.mode", json!({"mode":"plan"})),
        ("loopal.thinking", json!({"thinking":r#"{"type":"effort","level":"high"}"#})),
        ("loopal.model", json!({"model":"claude-opus-4-7"})),
    ];
    for (kind, data) in seq {
        dispatch_event(&mut mgr, "p", kind, data);
    }
    let s = mgr.get("p").unwrap();
    assert_eq!(s.bg_tasks.len(), 1);
    assert_eq!(s.bg_tasks[0].status, "Completed");
    assert_eq!(s.bg_tasks[0].output, "done");
    assert_eq!(s.crons.len(), 1);
    assert_eq!(s.tasks.len(), 1);
    assert_eq!(s.mcp.len(), 1);
    assert_eq!(s.topology.len(), 1);
    assert_eq!(s.topology[0].name, "worker");
    assert_eq!(s.topology[0].agent_id, "a1");
    assert_eq!(s.topology[0].parent.as_deref(), Some("main"));
    assert_eq!(s.topology[0].model.as_deref(), Some("opus"));
    assert_eq!(s.thread_goal.as_ref().unwrap().objective, "ship it");
    assert_eq!(s.mode.as_deref(), Some("plan"));
    assert_eq!(s.thinking.as_deref(), Some(r#"{"type":"effort","level":"high"}"#));
    assert_eq!(s.model.as_deref(), Some("claude-opus-4-7"));
}

// Mirror of Go acp::TestLoopalState_FoldsEdgeCaseContract: the subtle behaviors
// most likely to drift across the two reducers — out-of-order bgTask upsert and
// present-key preserve (absent-key crons/mode must not wipe).
#[test]
fn folds_edge_case_contract() {
    let mut mgr = LoopalSessionManager::new();
    let seq: &[(&str, serde_json::Value)] = &[
        ("loopal.bgTask.output", json!({"id":"bgX","output_delta":"early\n"})),
        ("loopal.bgTask.spawned", json!({"id":"bgX","description":"task X","created_at_unix_ms":5u64})),
        ("loopal.bgTask.completed", json!({"id":"bgX","status":"Completed","exit_code":0})),
        ("loopal.crons", json!({"crons":[{"id":"c1","cron_expr":"* * * * *","prompt":"p","recurring":true,"durable":false}]})),
        ("loopal.crons", json!({})),
        ("loopal.mode", json!({"mode":"plan"})),
        ("loopal.mode", json!({})),
        ("loopal.mode", json!({"mode":""})),
    ];
    for (kind, data) in seq {
        dispatch_event(&mut mgr, "p", kind, data);
    }
    let s = mgr.get("p").unwrap();
    assert_eq!(s.bg_tasks.len(), 1);
    assert_eq!(s.bg_tasks[0].id, "bgX");
    assert_eq!(s.bg_tasks[0].status, "Completed");
    assert_eq!(s.bg_tasks[0].output, "early\n");
    assert_eq!(s.bg_tasks[0].description, "task X");
    assert_eq!(s.crons.len(), 1, "absent-key crons must not wipe");
    assert_eq!(s.crons[0].id, "c1");
    assert_eq!(s.mode.as_deref(), Some("plan"), "absent-key and empty-string mode must not wipe");
}

// Rust-only resilience (the Go cache stores list blobs verbatim; only this
// reducer parses them): a single malformed list element is dropped rather than
// wiping the whole panel, matching the bg_tasks filter_map in dispatch_snapshot.
#[test]
fn parse_vec_drops_malformed_element() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.crons",
        &json!({"crons":[
            {"id":"good","cron_expr":"* * * * *","prompt":"p","recurring":true,"durable":false},
            {"cron_expr":"missing-id"}
        ]}),
    );
    let s = mgr.get("p").unwrap();
    assert_eq!(s.crons.len(), 1, "malformed element dropped, panel not wiped");
    assert_eq!(s.crons[0].id, "good");
}
