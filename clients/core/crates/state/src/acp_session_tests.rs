use crate::acp_session::AcpSessionManager;
use crate::acp_types::*;

fn make_tool_call(id: &str, status: &str) -> AcpToolCall {
    AcpToolCall {
        id: id.to_string(),
        name: format!("tool_{id}"),
        status: status.to_string(),
        args: None,
        result_text: None,
        error_message: None,
        success: None,
        timestamp: 0,
    }
}

#[test]
fn get_or_create_session_returns_default() {
    let mut mgr = AcpSessionManager::new();
    let session = mgr.get_or_create_session("pod1");
    assert_eq!(session.state, AcpState::Idle);
    assert!(session.messages.is_empty());
    assert!(session.tool_calls.is_empty());
}

#[test]
fn get_session_returns_none_for_missing() {
    let mgr = AcpSessionManager::new();
    assert!(mgr.get_session("nonexistent").is_none());
}

#[test]
fn clear_session_removes_data() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_content_chunk("pod1", "hello", "assistant");
    mgr.clear_session("pod1");
    assert!(mgr.get_session("pod1").is_none());
}

#[test]
fn configuration_serde_roundtrip_preserves_capability() {
    let json = r#"{"permission_mode":"bypass","model":"","supported_permission_modes":["bypass","ask_dangerous","ask_any_write"]}"#;
    let cfg: AcpConfiguration = serde_json::from_str(json).unwrap();
    assert_eq!(
        cfg.supported_permission_modes,
        vec!["bypass", "ask_dangerous", "ask_any_write"]
    );
    let out = serde_json::to_string(&cfg).unwrap();
    assert!(out.contains("supported_permission_modes"), "got: {out}");
}

#[test]
fn update_configuration_preserves_capability_across_delta() {
    let mut mgr = AcpSessionManager::new();
    // Snapshot seeds capability + initial mode.
    mgr.update_configuration(
        "p",
        AcpConfiguration {
            permission_mode: "bypass".into(),
            model: String::new(),
            supported_permission_modes: vec![
                "bypass".into(),
                "ask_dangerous".into(),
                "ask_any_write".into(),
            ],
        },
    );
    // A configChanged delta (mode only, empty capability) must not wipe it.
    mgr.update_configuration(
        "p",
        AcpConfiguration {
            permission_mode: "ask_dangerous".into(),
            model: String::new(),
            supported_permission_modes: Vec::new(),
        },
    );
    let cfg = &mgr.get_session("p").unwrap().configuration;
    assert_eq!(cfg.permission_mode, "ask_dangerous");
    assert_eq!(
        cfg.supported_permission_modes,
        vec!["bypass", "ask_dangerous", "ask_any_write"]
    );
}

// --- Content chunk streaming ---

#[test]
fn content_chunk_creates_new_message() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_content_chunk("p", "hello", "assistant");
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.messages.len(), 1);
    assert_eq!(s.messages[0].text, "hello");
    assert_eq!(s.messages[0].role, "assistant");
    assert!(!s.messages[0].complete);
}

#[test]
fn content_chunk_accumulates_same_role() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_content_chunk("p", "hel", "assistant");
    mgr.add_content_chunk("p", "lo", "assistant");
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.messages.len(), 1);
    assert_eq!(s.messages[0].text, "hello");
}

#[test]
fn content_chunk_new_role_creates_new_message() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_content_chunk("p", "hello", "assistant");
    mgr.add_content_chunk("p", "hi", "user");
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.messages.len(), 2);
    assert_eq!(s.messages[1].role, "user");
}

#[test]
fn user_message_always_complete() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_content_chunk("p", "hi", "user");
    let s = mgr.get_session("p").unwrap();
    assert!(s.messages[0].complete);
}

#[test]
fn user_message_dedup() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_content_chunk("p", "hi", "user");
    mgr.add_content_chunk("p", "hi", "user");
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.messages.len(), 1);
}

#[test]
fn user_message_different_text_not_deduped() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_content_chunk("p", "hi", "user");
    mgr.add_content_chunk("p", "bye", "user");
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.messages.len(), 2);
}

#[test]
fn content_chunk_after_complete_creates_new() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_content_chunk("p", "first", "assistant");
    mgr.mark_last_message_complete("p");
    mgr.add_content_chunk("p", "second", "assistant");
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.messages.len(), 2);
    assert!(s.messages[0].complete);
    assert!(!s.messages[1].complete);
}

#[test]
fn messages_capped_at_max() {
    let mut mgr = AcpSessionManager::new();
    for i in 0..MAX_MESSAGES + 10 {
        mgr.add_content_chunk("p", &format!("msg{i}"), "user");
    }
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.messages.len(), MAX_MESSAGES);
    assert_eq!(s.messages[0].text, "msg10");
}

// --- Tool calls ---

#[test]
fn update_tool_call_creates_entry() {
    let mut mgr = AcpSessionManager::new();
    mgr.update_tool_call("p", make_tool_call("tc1", "running"));
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.tool_calls.len(), 1);
    assert_eq!(s.tool_calls["tc1"].status, "running");
}

#[test]
fn update_tool_call_preserves_timestamp() {
    let mut mgr = AcpSessionManager::new();
    mgr.update_tool_call("p", make_tool_call("tc1", "running"));
    let ts1 = mgr.get_session("p").unwrap().tool_calls["tc1"].timestamp;
    mgr.update_tool_call("p", make_tool_call("tc1", "running"));
    let ts2 = mgr.get_session("p").unwrap().tool_calls["tc1"].timestamp;
    assert_eq!(ts1, ts2);
}

#[test]
fn set_tool_call_result_updates_fields() {
    let mut mgr = AcpSessionManager::new();
    mgr.update_tool_call("p", make_tool_call("tc1", "running"));
    mgr.set_tool_call_result("p", "tc1", true, Some("ok".into()), None);
    let tc = &mgr.get_session("p").unwrap().tool_calls["tc1"];
    assert_eq!(tc.status, "completed");
    assert_eq!(tc.success, Some(true));
    assert_eq!(tc.result_text.as_deref(), Some("ok"));
    assert!(tc.error_message.is_none());
}

#[test]
fn set_tool_call_result_ignores_missing() {
    let mut mgr = AcpSessionManager::new();
    mgr.set_tool_call_result("p", "missing", true, None, None);
    let s = mgr.get_session("p").unwrap();
    assert!(s.tool_calls.is_empty());
}

#[test]
fn trim_tool_calls_evicts_completed_first() {
    let mut mgr = AcpSessionManager::new();
    for i in 0..MAX_TOOL_CALLS {
        let mut tc = make_tool_call(&format!("tc{i}"), "completed");
        tc.timestamp = i as i64;
        mgr.get_or_create_session("p")
            .tool_calls
            .insert(tc.id.clone(), tc);
    }
    let mut overflow = make_tool_call("overflow", "running");
    overflow.timestamp = MAX_TOOL_CALLS as i64;
    mgr.update_tool_call("p", overflow);

    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.tool_calls.len(), MAX_TOOL_CALLS);
    assert!(s.tool_calls.contains_key("overflow"));
    assert!(!s.tool_calls.contains_key("tc0"));
}

#[test]
fn trim_tool_calls_evicts_running_as_fallback() {
    let mut mgr = AcpSessionManager::new();
    for i in 0..MAX_TOOL_CALLS {
        let mut tc = make_tool_call(&format!("tc{i}"), "running");
        tc.timestamp = i as i64;
        mgr.get_or_create_session("p")
            .tool_calls
            .insert(tc.id.clone(), tc);
    }
    let mut overflow = make_tool_call("overflow", "running");
    overflow.timestamp = MAX_TOOL_CALLS as i64;
    mgr.update_tool_call("p", overflow);

    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.tool_calls.len(), MAX_TOOL_CALLS);
    assert!(s.tool_calls.contains_key("overflow"));
    assert!(!s.tool_calls.contains_key("tc0"));
}

// --- Thinking ---

#[test]
fn thinking_accumulates() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_thinking("p", "part1");
    mgr.add_thinking("p", " part2");
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.thinkings.len(), 1);
    assert_eq!(s.thinkings[0].text, "part1 part2");
    assert!(!s.thinkings[0].complete);
}

#[test]
fn thinking_new_round_after_seal() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_thinking("p", "round1");
    mgr.add_content_chunk("p", "x", "assistant"); // seals thinking
    mgr.add_thinking("p", "round2");
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.thinkings.len(), 2);
    assert!(s.thinkings[0].complete);
    assert!(!s.thinkings[1].complete);
}

#[test]
fn thinkings_capped() {
    let mut mgr = AcpSessionManager::new();
    for i in 0..MAX_THINKINGS + 5 {
        let session = mgr.get_or_create_session("p");
        session.thinkings.push(AcpThinking {
            text: format!("t{i}"),
            timestamp: i as i64,
            complete: true,
        });
    }
    mgr.add_thinking("p", "new");
    let s = mgr.get_session("p").unwrap();
    assert!(s.thinkings.len() <= MAX_THINKINGS);
}

// --- State transitions ---

#[test]
fn update_state_seals_thinking_and_marks_complete() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_thinking("p", "hmm");
    mgr.add_content_chunk("p", "resp", "assistant");
    mgr.update_session_state("p", AcpState::Idle);
    let s = mgr.get_session("p").unwrap();
    assert!(s.thinkings[0].complete);
    assert!(s.messages[0].complete);
    assert_eq!(s.state, AcpState::Idle);
}

#[test]
fn state_enum_roundtrip() {
    assert_eq!(AcpState::from_str_lossy("idle"), AcpState::Idle);
    assert_eq!(AcpState::from_str_lossy("processing"), AcpState::Processing);
    assert_eq!(
        AcpState::from_str_lossy("waiting_permission"),
        AcpState::WaitingPermission
    );
    assert_eq!(AcpState::from_str_lossy("unknown"), AcpState::Idle);
}

// --- Permissions ---

#[test]
fn permission_add_and_remove() {
    let mut mgr = AcpSessionManager::new();
    let req = AcpPermissionRequest {
        id: "r1".into(),
        tool_name: "bash".into(),
        args: None,
        description: Some("run ls".into()),
    };
    mgr.add_permission_request("p", req);
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.pending_permissions.len(), 1);

    mgr.remove_permission_request("p", "r1");
    let s = mgr.get_session("p").unwrap();
    assert!(s.pending_permissions.is_empty());
}

#[test]
fn remove_permission_on_missing_session_is_noop() {
    let mut mgr = AcpSessionManager::new();
    mgr.remove_permission_request("missing", "r1");
}

// --- Logs ---

#[test]
fn logs_capped_at_max() {
    let mut mgr = AcpSessionManager::new();
    for i in 0..MAX_LOGS + 5 {
        mgr.add_log("p", "error", &format!("log{i}"));
    }
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.logs.len(), MAX_LOGS);
    assert_eq!(s.logs[0].message, "log5");
}

// --- Plan ---

#[test]
fn update_plan_replaces_steps() {
    let mut mgr = AcpSessionManager::new();
    mgr.update_plan(
        "p",
        vec![AcpPlanStep {
            title: "step1".into(),
            status: "pending".into(),
        }],
    );
    mgr.update_plan(
        "p",
        vec![AcpPlanStep {
            title: "step2".into(),
            status: "done".into(),
        }],
    );
    let s = mgr.get_session("p").unwrap();
    assert_eq!(s.plan.len(), 1);
    assert_eq!(s.plan[0].title, "step2");
}

// --- Seal thinking on non-thinking events ---

#[test]
fn tool_call_seals_thinking() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_thinking("p", "think");
    mgr.update_tool_call("p", make_tool_call("tc1", "running"));
    assert!(mgr.get_session("p").unwrap().thinkings[0].complete);
}

#[test]
fn plan_update_seals_thinking() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_thinking("p", "think");
    mgr.update_plan("p", vec![]);
    assert!(mgr.get_session("p").unwrap().thinkings[0].complete);
}

#[test]
fn permission_request_seals_thinking() {
    let mut mgr = AcpSessionManager::new();
    mgr.add_thinking("p", "think");
    mgr.add_permission_request(
        "p",
        AcpPermissionRequest {
            id: "r1".into(),
            tool_name: "t".into(),
            args: None,
            description: None,
        },
    );
    assert!(mgr.get_session("p").unwrap().thinkings[0].complete);
}

#[test]
fn set_tool_call_result_seals_thinking() {
    let mut mgr = AcpSessionManager::new();
    mgr.update_tool_call("p", make_tool_call("tc1", "running"));
    mgr.add_thinking("p", "think");
    mgr.set_tool_call_result("p", "tc1", true, None, None);
    assert!(mgr.get_session("p").unwrap().thinkings[0].complete);
}

// --- Mark last message complete ---

#[test]
fn mark_last_message_complete_on_empty_is_noop() {
    let mut mgr = AcpSessionManager::new();
    mgr.mark_last_message_complete("missing");
}
