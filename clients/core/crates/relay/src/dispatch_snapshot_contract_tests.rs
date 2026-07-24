use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};

use agentsmesh_protocol::MsgType;

use crate::dispatch::{dispatch_message, DispatchAction, SnapshotPayload};
use crate::dispatch_snapshot::{
    ANSI_ALT_SCREEN_EXIT, ANSI_CANCEL_SEQUENCE, ANSI_CLEAR_SCROLLBACK, ANSI_RESET_SGR,
};
use crate::types::OutputCallback;

const SNAPSHOT_FIXTURE_DIR: &str = "proto/testdata/terminal_snapshot";
const ANSI_AUTOWRAP_ENABLE: &[u8] = b"\x1b[?7h";

fn make_callback() -> (OutputCallback, Arc<Mutex<Vec<Vec<u8>>>>) {
    let received = Arc::new(Mutex::new(Vec::new()));
    let r = received.clone();
    let cb: OutputCallback = Arc::new(move |data| r.lock().unwrap().push(data));
    (cb, received)
}

#[test]
fn legacy_screen_only_snapshot_does_not_erase_existing_scrollback() {
    let (cb, received) = make_callback();
    let payload = serde_json::to_vec(&serde_json::json!({
        "serialized_content": "current viewport",
        "is_alt_screen": false,
        "reset_all": false,
        "cols": 80,
        "rows": 24
    }))
    .unwrap();

    let action = dispatch_message(MsgType::Snapshot, &payload, &[&cb]);
    let DispatchAction::Snapshot(snapshot) = action else {
        panic!("expected snapshot action");
    };
    let replay = &snapshot.replay;
    assert_eq!(replay.first(), Some(&ANSI_CANCEL_SEQUENCE));
    assert!(replay[1..].starts_with(ANSI_ALT_SCREEN_EXIT));
    assert!(replay
        .windows(ANSI_RESET_SGR.len())
        .any(|window| window == ANSI_RESET_SGR));
    // CSI 3 J is xterm's scrollback erase command. Replacing the viewport must
    // not discard lines the user can still scroll back to.
    assert!(!replay
        .windows(b"\x1b[3J".len())
        .any(|window| window == b"\x1b[3J"));
    assert!(received.lock().unwrap().is_empty());
}

#[test]
fn snapshot_v2_full_history_replaces_target_scrollback() {
    let snapshot = snapshot_from_json(serde_json::json!({
        "snapshot_version": 2,
        "serialized_active_content": "full history and viewport",
        "serialized_content": "COMPATIBILITY_REPLAY",
        "is_alt_screen": false,
        "reset_all": false,
        "cols": 80,
        "rows": 24
    }));

    let expected_prefix = [
        &[ANSI_CANCEL_SEQUENCE][..],
        ANSI_ALT_SCREEN_EXIT,
        ANSI_CLEAR_SCROLLBACK,
    ]
    .concat();
    assert!(snapshot.replay.starts_with(&expected_prefix));
    assert!(snapshot.replay.ends_with(b"full history and viewport"));
}

#[test]
fn snapshot_recovery_returns_to_normal_before_erasing_scrollback() {
    let snapshot = snapshot_from_json(serde_json::json!({
        "snapshot_version": 2,
        "serialized_active_content": "recovery viewport",
        "serialized_content": "COMPATIBILITY_REPLAY",
        "reset_all": true,
        "cols": 80,
        "rows": 24
    }));

    let expected_prefix = [
        &[ANSI_CANCEL_SEQUENCE][..],
        ANSI_ALT_SCREEN_EXIT,
        ANSI_CLEAR_SCROLLBACK,
    ]
    .concat();
    assert!(snapshot.replay.starts_with(&expected_prefix));
}

#[test]
fn snapshot_v2_prefers_structured_active_content() {
    let snapshot = snapshot_from_json(serde_json::json!({
        "snapshot_version": 2,
        "serialized_active_content": "ACTIVE",
        "serialized_content": "COMPATIBILITY_REPLAY",
        "reset_all": false
    }));

    assert!(snapshot.replay.ends_with(b"ACTIVE"));
    assert!(!contains_bytes(&snapshot.replay, b"COMPATIBILITY_REPLAY"));
    assert!(contains_bytes(&snapshot.replay, ANSI_AUTOWRAP_ENABLE));
}

#[test]
fn snapshot_v2_without_active_content_uses_compatibility_replay_verbatim() {
    let compatibility_replay = "\u{0018}\u{001b}[?1049lCOMPATIBILITY_REPLAY";
    let snapshot = snapshot_from_json(serde_json::json!({
        "snapshot_version": 2,
        "serialized_content": compatibility_replay,
        "reset_all": false
    }));

    assert_eq!(snapshot.replay, compatibility_replay.as_bytes());
}

#[test]
fn snapshot_future_version_uses_only_compatibility_replay() {
    let compatibility_replay = "\u{0018}FUTURE_COMPATIBILITY_REPLAY";
    let snapshot = snapshot_from_json(serde_json::json!({
        "snapshot_version": 37,
        "serialized_active_content": "UNKNOWN_STRUCTURED_CONTENT",
        "serialized_content": compatibility_replay,
        "reset_all": false
    }));

    assert_eq!(snapshot.replay, compatibility_replay.as_bytes());
}

#[test]
fn snapshot_future_version_without_compatibility_replay_is_rejected() {
    let payload = serde_json::to_vec(&serde_json::json!({
        "snapshot_version": 37,
        "serialized_active_content": "UNKNOWN_STRUCTURED_CONTENT",
        "reset_all": false
    }))
    .unwrap();

    assert_eq!(
        dispatch_message(MsgType::Snapshot, &payload, &[]),
        DispatchAction::None
    );
}

#[test]
fn snapshot_global_compatibility_replay_canonicalizes_before_scrollback_clear() {
    for compatibility_replay in ["\u{0018}SELF_CONTAINED", "SELF_CONTAINED"] {
        let snapshot = snapshot_from_json(serde_json::json!({
            "snapshot_version": 2,
            "serialized_content": compatibility_replay,
            "reset_all": true
        }));

        let expected = [
            &[ANSI_CANCEL_SEQUENCE][..],
            ANSI_ALT_SCREEN_EXIT,
            ANSI_CLEAR_SCROLLBACK,
            b"SELF_CONTAINED",
        ]
        .concat();
        assert_eq!(snapshot.replay, expected);
    }
}

#[test]
fn snapshot_without_scope_keeps_legacy_global_semantics() {
    let payload = serde_json::to_vec(&serde_json::json!({
        "serialized_content": "legacy viewport",
        "cols": 80,
        "rows": 24
    }))
    .unwrap();

    let DispatchAction::Snapshot(snapshot) = dispatch_message(MsgType::Snapshot, &payload, &[])
    else {
        panic!("expected snapshot action");
    };
    assert!(snapshot.reset_all);
    assert_eq!(snapshot.replay[0], ANSI_CANCEL_SEQUENCE);
    assert_eq!(
        &snapshot.replay[1..1 + ANSI_ALT_SCREEN_EXIT.len()],
        ANSI_ALT_SCREEN_EXIT
    );
    assert_eq!(
        &snapshot.replay[1 + ANSI_ALT_SCREEN_EXIT.len()
            ..1 + ANSI_ALT_SCREEN_EXIT.len() + ANSI_CLEAR_SCROLLBACK.len()],
        ANSI_CLEAR_SCROLLBACK,
    );
}

#[test]
fn shared_snapshot_fixtures_lock_scope_and_terminal_state_contract() {
    let requester = snapshot_fixture(&read_snapshot_fixture("current_requester"));
    assert!(!requester.reset_all);
    assert!(contains_bytes(&requester.replay, ANSI_CLEAR_SCROLLBACK));
    assert!(requester
        .replay
        .windows(16)
        .any(|w| w == b"requester-screen"));

    let recovery = snapshot_fixture(&read_snapshot_fixture("current_recovery"));
    assert!(recovery.reset_all);
    assert!(contains_bytes(&recovery.replay, ANSI_CLEAR_SCROLLBACK));
    assert!(recovery.replay.windows(15).any(|w| w == b"recovery-screen"));

    let legacy = snapshot_fixture(&read_snapshot_fixture("legacy_missing_scope"));
    assert!(legacy.reset_all);
    assert!(contains_bytes(&legacy.replay, ANSI_CLEAR_SCROLLBACK));

    let alt = snapshot_fixture(&read_snapshot_fixture("alt_terminal_state"));
    assert!(!alt.reset_all);
    assert_eq!(alt.replay.first(), Some(&ANSI_CANCEL_SEQUENCE));
    assert!(alt.replay.windows(13).any(|w| w == b"normal-screen"));
    assert!(alt.replay.windows(10).any(|w| w == b"alt-screen"));
    assert!(alt.replay.windows(8).any(|w| w == b"\x1b[?2004h"));
    assert!(alt.replay.ends_with(b"\x1b[31"));
}

fn read_snapshot_fixture(name: &str) -> Vec<u8> {
    let relative = Path::new(SNAPSHOT_FIXTURE_DIR).join(format!("{name}.json"));
    if let Some(srcdir) = std::env::var_os("TEST_SRCDIR") {
        let workspace = std::env::var_os("TEST_WORKSPACE").unwrap_or_else(|| "_main".into());
        return std::fs::read(PathBuf::from(srcdir).join(workspace).join(relative)).unwrap();
    }
    if let Some(root) = find_workspace_root() {
        return std::fs::read(root.join(relative)).unwrap();
    }
    panic!("could not locate terminal snapshot fixtures");
}

fn find_workspace_root() -> Option<PathBuf> {
    let mut directory = std::env::current_dir().ok()?;
    loop {
        if directory.join("MODULE.bazel").is_file() {
            return Some(directory);
        }
        if !directory.pop() {
            return None;
        }
    }
}

fn snapshot_fixture(payload: &[u8]) -> SnapshotPayload {
    let DispatchAction::Snapshot(snapshot) = dispatch_message(MsgType::Snapshot, payload, &[])
    else {
        panic!("shared fixture did not parse as a snapshot");
    };
    snapshot
}

fn snapshot_from_json(value: serde_json::Value) -> SnapshotPayload {
    snapshot_fixture(&serde_json::to_vec(&value).unwrap())
}

fn contains_bytes(haystack: &[u8], needle: &[u8]) -> bool {
    haystack
        .windows(needle.len())
        .any(|window| window == needle)
}
