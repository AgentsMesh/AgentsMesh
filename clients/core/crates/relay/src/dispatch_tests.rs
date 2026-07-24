use std::sync::{Arc, Mutex};

use agentsmesh_protocol::MsgType;

use crate::dispatch::{dispatch_message, DispatchAction};
use crate::types::OutputCallback;

fn make_callback() -> (OutputCallback, Arc<Mutex<Vec<Vec<u8>>>>) {
    let received = Arc::new(Mutex::new(Vec::new()));
    let r = received.clone();
    let cb: OutputCallback = Arc::new(move |data| r.lock().unwrap().push(data));
    (cb, received)
}

#[test]
fn output_broadcasts_to_all() {
    let (cb1, r1) = make_callback();
    let (cb2, r2) = make_callback();
    let subs = vec![&cb1, &cb2];

    let action = dispatch_message(MsgType::Output, b"hello", &subs);
    assert_eq!(action, DispatchAction::None);
    assert_eq!(r1.lock().unwrap().len(), 1);
    assert_eq!(r2.lock().unwrap().len(), 1);
    assert_eq!(r1.lock().unwrap()[0], b"hello");
}

#[test]
fn control_pod_resized() {
    let json = serde_json::json!({"type": "pod_resized", "cols": 120, "rows": 40});
    let payload = serde_json::to_vec(&json).unwrap();
    let action = dispatch_message(MsgType::Control, &payload, &[]);
    assert_eq!(
        action,
        DispatchAction::PodResized {
            cols: 120,
            rows: 40
        }
    );
}

#[test]
fn control_unknown_type() {
    let json = serde_json::json!({"type": "something_else"});
    let payload = serde_json::to_vec(&json).unwrap();
    let action = dispatch_message(MsgType::Control, &payload, &[]);
    assert_eq!(action, DispatchAction::None);
}

#[test]
fn malformed_control_and_unhandled_client_message_are_noops() {
    assert_eq!(
        dispatch_message(MsgType::Control, b"not json", &[]),
        DispatchAction::None
    );
    assert_eq!(
        dispatch_message(MsgType::Input, b"client-only", &[]),
        DispatchAction::None
    );
}

#[test]
fn runner_disconnected_broadcasts() {
    let (cb, received) = make_callback();
    let action = dispatch_message(MsgType::RunnerDisconnected, &[], &[&cb]);
    assert_eq!(action, DispatchAction::RunnerDisconnected);
    assert_eq!(received.lock().unwrap().len(), 1);
}

#[test]
fn runner_reconnected_broadcasts() {
    let (cb, received) = make_callback();
    let action = dispatch_message(MsgType::RunnerReconnected, &[], &[&cb]);
    assert_eq!(action, DispatchAction::RunnerReconnected);
    assert_eq!(received.lock().unwrap().len(), 1);
}

#[test]
fn acp_event_parsed() {
    let json = serde_json::json!({"event": "test"});
    let payload = serde_json::to_vec(&json).unwrap();
    let action = dispatch_message(MsgType::AcpEvent, &payload, &[]);
    match action {
        DispatchAction::AcpMessage { msg_type, payload } => {
            assert_eq!(msg_type, MsgType::AcpEvent);
            assert_eq!(payload["event"], "test");
        }
        _ => panic!("expected AcpMessage"),
    }
}

#[test]
fn acp_invalid_json() {
    let action = dispatch_message(MsgType::AcpEvent, b"not json", &[]);
    assert_eq!(action, DispatchAction::None);
}

#[test]
fn pong_is_noop() {
    let action = dispatch_message(MsgType::Pong, &[], &[]);
    assert_eq!(action, DispatchAction::None);
}

#[test]
fn invalid_snapshot_json() {
    let action = dispatch_message(MsgType::Snapshot, b"bad json", &[]);
    assert_eq!(action, DispatchAction::None);
}
