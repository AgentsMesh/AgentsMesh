use serde_json::json;

use crate::loopal_dispatch::{dispatch_event, dispatch_snapshot};
use crate::loopal_session::LoopalSessionManager;

#[test]
fn bg_task_spawned_inserts() {
    let mut mgr = LoopalSessionManager::new();
    let data = json!({"id": "bg1", "description": "npm test", "created_at_unix_ms": 1_717_000_000_000u64});
    dispatch_event(&mut mgr, "p", "loopal.bgTask.spawned", &data);
    let s = mgr.get("p").unwrap();
    assert_eq!(s.bg_tasks.len(), 1);
    assert_eq!(s.bg_tasks[0].id, "bg1");
    assert_eq!(s.bg_tasks[0].status, "Running");
}

#[test]
fn bg_task_spawned_is_idempotent() {
    let mut mgr = LoopalSessionManager::new();
    let data = json!({"id": "bg1", "description": "x", "created_at_unix_ms": 1u64});
    dispatch_event(&mut mgr, "p", "loopal.bgTask.spawned", &data);
    dispatch_event(&mut mgr, "p", "loopal.bgTask.spawned", &data);
    assert_eq!(mgr.get("p").unwrap().bg_tasks.len(), 1);
}

#[test]
fn bg_task_output_appends() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.spawned",
        &json!({"id":"bg1","description":"x","created_at_unix_ms":1u64}),
    );
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.output",
        &json!({"id":"bg1","output_delta":"line1\n"}),
    );
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.output",
        &json!({"id":"bg1","output_delta":"line2\n"}),
    );
    assert_eq!(mgr.get("p").unwrap().bg_tasks[0].output, "line1\nline2\n");
}

#[test]
fn bg_task_completed_sets_status_and_exit() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.spawned",
        &json!({"id":"bg1","description":"x","created_at_unix_ms":1u64}),
    );
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.completed",
        &json!({"id":"bg1","status":"Completed","exit_code":0,"output":"done"}),
    );
    let t = &mgr.get("p").unwrap().bg_tasks[0];
    assert_eq!(t.status, "Completed");
    assert_eq!(t.exit_code, Some(0));
    assert_eq!(t.output, "done");
}

#[test]
fn crons_replace_full_list() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.crons",
        &json!({"crons":[{"id":"c1","cron_expr":"0 9 * * *","prompt":"daily","recurring":true,"durable":true}]}),
    );
    let s = mgr.get("p").unwrap();
    assert_eq!(s.crons.len(), 1);
    assert_eq!(s.crons[0].cron_expr, "0 9 * * *");
    assert!(s.crons[0].recurring);
}

#[test]
fn tasks_replace_full_list() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.tasks",
        &json!({"tasks":[{"id":"t1","subject":"build","status":"in_progress","blocked_by":[]}]}),
    );
    let s = mgr.get("p").unwrap();
    assert_eq!(s.tasks.len(), 1);
    assert_eq!(s.tasks[0].subject, "build");
}

#[test]
fn mcp_replace_full_list() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.mcp",
        &json!({"servers":[{"name":"fs","status":"connected","tool_count":3}]}),
    );
    let s = mgr.get("p").unwrap();
    assert_eq!(s.mcp.len(), 1);
    assert_eq!(s.mcp[0].tool_count, 3);
}

#[test]
fn topology_spawn_adds_node_idempotent() {
    let mut mgr = LoopalSessionManager::new();
    let ev = json!({"spawn":{"name":"worker","agent_id":"a1","parent":{"agent":"main"}}});
    dispatch_event(&mut mgr, "p", "loopal.topology.spawn", &ev);
    dispatch_event(&mut mgr, "p", "loopal.topology.spawn", &ev);
    let s = mgr.get("p").unwrap();
    assert_eq!(s.topology.len(), 1);
    assert_eq!(s.topology[0].name, "worker");
    assert_eq!(s.topology[0].parent.as_deref(), Some("main"));
}

#[test]
fn unknown_panel_event_creates_no_session() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(&mut mgr, "p", "loopal.bogus", &json!({"reason": "x"}));
    assert!(mgr.get("p").is_none());
}

#[test]
fn snapshot_rebuilds_bg_crons_tasks_and_drops_stale() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.spawned",
        &json!({"id":"stale","description":"old","created_at_unix_ms":1u64}),
    );
    let snap = json!({
        "bg_tasks": [
            {"id":"bg1","description":"build","status":"Running","exit_code":null,"output":"","created_at_unix_ms":2u64}
        ],
        "crons": [{"id":"c1","cron_expr":"* * * * *","prompt":"p","recurring":true,"durable":false}],
        "tasks": [{"id":"t1","subject":"s","status":"pending","blocked_by":[]}]
    });
    dispatch_snapshot(&mut mgr, "p", &snap);
    let s = mgr.get("p").unwrap();
    assert_eq!(s.bg_tasks.len(), 1);
    assert!(s.bg_tasks.iter().all(|t| t.id != "stale"));
    assert_eq!(s.crons.len(), 1);
    assert_eq!(s.tasks.len(), 1);
}

#[test]
fn snapshot_rebuilds_topology_and_mcp() {
    let mut mgr = LoopalSessionManager::new();
    let snap = json!({
        "bg_tasks": [],
        "topology": [{"name":"worker","agent_id":"a1","parent":"main","model":"opus"}],
        "mcp": [{"name":"fs","status":"connected","tool_count":3}]
    });
    dispatch_snapshot(&mut mgr, "p", &snap);
    let s = mgr.get("p").unwrap();
    assert_eq!(s.topology.len(), 1);
    assert_eq!(s.topology[0].name, "worker");
    assert_eq!(s.topology[0].parent.as_deref(), Some("main"));
    assert_eq!(s.topology[0].model.as_deref(), Some("opus"));
    assert_eq!(s.mcp.len(), 1);
    assert_eq!(s.mcp[0].tool_count, 3);
}

#[test]
fn bg_task_output_capped_keeps_tail() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.spawned",
        &json!({"id":"bg1","description":"x","created_at_unix_ms":1u64}),
    );
    let big = "x".repeat(100_000);
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.output",
        &json!({"id":"bg1","output_delta": big}),
    );
    let out = &mgr.get("p").unwrap().bg_tasks[0].output;
    assert!(out.len() <= 64 * 1024, "output len = {}", out.len());
}

#[test]
fn bg_tasks_capped_drops_oldest() {
    let mut mgr = LoopalSessionManager::new();
    for i in 0..250 {
        dispatch_event(
            &mut mgr,
            "p",
            "loopal.bgTask.spawned",
            &json!({"id": format!("bg{i}"), "description":"x", "created_at_unix_ms":1u64}),
        );
    }
    let s = mgr.get("p").unwrap();
    assert_eq!(s.bg_tasks.len(), 200);
    assert!(s.bg_tasks.iter().all(|t| t.id != "bg0"));
}

#[test]
fn bg_task_completed_output_capped() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.spawned",
        &json!({"id":"bg1","description":"x","created_at_unix_ms":1u64}),
    );
    let big = "x".repeat(100_000);
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.completed",
        &json!({"id":"bg1","status":"Completed","exit_code":0,"output": big}),
    );
    let out = &mgr.get("p").unwrap().bg_tasks[0].output;
    assert!(out.len() <= 64 * 1024, "completed output len = {}", out.len());
}

#[test]
fn full_replace_lists_capped() {
    let mut mgr = LoopalSessionManager::new();
    let crons: Vec<_> = (0..600).map(|i| json!({"id": format!("c{i}")})).collect();
    dispatch_event(&mut mgr, "p", "loopal.crons", &json!({ "crons": crons }));
    assert!(
        mgr.get("p").unwrap().crons.len() <= 500,
        "crons len = {}",
        mgr.get("p").unwrap().crons.len()
    );
}

#[test]
fn topology_spawn_captures_model() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.topology.spawn",
        &json!({"spawn":{"name":"worker","agent_id":"a1","parent":{"agent":"main"},"model":"claude-opus-4"}}),
    );
    let s = mgr.get("p").unwrap();
    assert_eq!(s.topology[0].model.as_deref(), Some("claude-opus-4"));
}

#[test]
fn snapshot_without_topology_key_preserves_accumulated_topology() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.topology.spawn",
        &json!({"spawn":{"name":"worker","agent_id":"a1","parent":{"agent":"main"}}}),
    );
    // A Loopal view_snapshot carries bg_tasks/crons/tasks/mcp but no topology
    // (single-agent view). Present-key semantics must not wipe the topology the
    // runner accumulated from SubAgentSpawned events.
    dispatch_snapshot(
        &mut mgr,
        "p",
        &json!({"bg_tasks": [], "crons": [], "tasks": [], "mcp": []}),
    );
    assert_eq!(mgr.get("p").unwrap().topology.len(), 1);
}

#[test]
fn goal_set_and_cleared() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.goal",
        &json!({"goal":{"goal_id":"g1","objective":"ship it","status":"active"}}),
    );
    assert_eq!(
        mgr.get("p").unwrap().thread_goal.as_ref().unwrap().objective,
        "ship it"
    );
    dispatch_event(&mut mgr, "p", "loopal.goal", &json!({ "goal": null }));
    assert!(mgr.get("p").unwrap().thread_goal.is_none());
}

#[test]
fn mode_thinking_model_set_and_replace() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(&mut mgr, "p", "loopal.mode", &json!({"mode": "plan"}));
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.thinking",
        &json!({"thinking": r#"{"type":"effort","level":"high"}"#}),
    );
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.model",
        &json!({"model": "claude-opus-4-7"}),
    );
    let s = mgr.get("p").unwrap();
    assert_eq!(s.mode.as_deref(), Some("plan"));
    assert_eq!(s.thinking.as_deref(), Some(r#"{"type":"effort","level":"high"}"#));
    assert_eq!(s.model.as_deref(), Some("claude-opus-4-7"));

    dispatch_event(&mut mgr, "p", "loopal.mode", &json!({"mode": "act"}));
    assert_eq!(mgr.get("p").unwrap().mode.as_deref(), Some("act"));
}

#[test]
fn bg_task_out_of_order_upserts() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.output",
        &json!({"id":"bg9","output_delta":"early\n"}),
    );
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.completed",
        &json!({"id":"bg9","status":"Completed","exit_code":0}),
    );
    let t = &mgr.get("p").unwrap().bg_tasks[0];
    assert_eq!(t.id, "bg9");
    assert_eq!(t.status, "Completed");
    assert_eq!(t.output, "early\n");
}

#[test]
fn bg_task_completed_without_status_preserves_prior() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.spawned",
        &json!({"id":"bg1","description":"x","created_at_unix_ms":1u64}),
    );
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.bgTask.completed",
        &json!({"id":"bg1","exit_code":0,"output":"done"}),
    );
    assert_eq!(mgr.get("p").unwrap().bg_tasks[0].status, "Running");
}

#[test]
fn malformed_session_param_event_does_not_wipe() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(&mut mgr, "p", "loopal.mode", &json!({"mode": "plan"}));
    dispatch_event(&mut mgr, "p", "loopal.mode", &json!({}));
    dispatch_event(&mut mgr, "p", "loopal.mode", &json!({"mode": null}));
    assert_eq!(mgr.get("p").unwrap().mode.as_deref(), Some("plan"));
}

#[test]
fn goal_event_without_key_preserves_prior() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.goal",
        &json!({"goal":{"goal_id":"g1","objective":"x","status":"active"}}),
    );
    dispatch_event(&mut mgr, "p", "loopal.goal", &json!({}));
    assert!(
        mgr.get("p").unwrap().thread_goal.is_some(),
        "missing-key goal event must not wipe the goal"
    );
    dispatch_event(&mut mgr, "p", "loopal.goal", &json!({"goal": null}));
    assert!(
        mgr.get("p").unwrap().thread_goal.is_none(),
        "explicit null must clear the goal"
    );
}

#[test]
fn topology_spawn_skips_empty_agent_id() {
    let mut mgr = LoopalSessionManager::new();
    dispatch_event(
        &mut mgr,
        "p",
        "loopal.topology.spawn",
        &json!({"spawn":{"name":"x","agent_id":"","parent":{"agent":"main"}}}),
    );
    assert!(mgr.get("p").map(|s| s.topology.is_empty()).unwrap_or(true));
}
