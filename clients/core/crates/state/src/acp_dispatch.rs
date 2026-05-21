use crate::acp_session::AcpSessionManager;
use crate::acp_types::*;
use serde_json::Value;

fn parse_args(data: &Value, field: &str) -> Option<serde_json::Value> {
    data.get(field)
        .and_then(|v| v.as_str())
        .and_then(|s| serde_json::from_str(s).ok())
}

pub fn dispatch_event(
    manager: &mut AcpSessionManager,
    pod_key: &str,
    _session_id: &str,
    event_type: &str,
    data: &Value,
) {
    match event_type {
        "contentChunk" => {
            let text = data["text"].as_str().unwrap_or("");
            let role = data["role"].as_str().unwrap_or("");
            manager.add_content_chunk(pod_key, text, role);
        }
        "toolCallUpdate" => {
            let tc = AcpToolCall {
                id: data["toolCallId"].as_str().unwrap_or("").to_string(),
                name: data["toolName"].as_str().unwrap_or("").to_string(),
                status: data["status"].as_str().unwrap_or("running").to_string(),
                args: parse_args(data, "argumentsJson"),
                result_text: None,
                error_message: None,
                success: None,
                timestamp: 0,
            };
            manager.update_tool_call(pod_key, tc);
        }
        "toolCallResult" => {
            let id = data["toolCallId"].as_str().unwrap_or("");
            let success = data["success"].as_bool().unwrap_or(false);
            let result_text = data["resultText"].as_str().map(String::from);
            let error_message = data["errorMessage"].as_str().map(String::from);
            manager.set_tool_call_result(pod_key, id, success, result_text, error_message);
        }
        "planUpdate" => {
            if let Some(steps_arr) = data["steps"].as_array() {
                let steps: Vec<AcpPlanStep> = steps_arr
                    .iter()
                    .filter_map(|s| {
                        Some(AcpPlanStep {
                            title: s["title"].as_str()?.to_string(),
                            status: s["status"].as_str().unwrap_or("pending").to_string(),
                        })
                    })
                    .collect();
                manager.update_plan(pod_key, steps);
            }
        }
        "thinkingUpdate" => {
            let text = data["text"].as_str().unwrap_or("");
            manager.add_thinking(pod_key, text);
        }
        "permissionRequest" => {
            let req = AcpPermissionRequest {
                id: data["requestId"].as_str().unwrap_or("").to_string(),
                tool_name: data["toolName"].as_str().unwrap_or("").to_string(),
                args: parse_args(data, "argumentsJson"),
                description: data["description"].as_str().map(String::from),
            };
            manager.add_permission_request(pod_key, req);
        }
        "sessionState" => {
            let state = AcpState::from_str_lossy(data["state"].as_str().unwrap_or("idle"));
            manager.update_session_state(pod_key, state);
        }
        "log" => {
            let level = data["level"].as_str().unwrap_or("");
            if level == "error" || level == "warn" {
                let message = data["message"].as_str().unwrap_or("");
                manager.add_log(pod_key, level, message);
            }
        }
        "configChanged" => {
            manager.update_configuration(
                pod_key,
                AcpConfiguration {
                    permission_mode: data["permissionMode"]
                        .as_str()
                        .unwrap_or("")
                        .to_string(),
                    model: data["model"].as_str().unwrap_or("").to_string(),
                },
            );
        }
        "configChangeFailed" => {
            // Surfaced in the UI as a log entry by the TS dispatcher (the wasm
            // session manager has no log-channel mutator for this case); the
            // Rust side just acknowledges the event so unknown-event-type warn
            // noise doesn't appear in console.
        }
        _ => {
            tracing::warn!("[ACP] Unknown event type: {}", event_type);
        }
    }
}

pub fn dispatch_snapshot(
    manager: &mut AcpSessionManager,
    pod_key: &str,
    session_id: &str,
    snapshot: &Value,
) {
    manager.clear_session(pod_key);

    if let Some(state_str) = snapshot["state"].as_str() {
        manager.update_session_state(pod_key, AcpState::from_str_lossy(state_str));
    }

    if let Some(plan_arr) = snapshot["plan"].as_array() {
        let steps: Vec<AcpPlanStep> = plan_arr
            .iter()
            .filter_map(|s| {
                Some(AcpPlanStep {
                    title: s["title"].as_str()?.to_string(),
                    status: s["status"].as_str().unwrap_or("pending").to_string(),
                })
            })
            .collect();
        manager.update_plan(pod_key, steps);
    }

    if let Some(tool_calls_arr) = snapshot["toolCalls"].as_array() {
        for tc_val in tool_calls_arr {
            let id = tc_val["toolCallId"].as_str().unwrap_or("").to_string();
            let tc = AcpToolCall {
                id: id.clone(),
                name: tc_val["toolName"].as_str().unwrap_or("").to_string(),
                status: tc_val["status"].as_str().unwrap_or("running").to_string(),
                args: parse_args(tc_val, "argumentsJson"),
                result_text: None,
                error_message: None,
                success: None,
                timestamp: 0,
            };
            manager.update_tool_call(pod_key, tc);

            if tc_val.get("success").and_then(|v| v.as_bool()).is_some() {
                manager.set_tool_call_result(
                    pod_key,
                    &id,
                    tc_val["success"].as_bool().unwrap_or(false),
                    tc_val["resultText"].as_str().map(String::from),
                    tc_val["errorMessage"].as_str().map(String::from),
                );
            }
        }
    }

    if let Some(messages) = snapshot["messages"].as_array() {
        for msg in messages {
            let text = msg["text"].as_str().unwrap_or("");
            let role = msg["role"].as_str().unwrap_or("");
            manager.add_content_chunk(pod_key, text, role);
        }
    }

    if let Some(thinkings) = snapshot["thinkings"].as_array() {
        for t in thinkings {
            let text = t["text"].as_str().unwrap_or("");
            if !text.is_empty() {
                manager.add_thinking(pod_key, text);
            }
        }
    }

    if let Some(logs) = snapshot["logs"].as_array() {
        for log in logs {
            let level = log["level"].as_str().unwrap_or("");
            let message = log["message"].as_str().unwrap_or("");
            if !level.is_empty() && !message.is_empty() {
                manager.add_log(pod_key, level, message);
            }
        }
    }

    if let Some(permissions) = snapshot["pendingPermissions"].as_array() {
        for perm in permissions {
            let req = AcpPermissionRequest {
                id: perm["requestId"].as_str().unwrap_or("").to_string(),
                tool_name: perm["toolName"].as_str().unwrap_or("").to_string(),
                args: parse_args(perm, "argumentsJson"),
                description: perm["description"].as_str().map(String::from),
            };
            manager.add_permission_request(pod_key, req);
        }
    }

    if let Some(cfg) = snapshot.get("configuration") {
        manager.update_configuration(
            pod_key,
            AcpConfiguration {
                permission_mode: cfg["permissionMode"].as_str().unwrap_or("").to_string(),
                model: cfg["model"].as_str().unwrap_or("").to_string(),
            },
        );
    }
    let _ = session_id;
}
