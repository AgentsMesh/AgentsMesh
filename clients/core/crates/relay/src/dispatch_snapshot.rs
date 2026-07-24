use serde::Deserialize;
use tracing::warn;

mod replay;

use replay::{apply_scope_reset, build_replay};

pub(crate) const ANSI_CLEAR: &[u8] = b"\x1b[2J\x1b[H";
pub(crate) const ANSI_CLEAR_SCROLLBACK: &[u8] = b"\x1b[3J";
pub(crate) const ANSI_RESET_SGR: &[u8] = b"\x1b[0m";
pub(crate) const ANSI_ALT_SCREEN_ENTER: &[u8] = b"\x1b[?1049h";
pub(crate) const ANSI_ALT_SCREEN_EXIT: &[u8] = b"\x1b[?1049l";
pub(crate) const ANSI_CANCEL_SEQUENCE: u8 = 0x18;
const SNAPSHOT_VERSION_V2: u16 = 2;

#[derive(Debug, PartialEq)]
pub struct SnapshotPayload {
    pub cols: u16,
    pub rows: u16,
    /// Recovery snapshots replace every subscriber after a delivery gap. Normal
    /// requester snapshots are replayed only to subscribers awaiting a baseline.
    pub reset_all: bool,
    /// Complete terminal replay. Dispatch only parses/builds this payload; the
    /// driver decides which pending subscribers are allowed to receive it.
    pub replay: Vec<u8>,
}

#[derive(Deserialize)]
struct SnapshotJson {
    snapshot_version: Option<u16>,
    serialized_active_content: Option<String>,
    serialized_content: Option<String>,
    serialized_normal_content: Option<String>,
    saved_cursor_replay: Option<String>,
    saved_normal_cursor_replay: Option<String>,
    preceding_join_replay: Option<String>,
    #[serde(default)]
    is_alt_screen: bool,
    #[serde(default)]
    alt_screen_mode: u16,
    reset_all: Option<bool>,
    #[serde(default)]
    parser_prefix: Vec<u8>,
    #[serde(default)]
    terminal_modes: Vec<String>,
    #[serde(default)]
    cols: u16,
    #[serde(default)]
    rows: u16,
}

pub(super) fn decode_snapshot(payload: &[u8]) -> Option<SnapshotPayload> {
    match serde_json::from_slice::<SnapshotJson>(payload) {
        Ok(snap) => snapshot_payload(snap),
        Err(e) => {
            warn!("failed to parse snapshot: {e}");
            None
        }
    }
}

fn snapshot_payload(snap: SnapshotJson) -> Option<SnapshotPayload> {
    let reset_all = snap.reset_all.unwrap_or(true);
    let replay = decode_replay(&snap)?;
    Some(SnapshotPayload {
        cols: snap.cols,
        rows: snap.rows,
        // Legacy runners broadcast snapshots globally and have no scope field.
        reset_all,
        replay: apply_scope_reset(replay, reset_all),
    })
}

fn decode_replay(snap: &SnapshotJson) -> Option<Vec<u8>> {
    match snap.snapshot_version {
        // V1 serialized_content is raw active-buffer content. Version 0 was
        // never published, but treating it as V1 is the conservative legacy
        // behavior for producers that emitted a default numeric field.
        None | Some(0 | 1) => Some(
            snap.serialized_content
                .as_deref()
                .map(|content| build_replay(snap, content, false))
                .unwrap_or_default(),
        ),
        Some(SNAPSHOT_VERSION_V2) => Some(
            snap.serialized_active_content
                .as_deref()
                .map(|content| build_replay(snap, content, true))
                // V2 keeps a self-contained replay in the legacy field so an
                // older or partially upgraded client can recover safely.
                .or_else(|| {
                    snap.serialized_content
                        .as_ref()
                        .map(|content| content.as_bytes().to_vec())
                })
                .unwrap_or_default(),
        ),
        Some(version) => match &snap.serialized_content {
            // Future structured fields are intentionally ignored. Only the
            // compatibility replay has semantics this decoder understands.
            Some(content) => Some(content.as_bytes().to_vec()),
            None => {
                warn!(
                    "unsupported terminal snapshot version {version} without compatibility replay"
                );
                None
            }
        },
    }
}
