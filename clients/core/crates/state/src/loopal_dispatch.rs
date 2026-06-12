use serde::de::DeserializeOwned;
use serde_json::Value;

use crate::loopal_session::LoopalSessionManager;
use crate::loopal_types::{AgentNode, BgTask, GoalInfo};

// Per-element parse (mirrors the bg_tasks filter_map in dispatch_snapshot): a
// single malformed element is dropped, not allowed to wipe the whole panel the
// way an all-or-nothing Vec deserialize would.
fn parse_vec<T: DeserializeOwned>(data: &Value, key: &str) -> Vec<T> {
    data.get(key)
        .and_then(Value::as_array)
        .map(|arr| {
            arr.iter()
                .filter_map(|v| serde_json::from_value(v.clone()).ok())
                .collect()
        })
        .unwrap_or_default()
}

pub fn dispatch_event(
    mgr: &mut LoopalSessionManager,
    pod_key: &str,
    event_type: &str,
    data: &Value,
) {
    let kind = event_type.strip_prefix("loopal.").unwrap_or(event_type);
    match kind {
        "bgTask.spawned" => mgr.bg_task_spawned(
            pod_key,
            data["id"].as_str().unwrap_or_default(),
            data["description"].as_str().unwrap_or_default(),
            data["created_at_unix_ms"].as_u64().unwrap_or(0),
        ),
        "bgTask.output" => mgr.bg_task_output(
            pod_key,
            data["id"].as_str().unwrap_or_default(),
            data["output_delta"].as_str().unwrap_or_default(),
        ),
        "bgTask.completed" => mgr.bg_task_completed(
            pod_key,
            data["id"].as_str().unwrap_or_default(),
            data["status"].as_str().unwrap_or_default(),
            data["exit_code"].as_i64().map(|v| v as i32),
            data["output"].as_str().unwrap_or_default(),
        ),
        // Present-key guard (matches goal/mode/thinking/model + dispatch_snapshot):
        // a malformed event missing its list key must not wipe the panel;
        // {"crons": []} still legitimately clears it.
        "crons" => {
            if data.get("crons").is_some() {
                mgr.set_crons(pod_key, parse_vec(data, "crons"));
            }
        }
        "tasks" => {
            if data.get("tasks").is_some() {
                mgr.set_tasks(pod_key, parse_vec(data, "tasks"));
            }
        }
        "mcp" => {
            if data.get("servers").is_some() {
                mgr.set_mcp(pod_key, parse_vec(data, "servers"));
            }
        }
        "topology.spawn" => {
            let spawn = &data["spawn"];
            mgr.add_agent(
                pod_key,
                AgentNode {
                    name: spawn["name"].as_str().unwrap_or_default().to_string(),
                    agent_id: spawn["agent_id"].as_str().unwrap_or_default().to_string(),
                    parent: spawn["parent"]["agent"].as_str().filter(|s| !s.is_empty()).map(String::from),
                    model: spawn["model"].as_str().filter(|s| !s.is_empty()).map(String::from),
                },
            );
        }
        "goal" => {
            // Present-key guard (matches dispatch_snapshot + mode/thinking/model):
            // a malformed event missing "goal" must not wipe the thread goal;
            // {"goal": null} still legitimately clears it.
            if data.get("goal").is_some() {
                let goal = serde_json::from_value::<Option<GoalInfo>>(data["goal"].clone())
                    .unwrap_or_default();
                mgr.set_goal(pod_key, goal);
            }
        }
        // Present-key + non-empty semantics, mirroring the Go accumulator
        // (loopal_snapshot.go `if v != ""`): a malformed event (field absent /
        // non-string / empty) must not wipe a previously-set value.
        "mode" => {
            if let Some(m) = data.get("mode").and_then(|v| v.as_str()).filter(|s| !s.is_empty()) {
                mgr.set_mode(pod_key, Some(m.to_string()));
            }
        }
        "thinking" => {
            if let Some(t) = data.get("thinking").and_then(|v| v.as_str()).filter(|s| !s.is_empty()) {
                mgr.set_thinking(pod_key, Some(t.to_string()));
            }
        }
        "model" => {
            if let Some(m) = data.get("model").and_then(|v| v.as_str()).filter(|s| !s.is_empty()) {
                mgr.set_model(pod_key, Some(m.to_string()));
            }
        }
        _ => {}
    }
}

pub fn dispatch_snapshot(mgr: &mut LoopalSessionManager, pod_key: &str, snapshot: &Value) {
    // Present-key semantics: each carried field is a full replace; absent fields
    // are preserved. The two snapshot sources differ — Loopal's view_snapshot
    // carries bg_tasks/crons/tasks/mcp (single-agent, no global topology); the
    // runner accumulator carries whatever it has observed, including topology.
    // Each source rebuilds only the fields it owns, so a Loopal snapshot never
    // wipes the topology the runner accumulated from SubAgentSpawned events.
    if let Some(arr) = snapshot["bg_tasks"].as_array() {
        let tasks = arr
            .iter()
            .filter_map(|v| {
                let id = v["id"].as_str()?;
                Some(BgTask {
                    id: id.to_string(),
                    description: v["description"].as_str().unwrap_or_default().to_string(),
                    status: v["status"].as_str().unwrap_or("Running").to_string(),
                    exit_code: v["exit_code"].as_i64().map(|x| x as i32),
                    output: v["output"].as_str().unwrap_or_default().to_string(),
                    created_at_unix_ms: v["created_at_unix_ms"].as_u64().unwrap_or(0),
                })
            })
            .collect();
        mgr.set_bg_tasks(pod_key, tasks);
    }
    if snapshot.get("crons").is_some() {
        mgr.set_crons(pod_key, parse_vec(snapshot, "crons"));
    }
    if snapshot.get("tasks").is_some() {
        mgr.set_tasks(pod_key, parse_vec(snapshot, "tasks"));
    }
    if snapshot.get("mcp").is_some() {
        mgr.set_mcp(pod_key, parse_vec(snapshot, "mcp"));
    }
    if snapshot.get("topology").is_some() {
        mgr.set_topology(pod_key, parse_vec(snapshot, "topology"));
    }
    if snapshot.get("goal").is_some() {
        let goal = serde_json::from_value::<Option<GoalInfo>>(snapshot["goal"].clone())
            .unwrap_or_default();
        mgr.set_goal(pod_key, goal);
    }
    if let Some(m) = snapshot.get("mode").and_then(|v| v.as_str()).filter(|s| !s.is_empty()) {
        mgr.set_mode(pod_key, Some(m.to_string()));
    }
    if let Some(t) = snapshot.get("thinking").and_then(|v| v.as_str()).filter(|s| !s.is_empty()) {
        mgr.set_thinking(pod_key, Some(t.to_string()));
    }
    if let Some(m) = snapshot.get("model").and_then(|v| v.as_str()).filter(|s| !s.is_empty()) {
        mgr.set_model(pod_key, Some(m.to_string()));
    }
}
