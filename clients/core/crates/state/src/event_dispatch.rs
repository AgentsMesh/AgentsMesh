use agentsmesh_events::event_types::EventType;
use agentsmesh_events::types::RealtimeEvent;
use agentsmesh_types::proto_pod_v1::Pod;
use agentsmesh_types::proto_runner_api_v1::Runner;
use agentsmesh_types::proto_ticket_v1::Ticket;
use serde_json::Value;

use crate::app_state::{AppState, NotificationSpec, ToastSpec};
use crate::autopilot_state::{AutopilotController, AutopilotIteration};
use crate::channel_types::{ChannelMessage, SenderAgentInfo, SenderPodInfo, SenderUser};
use crate::loop_state::{LoopRunData, loop_run_status};

/// Extract an int64 from a protojson event field. The backend serializes
/// proto `int64` as a JSON **string** (protojson UseProtoNames convention),
/// so a naive `as_i64()` returns None on the wire. Accept both number and
/// numeric-string encodings.
fn ji64(data: &Value, key: &str) -> Option<i64> {
    let v = data.get(key)?;
    v.as_i64().or_else(|| v.as_str().and_then(|s| s.parse::<i64>().ok()))
}

fn jstr<'a>(data: &'a Value, key: &str) -> Option<&'a str> {
    data.get(key).and_then(|v| v.as_str())
}

/// Build a state `ChannelMessage` from a `ChannelMessageEventData`
/// protojson object. The event shape differs from the state ChannelMessage
/// (it carries `sender_name` instead of a full `sender_user`, and int64
/// fields are strings), so a blanket `serde_json::from_value` won't work —
/// we map field-by-field. Mirrors the mapping the old JS handler did.
fn channel_message_from_event(data: &Value) -> Option<ChannelMessage> {
    let id = ji64(data, "id")?;
    let channel_id = ji64(data, "channel_id")?;
    let sender_user_id = ji64(data, "sender_user_id");
    let sender_name = jstr(data, "sender_name").unwrap_or("");

    let sender_user = match (sender_user_id, sender_name.is_empty()) {
        (Some(uid), false) => Some(SenderUser {
            id: uid,
            name: Some(sender_name.to_string()),
            username: sender_name.to_string(),
            ..Default::default()
        }),
        _ => None,
    };

    let sender_pod_info = data.get("sender_pod_info").and_then(|p| {
        let pod_key = jstr(p, "pod_key")?.to_string();
        let agent = p.get("agent").and_then(|a| {
            jstr(a, "name").map(|name| SenderAgentInfo {
                name: name.to_string(),
                ..Default::default()
            })
        });
        Some(SenderPodInfo {
            pod_key,
            alias: jstr(p, "alias").map(String::from),
            agent,
        })
    });

    Some(ChannelMessage {
        id,
        channel_id,
        sender_pod: jstr(data, "sender_pod").map(String::from),
        sender_pod_info,
        sender_user_id,
        sender_user,
        body: jstr(data, "body").unwrap_or("").to_string(),
        content_json: jstr(data, "content_json").map(String::from),
        mentions_json: jstr(data, "mentions_json").map(String::from),
        reply_to: ji64(data, "reply_to"),
        message_type: jstr(data, "message_type").map(String::from),
        created_at: jstr(data, "created_at").map(String::from),
        ..Default::default()
    })
}

pub fn dispatch(state: &mut AppState, event: &RealtimeEvent) {
    match event.event_type {
        EventType::PodCreated | EventType::PodRestarting => {
            // Try to apply directly if the payload carries a full Pod;
            // otherwise queue a refetch by pod_key — backend can publish
            // either shape depending on the trigger path.
            if let Ok(pod) = serde_json::from_value::<Pod>(event.data.clone()) {
                state.pods.upsert_pod(pod, Some(event.timestamp));
            } else if let Some(key) = event.data.get("pod_key").and_then(|v| v.as_str()) {
                state.pending_refetch_pod_keys.push(key.to_string());
            }
        }
        EventType::PodStatusChanged => {
            if let Some(key) = event.data.get("pod_key").and_then(|v| v.as_str()) {
                let status = event.data.get("status").and_then(|v| v.as_str()).unwrap_or("");
                let agent = event.data.get("agent_status").and_then(|v| v.as_str());
                let code = event.data.get("error_code").and_then(|v| v.as_str());
                let msg = event.data.get("error_message").and_then(|v| v.as_str());
                if state.pods.get_pod(key).is_some() {
                    state.pods.update_pod_status(key, status, agent, code, msg, Some(event.timestamp));
                } else {
                    // Pod missing from cache (e.g. created on another tab before this
                    // subscriber connected) — refetch the pod by key.
                    state.pending_refetch_pod_keys.push(key.to_string());
                }
                // Keep the mesh graph node in sync (no-op if not a topology node).
                state.mesh.update_node_status(key, status, agent);
            }
        }
        EventType::PodAgentStatusChanged => {
            if let Some(key) = event.data.get("pod_key").and_then(|v| v.as_str()) {
                let status = event.data.get("status").and_then(|v| v.as_str()).unwrap_or("");
                let agent = event.data.get("agent_status").and_then(|v| v.as_str());
                state.pods.update_pod_status(key, status, agent, None, None, Some(event.timestamp));
                state.mesh.update_node_status(key, status, agent);
            }
        }
        EventType::PodTerminated => {
            if let Some(key) = event.data.get("pod_key").and_then(|v| v.as_str()) {
                state.pods.update_pod_status(key, "terminated", None, None, None, Some(event.timestamp));
                state.mesh.update_node_status(key, "terminated", None);
            }
        }
        EventType::PodTitleChanged => {
            if let Some(key) = event.data.get("pod_key").and_then(|v| v.as_str()) {
                if let Some(title) = event.data.get("title").and_then(|v| v.as_str()) {
                    state.pods.update_pod_title(key, title, Some(event.timestamp));
                }
            }
        }
        EventType::PodAliasChanged => {
            if let Some(key) = event.data.get("pod_key").and_then(|v| v.as_str()) {
                if let Some(alias) = event.data.get("alias").and_then(|v| v.as_str()) {
                    state.pods.update_pod_alias(key, alias);
                }
            }
        }
        EventType::PodInitProgress => {
            if let Some(key) = event.data.get("pod_key").and_then(|v| v.as_str()) {
                let phase = event.data.get("phase").and_then(|v| v.as_str()).unwrap_or("");
                let progress = event.data.get("progress").and_then(|v| v.as_f64()).unwrap_or(0.0);
                let message = event.data.get("message").and_then(|v| v.as_str());
                state.pods.update_init_progress(key, phase, progress, message);
            }
        }
        EventType::ChannelMessage => {
            if let Some(msg) = channel_message_from_event(&event.data) {
                state.channels.on_new_message(msg);
            }
        }
        EventType::ChannelMessageEdited => {
            // Edited carries id/channel_id + new body/content/mentions/edited_at.
            if let (Some(id), Some(channel_id)) =
                (ji64(&event.data, "id"), ji64(&event.data, "channel_id"))
            {
                let patch = ChannelMessage {
                    id,
                    channel_id,
                    body: jstr(&event.data, "body").unwrap_or("").to_string(),
                    content_json: jstr(&event.data, "content_json").map(String::from),
                    mentions_json: jstr(&event.data, "mentions_json").map(String::from),
                    edited_at: jstr(&event.data, "edited_at").map(String::from),
                    ..Default::default()
                };
                state.channels.update_message(channel_id, patch);
            }
        }
        EventType::ChannelMessageDeleted => {
            if let (Some(ch), Some(id)) =
                (ji64(&event.data, "channel_id"), ji64(&event.data, "id"))
            {
                state.channels.remove_message(ch, id);
            }
        }
        EventType::TicketCreated => {
            // Wire carries only {slug,status} — full ticket data is pulled
            // by the platform refetch (queued). Try a direct insert if the
            // payload happens to be a full Ticket (number int64 in tests).
            if let Ok(t) = serde_json::from_value::<Ticket>(event.data.clone()) {
                state.tickets.add_ticket(t);
            } else if let Some(slug) = jstr(&event.data, "slug") {
                state.pending_refetch_ticket_slugs.push(slug.to_string());
            }
        }
        EventType::TicketUpdated | EventType::TicketStatusChanged | EventType::TicketMoved => {
            // The realtime business fact is the status transition; full
            // ticket fields (assignees, labels) come via refetch. Apply the
            // status in place so the board/badge updates immediately.
            if let Some(slug) = jstr(&event.data, "slug") {
                if let Some(status) = jstr(&event.data, "status") {
                    if !status.is_empty() {
                        state.tickets.update_ticket_status(slug, status);
                    }
                }
                state.pending_refetch_ticket_slugs.push(slug.to_string());
            }
        }
        EventType::TicketDeleted => {
            if let Some(slug) = jstr(&event.data, "slug") {
                state.tickets.remove_ticket(slug);
            }
        }
        EventType::RunnerOnline => {
            if let Some(id) = ji64(&event.data, "runner_id") {
                state.runners.update_runner_status(id, "online");
            }
        }
        EventType::RunnerOffline => {
            if let Some(id) = ji64(&event.data, "runner_id") {
                state.runners.update_runner_status(id, "offline");
            }
        }
        EventType::RunnerUpdated => {
            // Wire is RunnerStatusEventData {runner_id, node_id, status, ...}
            // — not a full Runner. Apply the status transition in place.
            if let Some(id) = ji64(&event.data, "runner_id") {
                let status = jstr(&event.data, "status").unwrap_or("");
                if !status.is_empty() {
                    state.runners.update_runner_status(id, status);
                }
            }
        }
        EventType::LoopRunStarted => {
            // Full run row is pulled by the platform refetch (fetchRuns).
            // Queue it; if the payload is a full LoopRunData (tests) insert.
            if let Ok(run) = serde_json::from_value::<LoopRunData>(event.data.clone()) {
                state.loops.add_run(run);
            }
        }
        EventType::LoopRunCompleted => {
            if let Some(id) = ji64(&event.data, "run_id").or_else(|| ji64(&event.data, "id")) {
                state.loops.update_run_status(id, loop_run_status::COMPLETED);
            }
        }
        EventType::LoopRunFailed => {
            if let Some(id) = ji64(&event.data, "run_id").or_else(|| ji64(&event.data, "id")) {
                state.loops.update_run_status(id, loop_run_status::FAILED);
            }
        }
        EventType::AutopilotStatusChanged => {
            if let Some(key) = event.data.get("autopilot_controller_key").and_then(|v| v.as_str()) {
                let ctrls: Vec<AutopilotController> = serde_json::from_value(
                    serde_json::Value::Array(
                        state.autopilot.controllers().iter()
                            .map(|c| serde_json::to_value(c).unwrap_or_default())
                            .collect()
                    )
                ).unwrap_or_default();
                if let Some(mut c) = ctrls.into_iter().find(|c| c.autopilot_controller_key == key) {
                    if let Some(phase) = event.data.get("phase").and_then(|v| v.as_str()) {
                        c.phase = Some(phase.to_string());
                    }
                    if let Some(cur) = ji64(&event.data, "current_iteration") {
                        c.current_iteration = Some(cur);
                    }
                    if let Some(max) = ji64(&event.data, "max_iterations") {
                        c.max_iterations = Some(max);
                    }
                    state.autopilot.update_controller(key, c);
                }
            }
        }
        EventType::AutopilotIteration => {
            if let Some(key) = event.data.get("autopilot_controller_key").and_then(|v| v.as_str()) {
                if let Ok(iter) = serde_json::from_value::<AutopilotIteration>(event.data.clone()) {
                    state.autopilot.add_iteration(key.to_string(), iter);
                }
            }
        }
        EventType::AutopilotCreated => {
            if let Ok(c) = serde_json::from_value::<AutopilotController>(event.data.clone()) {
                state.autopilot.add_controller(c);
            }
        }
        EventType::AutopilotTerminated => {
            if let Some(key) = event.data.get("autopilot_controller_key").and_then(|v| v.as_str()) {
                state.autopilot.remove_controller(key);
            }
        }
        EventType::AutopilotThinking => {
            if let Some(key) = event.data.get("autopilot_controller_key").and_then(|v| v.as_str()) {
                state.autopilot.update_thinking(key.to_string(), event.data.clone());
            }
        }

        // ── Channel members (Phase 3) ──
        EventType::ChannelMemberAdded => {
            if let Some(ch) = ji64(&event.data, "channel_id") {
                state.channels.patch_member_count(ch, 1);
            }
        }
        EventType::ChannelMemberRemoved => {
            if let Some(ch) = ji64(&event.data, "channel_id") {
                state.channels.patch_member_count(ch, -1);
            }
        }

        // ── Pod perpetual toggle (Phase 3) ──
        EventType::PodPerpetualChanged => {
            if let Some(key) = event.data.get("pod_key").and_then(|v| v.as_str()) {
                let perpetual = event.data.get("perpetual").and_then(|v| v.as_bool()).unwrap_or(false);
                state.pods.patch_perpetual(key, perpetual);
            }
        }

        // ── MR / Pipeline indirect events (Phase 3) ──
        // The event payload only carries `{ticket_slug, pod_id}`. The full
        // MR/Pipeline data must be fetched via Connect-RPC. Push refetch
        // keys into pending_* queues — platforms drain on next tick.
        EventType::MrCreated
        | EventType::MrUpdated
        | EventType::MrMerged
        | EventType::MrClosed
        | EventType::PipelineUpdated => {
            if let Some(slug) = event.data.get("ticket_slug").and_then(|v| v.as_str()) {
                state.pending_refetch_ticket_slugs.push(slug.to_string());
            }
            if let Some(pod_id) = event.data.get("pod_id").and_then(|v| v.as_i64()) {
                // Resolve pod_id → pod_key against current cache. Platform
                // refetches by key. If not found, skip — the event will be
                // re-emitted when the pod is created.
                let key = state.pods.pods().iter()
                    .find(|p| p.id == pod_id)
                    .map(|p| p.pod_key.clone());
                if let Some(k) = key {
                    state.pending_refetch_pod_keys.push(k);
                }
            }
        }

        // ── Notification → browser notification queue (Phase 3) ──
        EventType::Notification => {
            let title = event.data.get("title").and_then(|v| v.as_str()).unwrap_or("").to_string();
            let body = event.data.get("body").and_then(|v| v.as_str()).unwrap_or("").to_string();
            let icon = event.data.get("icon").and_then(|v| v.as_str()).map(String::from);
            let link = event.data.get("link").and_then(|v| v.as_str()).map(String::from);
            state.pending_browser_notifications.push(NotificationSpec {
                title, body, icon, link,
            });
        }

        // ── LoopRunWarning → toast queue (Phase 3) ──
        // Handled separately from LoopRunStarted/Completed/Failed even
        // though those share the LoopRunFailed arm above — warnings carry
        // additional context (run_number + detail) for the toast.
        EventType::LoopRunWarning => {
            let run_number = event.data.get("run_number").and_then(|v| v.as_i64()).unwrap_or(0);
            let detail = event.data.get("detail").and_then(|v| v.as_str()).unwrap_or("").to_string();
            let warning = event.data.get("warning").and_then(|v| v.as_str()).unwrap_or("").to_string();
            let description = if detail.is_empty() { warning } else { detail };
            state.pending_toasts.push(ToastSpec {
                kind: "warning".into(),
                title_key: "loops.runWarningTitle".into(),
                title_params: serde_json::json!({"runNumber": run_number}),
                description,
                duration_ms: 8000,
            });
        }

        // ── Blockstore op (Phase 3) ──
        // Blockstore has its own apply path through `BlockstoreState`
        // which is NOT yet part of `AppState` (the blockstore lives in a
        // separate service for now). For now, we silently no-op here and
        // rely on the existing `crates/state/src/blockstore_apply.rs`
        // pipeline that BlocksPage wires up directly.
        // TODO Phase 3b: thread `blockstore: BlockstoreState` into
        //   AppState + call `blockstore_apply::apply_op` here.
        EventType::BlockstoreOp => {}

        // ── System maintenance banner (Phase 3) ──
        // Backend doesn't currently publish this, but if/when it does,
        // queue a toast with the maintenance message.
        EventType::SystemMaintenance => {
            let message = event.data.get("message").and_then(|v| v.as_str()).unwrap_or("").to_string();
            state.pending_toasts.push(ToastSpec {
                kind: "info".into(),
                title_key: "system.maintenanceMode".into(),
                title_params: serde_json::json!({}),
                description: message,
                duration_ms: 0, // 0 = persistent until dismissed
            });
        }

        _ => {}
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    fn make_event(event_type: EventType, data: serde_json::Value) -> RealtimeEvent {
        RealtimeEvent {
            event_type, category: None, organization_id: 1,
            target_user_id: None, target_user_ids: None,
            entity_type: None, entity_id: None, data, timestamp: 1000,
        }
    }

    #[test]
    fn pod_created() {
        let mut s = AppState::new();
        let e = make_event(EventType::PodCreated, json!({"pod_key":"p1","status":"running","agent_slug":"claude"}));
        dispatch(&mut s, &e);
        assert_eq!(s.pods.pods().len(), 1);
        assert_eq!(s.pods.get_pod("p1").unwrap().status, "running");
    }

    #[test]
    fn pod_terminated() {
        let mut s = AppState::new();
        s.pods.upsert_pod(Pod {
            pod_key: "p1".into(), status: "running".into(),
            agent_slug: "claude".into(), ..Default::default()
        }, Some(100));
        dispatch(&mut s, &make_event(EventType::PodTerminated, json!({"pod_key":"p1"})));
        assert_eq!(s.pods.get_pod("p1").unwrap().status, "terminated");
    }

    #[test]
    fn channel_message() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::ChannelMessage, json!({"id":1,"channel_id":10,"body":"hi"})));
        assert_eq!(s.channels.get_messages(10).unwrap().messages.len(), 1);
        assert_eq!(s.channels.get_messages(10).unwrap().messages[0].body, "hi");
    }

    // Regression: the backend serializes proto int64 as JSON *strings*
    // (protojson). event_dispatch must parse string-encoded int64s, not
    // just numbers — otherwise channel_id/id extraction silently fails and
    // no state mutation happens (the bug that left desktop/web realtime
    // dead before the SSOT cutover).
    #[test]
    fn channel_message_protojson_string_int64() {
        let mut s = AppState::new();
        s.channels.set_channels(vec![agentsmesh_types::proto_channel_state_v1::Channel {
            id: 10, name: "gen".into(), member_count: Some(1), ..Default::default()
        }]);
        s.channels.set_current_user_id(Some(1));
        dispatch(&mut s, &make_event(EventType::ChannelMessage, json!({
            "id": "55", "channel_id": "10", "sender_user_id": "2",
            "sender_name": "bob", "body": "wire hi", "message_type": "text",
        })));
        let msgs = &s.channels.get_messages(10).unwrap().messages;
        assert_eq!(msgs.len(), 1);
        assert_eq!(msgs[0].id, 55);
        assert_eq!(msgs[0].body, "wire hi");
        // unread incremented (other user, not viewing)
        assert_eq!(s.channels.get_unread_count(10), 1);
    }

    #[test]
    fn channel_member_added_protojson_string_int64() {
        let mut s = AppState::new();
        s.channels.set_channels(vec![agentsmesh_types::proto_channel_state_v1::Channel {
            id: 10, name: "gen".into(), member_count: Some(1), ..Default::default()
        }]);
        dispatch(&mut s, &make_event(EventType::ChannelMemberAdded, json!({
            "channel_id": "10", "user_id": "3", "role": "member",
        })));
        assert_eq!(s.channels.get_channel(10).unwrap().member_count, Some(2));
    }

    #[test]
    fn channel_message_edited_protojson() {
        let mut s = AppState::new();
        s.channels.add_message(10, ChannelMessage {
            id: 55, channel_id: 10, body: "old".into(), ..Default::default()
        });
        dispatch(&mut s, &make_event(EventType::ChannelMessageEdited, json!({
            "id": "55", "channel_id": "10", "body": "new", "edited_at": "2026-01-01T00:00:00Z",
        })));
        let m = &s.channels.get_messages(10).unwrap().messages[0];
        assert_eq!(m.body, "new");
        assert_eq!(m.edited_at.as_deref(), Some("2026-01-01T00:00:00Z"));
    }

    #[test]
    fn channel_message_deleted_protojson() {
        let mut s = AppState::new();
        s.channels.add_message(10, ChannelMessage { id: 55, channel_id: 10, body: "x".into(), ..Default::default() });
        dispatch(&mut s, &make_event(EventType::ChannelMessageDeleted, json!({"id":"55","channel_id":"10"})));
        assert_eq!(s.channels.get_messages(10).unwrap().messages.len(), 0);
    }

    #[test]
    fn ticket_created() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::TicketCreated, json!({"slug":"T-1","title":"Fix","status":"todo","priority":"high"})));
        assert_eq!(s.tickets.get_tickets().len(), 1);
    }

    #[test]
    fn runner_online() {
        let mut s = AppState::new();
        s.runners.set_runners(vec![Runner {
            id: 1, node_id: "r1".into(), status: "offline".into(),
            max_concurrent_pods: 4, current_pods: 0,
            is_enabled: true, ..Default::default()
        }]);
        // Wire RunnerStatusEventData carries `runner_id` as a protojson string.
        dispatch(&mut s, &make_event(EventType::RunnerOnline, json!({"runner_id":"1","node_id":"r1","status":"online"})));
        assert_eq!(s.runners.get_runner(1).unwrap().status, "online");
    }

    #[test]
    fn runner_updated_protojson() {
        let mut s = AppState::new();
        s.runners.set_runners(vec![Runner {
            id: 7, node_id: "r7".into(), status: "online".into(),
            max_concurrent_pods: 4, is_enabled: true, ..Default::default()
        }]);
        dispatch(&mut s, &make_event(EventType::RunnerUpdated, json!({"runner_id":"7","node_id":"r7","status":"maintenance"})));
        assert_eq!(s.runners.get_runner(7).unwrap().status, "maintenance");
    }

    #[test]
    fn ticket_status_changed_protojson() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::TicketCreated, json!({"slug":"T-1","title":"Fix","status":"todo","priority":"high"})));
        dispatch(&mut s, &make_event(EventType::TicketStatusChanged, json!({"slug":"T-1","status":"in_progress","previous_status":"todo"})));
        assert_eq!(s.tickets.get_tickets()[0].status, "in_progress");
        // refetch queued for full ticket fields
        assert!(s.take_pending_refetch_ticket_slugs().contains(&"T-1".to_string()));
    }

    #[test]
    fn loop_run_completed_protojson() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::LoopRunStarted, json!({"id":5,"loop_slug":"l-1","status":"running"})));
        dispatch(&mut s, &make_event(EventType::LoopRunCompleted, json!({"run_id":"5","loop_id":"1","status":"completed"})));
        assert_eq!(s.loops.get_runs()[0].status, loop_run_status::COMPLETED);
    }

    #[test]
    fn loop_run_started() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::LoopRunStarted, json!({"id":1,"loop_slug":"l-1","status":"running"})));
        assert_eq!(s.loops.get_runs().len(), 1);
    }

    #[test]
    fn pod_status_changed_syncs_mesh_node() {
        use agentsmesh_types::proto_mesh_v1::{MeshNode, MeshTopology};
        let mut s = AppState::new();
        s.mesh.set_topology(MeshTopology {
            nodes: vec![MeshNode {
                pod_key: "p1".into(),
                status: "initializing".into(),
                agent_status: "idle".into(),
                ..Default::default()
            }],
            ..Default::default()
        });
        // status_changed patches the mesh node even when the pod isn't cached.
        dispatch(&mut s, &make_event(EventType::PodStatusChanged, json!({"pod_key":"p1","status":"running"})));
        assert_eq!(s.mesh.get_node_by_key("p1").unwrap().status, "running");
        // agent_status_changed (no status field) patches agent_status only,
        // keeping the node status intact.
        dispatch(&mut s, &make_event(EventType::PodAgentStatusChanged, json!({"pod_key":"p1","agent_status":"executing"})));
        let n = s.mesh.get_node_by_key("p1").unwrap();
        assert_eq!(n.agent_status, "executing");
        assert_eq!(n.status, "running");
        // terminated flips the mesh node status too.
        dispatch(&mut s, &make_event(EventType::PodTerminated, json!({"pod_key":"p1"})));
        assert_eq!(s.mesh.get_node_by_key("p1").unwrap().status, "terminated");
    }

    #[test]
    fn unknown_event_noop() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::Ping, json!({})));
        assert!(s.pods.pods().is_empty());
    }

    #[test]
    fn channel_member_added_increments_count() {
        let mut s = AppState::new();
        s.channels.add_channel(agentsmesh_types::proto_channel_state_v1::Channel {
            id: 7, name: "general".into(), is_archived: false, is_member: true,
            member_count: Some(1), ..Default::default()
        });
        dispatch(&mut s, &make_event(EventType::ChannelMemberAdded, json!({"channel_id": 7, "user_id": 42})));
        assert_eq!(s.channels.get_channel(7).unwrap().member_count, Some(2));
    }

    #[test]
    fn channel_member_removed_decrements_count_clamped() {
        let mut s = AppState::new();
        s.channels.add_channel(agentsmesh_types::proto_channel_state_v1::Channel {
            id: 7, name: "general".into(), is_archived: false, is_member: true,
            member_count: Some(0), ..Default::default()
        });
        dispatch(&mut s, &make_event(EventType::ChannelMemberRemoved, json!({"channel_id": 7, "user_id": 42})));
        assert_eq!(s.channels.get_channel(7).unwrap().member_count, Some(0), "must clamp at 0");
    }

    #[test]
    fn pod_perpetual_changed_toggles_field() {
        let mut s = AppState::new();
        s.pods.upsert_pod(Pod {
            pod_key: "p".into(), status: "running".into(),
            agent_slug: "claude".into(), perpetual: false, ..Default::default()
        }, Some(1));
        dispatch(&mut s, &make_event(EventType::PodPerpetualChanged, json!({"pod_key": "p", "perpetual": true})));
        assert!(s.pods.get_pod("p").unwrap().perpetual);
    }

    #[test]
    fn mr_event_queues_refetch_ticket() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::MrCreated, json!({"ticket_slug": "T-9", "pod_id": 0})));
        let drained = s.take_pending_refetch_ticket_slugs();
        assert_eq!(drained, vec!["T-9".to_string()]);
        // Second drain returns empty.
        assert!(s.take_pending_refetch_ticket_slugs().is_empty());
    }

    #[test]
    fn pipeline_event_resolves_pod_id_to_key() {
        let mut s = AppState::new();
        s.pods.upsert_pod(Pod {
            id: 42, pod_key: "pod-abc".into(), status: "running".into(),
            agent_slug: "claude".into(), ..Default::default()
        }, Some(1));
        dispatch(&mut s, &make_event(EventType::PipelineUpdated, json!({"ticket_slug": "T-1", "pod_id": 42})));
        assert_eq!(s.take_pending_refetch_pod_keys(), vec!["pod-abc".to_string()]);
        assert_eq!(s.take_pending_refetch_ticket_slugs(), vec!["T-1".to_string()]);
    }

    #[test]
    fn notification_event_queues_browser_notification() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::Notification, json!({
            "title": "New message",
            "body": "@you got mentioned",
            "link": "/channels/5"
        })));
        let notifs = s.take_pending_browser_notifications();
        assert_eq!(notifs.len(), 1);
        assert_eq!(notifs[0].title, "New message");
        assert_eq!(notifs[0].link.as_deref(), Some("/channels/5"));
    }

    #[test]
    fn loop_run_warning_queues_toast() {
        let mut s = AppState::new();
        dispatch(&mut s, &make_event(EventType::LoopRunWarning, json!({
            "run_number": 7,
            "detail": "step timeout after 5m"
        })));
        let toasts = s.take_pending_toasts();
        assert_eq!(toasts.len(), 1);
        assert_eq!(toasts[0].kind, "warning");
        assert_eq!(toasts[0].title_key, "loops.runWarningTitle");
        assert_eq!(toasts[0].description, "step timeout after 5m");
    }
}
