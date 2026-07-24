use std::sync::{Arc, Mutex};

use agentsmesh_protocol::MsgType;

use crate::dispatch::{dispatch_message, DispatchAction, SnapshotPayload};
use crate::dispatch_snapshot::{
    ANSI_ALT_SCREEN_ENTER, ANSI_ALT_SCREEN_EXIT, ANSI_CANCEL_SEQUENCE, ANSI_CLEAR,
    ANSI_CLEAR_SCROLLBACK, ANSI_RESET_SGR,
};
use crate::types::OutputCallback;

const AUTOWRAP_ENABLE: &[u8] = b"\x1b[?7h";

fn make_callback() -> (OutputCallback, Arc<Mutex<Vec<Vec<u8>>>>) {
    let received = Arc::new(Mutex::new(Vec::new()));
    let r = received.clone();
    let cb: OutputCallback = Arc::new(move |data| r.lock().unwrap().push(data));
    (cb, received)
}

#[test]
fn snapshot_restores_normal_buffer_before_clear_and_content() {
    let (cb, received) = make_callback();
    let json = serde_json::json!({
        "serialized_content": "screen data",
        "is_alt_screen": false,
        "reset_all": false,
        "cols": 80,
        "rows": 24
    });
    let payload = serde_json::to_vec(&json).unwrap();

    let action = dispatch_message(MsgType::Snapshot, &payload, &[&cb]);
    let mut expected = vec![ANSI_CANCEL_SEQUENCE];
    append_buffer(&mut expected, ANSI_ALT_SCREEN_EXIT, b"screen data");
    assert_eq!(
        action,
        DispatchAction::Snapshot(SnapshotPayload {
            cols: 80,
            rows: 24,
            reset_all: false,
            replay: expected,
        })
    );
    assert!(received.lock().unwrap().is_empty());
}

#[test]
fn snapshot_legacy_alt_payload_restores_active_buffer() {
    let (cb, received) = make_callback();
    let json = serde_json::json!({
        "serialized_content": "tui data",
        "is_alt_screen": true,
        "reset_all": false,
        "cols": 120,
        "rows": 40
    });
    let payload = serde_json::to_vec(&json).unwrap();

    let action = dispatch_message(MsgType::Snapshot, &payload, &[&cb]);
    let mut expected = vec![ANSI_CANCEL_SEQUENCE];
    append_buffer(&mut expected, ANSI_ALT_SCREEN_ENTER, b"tui data");
    assert_eq!(
        action,
        DispatchAction::Snapshot(SnapshotPayload {
            cols: 120,
            rows: 40,
            reset_all: false,
            replay: expected,
        })
    );
    assert!(received.lock().unwrap().is_empty());
}

#[test]
fn snapshot_restores_hidden_normal_buffer_before_alt_buffer() {
    let (cb, received) = make_callback();
    let json = serde_json::json!({
        "snapshot_version": 2,
        "serialized_active_content": "active tui",
        "serialized_content": "COMPATIBILITY_REPLAY",
        "serialized_normal_content": "hidden shell",
        "is_alt_screen": true,
        "reset_all": false,
        "cols": 80,
        "rows": 24
    });
    let payload = serde_json::to_vec(&json).unwrap();

    let action = dispatch_message(MsgType::Snapshot, &payload, &[&cb]);
    let mut expected = vec![ANSI_CANCEL_SEQUENCE];
    append_full_buffer(&mut expected, ANSI_ALT_SCREEN_EXIT, b"hidden shell");
    append_buffer(&mut expected, ANSI_ALT_SCREEN_ENTER, b"active tui");
    assert_eq!(
        action,
        DispatchAction::Snapshot(SnapshotPayload {
            cols: 80,
            rows: 24,
            reset_all: false,
            replay: expected,
        })
    );
    assert!(received.lock().unwrap().is_empty());
}

#[test]
fn snapshot_no_content() {
    let (cb, received) = make_callback();
    let json = serde_json::json!({"reset_all": false, "cols": 120, "rows": 40});
    let payload = serde_json::to_vec(&json).unwrap();

    let action = dispatch_message(MsgType::Snapshot, &payload, &[&cb]);
    assert_eq!(
        action,
        DispatchAction::Snapshot(SnapshotPayload {
            cols: 120,
            rows: 40,
            reset_all: false,
            replay: Vec::new(),
        })
    );
    assert!(received.lock().unwrap().is_empty());
}

#[test]
fn snapshot_empty_content_still_clears_stale_screen() {
    let (cb, received) = make_callback();
    let json = serde_json::json!({
        "serialized_content": "",
        "is_alt_screen": false,
        "reset_all": false,
        "cols": 120,
        "rows": 40
    });
    let payload = serde_json::to_vec(&json).unwrap();

    let action = dispatch_message(MsgType::Snapshot, &payload, &[&cb]);
    let mut expected = vec![ANSI_CANCEL_SEQUENCE];
    append_buffer(&mut expected, ANSI_ALT_SCREEN_EXIT, b"");
    assert_eq!(
        action,
        DispatchAction::Snapshot(SnapshotPayload {
            cols: 120,
            rows: 40,
            reset_all: false,
            replay: expected,
        })
    );
    assert!(received.lock().unwrap().is_empty());
}

#[test]
fn snapshot_legacy_versions_wrap_raw_content_with_canonical_autowrap() {
    for snapshot_version in [None, Some(1)] {
        let mut json = serde_json::json!({
            "serialized_content": "right-margin-content",
            "reset_all": false,
            "cols": 80,
            "rows": 24
        });
        if let Some(version) = snapshot_version {
            json["snapshot_version"] = serde_json::json!(version);
        }

        let replay = snapshot_replay(json);
        let mut expected = vec![ANSI_CANCEL_SEQUENCE];
        append_buffer(&mut expected, ANSI_ALT_SCREEN_EXIT, b"right-margin-content");
        assert_eq!(replay, expected, "version {snapshot_version:?}");
    }
}

#[test]
fn normal_snapshot_replays_content_modes_register_then_parser_prefix() {
    let replay = snapshot_replay(serde_json::json!({
        "snapshot_version": 2,
        "serialized_active_content": "NORMAL",
        "serialized_content": "LEGACY",
        "saved_cursor_replay": "REGISTER",
        "terminal_modes": ["\u{001b}[?7l", "\u{001b}[?25l"],
        "parser_prefix": [27, 91, 51, 49],
        "is_alt_screen": false,
        "reset_all": false
    }));

    let mut expected = vec![ANSI_CANCEL_SEQUENCE];
    append_full_buffer(&mut expected, ANSI_ALT_SCREEN_EXIT, b"NORMAL");
    expected.extend_from_slice(b"\x1b[?7l\x1b[?25l");
    expected.extend_from_slice(b"REGISTER\x1b[31");
    assert_eq!(replay, expected);
}

#[test]
fn alternate_snapshot_replays_each_register_under_final_modes() {
    for (mode, enter) in [
        (47, b"\x1b[?47h".as_slice()),
        (1047, b"\x1b[?1047h".as_slice()),
        (1049, ANSI_ALT_SCREEN_ENTER),
    ] {
        let replay = snapshot_replay(serde_json::json!({
            "snapshot_version": 2,
            "serialized_active_content": "ALT",
            "serialized_content": "COMPATIBILITY_REPLAY",
            "serialized_normal_content": "NORMAL",
            "saved_normal_cursor_replay": "NORMAL_REGISTER",
            "saved_cursor_replay": "ALT_REGISTER",
            "terminal_modes": ["\u{001b}[?7l"],
            "parser_prefix": [27],
            "is_alt_screen": true,
            "alt_screen_mode": mode,
            "reset_all": false
        }));

        let mut expected = vec![ANSI_CANCEL_SEQUENCE];
        append_full_buffer(&mut expected, ANSI_ALT_SCREEN_EXIT, b"NORMAL");
        expected.extend_from_slice(b"\x1b[?7lNORMAL_REGISTER");
        append_buffer(&mut expected, enter, b"ALT");
        expected.extend_from_slice(b"\x1b[?7lALT_REGISTER\x1b");
        assert_eq!(replay, expected, "alternate mode {mode}");
    }
}

#[test]
fn snapshot_replay_restores_modes_then_pending_parser_prefix() {
    let replay = snapshot_replay(serde_json::json!({
        "snapshot_version": 2,
        "serialized_active_content": "screen",
        "serialized_content": "COMPATIBILITY_REPLAY",
        "terminal_modes": ["\u{001b}[?25l", "\u{001b}[?2004h", "\u{001b}[?1000l"],
        "parser_prefix": [27, 91, 51, 49],
        "reset_all": false,
        "cols": 80,
        "rows": 24
    }));

    assert_eq!(replay[0], ANSI_CANCEL_SEQUENCE);
    assert!(replay.ends_with(b"\x1b[?25l\x1b[?2004h\x1b[?1000l\x1b[31"));
}

#[test]
fn preceding_join_replay_runs_after_modes_and_cursor_registers() {
    let replay = snapshot_replay(serde_json::json!({
        "snapshot_version": 2,
        "serialized_active_content": "NORMAL",
        "serialized_content": "COMPATIBILITY_REPLAY",
        "saved_cursor_replay": "REGISTER",
        "preceding_join_replay": "JOIN",
        "terminal_modes": ["\u{001b}[?7l"],
        "is_alt_screen": false,
        "reset_all": false
    }));

    assert!(replay.ends_with(b"\x1b[?7lREGISTERJOIN"));
}

fn snapshot_replay(value: serde_json::Value) -> Vec<u8> {
    let payload = serde_json::to_vec(&value).unwrap();
    let DispatchAction::Snapshot(snapshot) = dispatch_message(MsgType::Snapshot, &payload, &[])
    else {
        panic!("expected snapshot action");
    };
    snapshot.replay
}

fn append_buffer(expected: &mut Vec<u8>, screen_mode: &[u8], content: &[u8]) {
    expected.extend_from_slice(screen_mode);
    expected.extend_from_slice(ANSI_RESET_SGR);
    expected.extend_from_slice(ANSI_CLEAR);
    expected.extend_from_slice(AUTOWRAP_ENABLE);
    expected.extend_from_slice(content);
}

fn append_full_buffer(expected: &mut Vec<u8>, screen_mode: &[u8], content: &[u8]) {
    expected.extend_from_slice(screen_mode);
    expected.extend_from_slice(ANSI_CLEAR_SCROLLBACK);
    expected.extend_from_slice(ANSI_RESET_SGR);
    expected.extend_from_slice(ANSI_CLEAR);
    expected.extend_from_slice(AUTOWRAP_ENABLE);
    expected.extend_from_slice(content);
}
