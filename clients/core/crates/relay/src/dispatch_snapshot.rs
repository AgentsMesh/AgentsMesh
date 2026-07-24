use serde::Deserialize;
use tracing::warn;

pub(crate) const ANSI_CLEAR: &[u8] = b"\x1b[2J\x1b[H";
pub(crate) const ANSI_CLEAR_SCROLLBACK: &[u8] = b"\x1b[3J";
pub(crate) const ANSI_RESET_SGR: &[u8] = b"\x1b[0m";
pub(crate) const ANSI_ALT_SCREEN_ENTER: &[u8] = b"\x1b[?1049h";
pub(crate) const ANSI_ALT_SCREEN_EXIT: &[u8] = b"\x1b[?1049l";
pub(crate) const ANSI_CANCEL_SEQUENCE: u8 = 0x18;
const ANSI_ALT_SCREEN_47_ENTER: &[u8] = b"\x1b[?47h";
const ANSI_ALT_SCREEN_1047_ENTER: &[u8] = b"\x1b[?1047h";
const ANSI_AUTOWRAP_ENABLE: &[u8] = b"\x1b[?7h";
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

fn apply_scope_reset(replay: Vec<u8>, reset_all: bool) -> Vec<u8> {
    if !reset_all {
        return replay;
    }

    // CSI 3 J only erases the active xterm buffer. Canonicalize to the normal
    // buffer first so a client currently showing an alternate screen cannot
    // retain stale normal-buffer scrollback across a global recovery.
    let replay = replay
        .strip_prefix(&[ANSI_CANCEL_SEQUENCE])
        .unwrap_or(&replay);
    let replay = replay.strip_prefix(ANSI_ALT_SCREEN_EXIT).unwrap_or(replay);
    let mut scoped = Vec::with_capacity(
        1 + ANSI_ALT_SCREEN_EXIT.len() + ANSI_CLEAR_SCROLLBACK.len() + replay.len(),
    );
    scoped.push(ANSI_CANCEL_SEQUENCE);
    scoped.extend_from_slice(ANSI_ALT_SCREEN_EXIT);
    scoped.extend_from_slice(ANSI_CLEAR_SCROLLBACK);
    scoped.extend_from_slice(replay);
    scoped
}

fn build_replay(snap: &SnapshotJson, content: &str, full_history: bool) -> Vec<u8> {
    let normal_len = snap
        .serialized_normal_content
        .as_ref()
        .map_or(0, String::len);
    let mode_len: usize = snap.terminal_modes.iter().map(String::len).sum();
    let cursor_len = snap.saved_cursor_replay.as_ref().map_or(0, String::len)
        + snap
            .saved_normal_cursor_replay
            .as_ref()
            .map_or(0, String::len);
    let join_len = snap.preceding_join_replay.as_ref().map_or(0, String::len);
    let mode_replays = if snap.is_alt_screen && snap.serialized_normal_content.is_some() {
        2
    } else {
        1
    };
    let mut replay = Vec::with_capacity(
        content.len()
            + normal_len
            + mode_len * mode_replays
            + cursor_len
            + join_len
            + snap.parser_prefix.len()
            + 65,
    );
    replay.push(ANSI_CANCEL_SEQUENCE);
    // Rebuild content with autowrap enabled, then apply final modes before each
    // buffer-local saved register. Right-margin register replay depends on DECAWM.
    if snap.is_alt_screen {
        if let Some(normal) = &snap.serialized_normal_content {
            append_snapshot_buffer(
                &mut replay,
                ANSI_ALT_SCREEN_EXIT,
                normal.as_bytes(),
                full_history,
            );
            append_modes(&mut replay, &snap.terminal_modes);
            append_optional(&mut replay, snap.saved_normal_cursor_replay.as_deref());
        }
        append_snapshot_buffer(
            &mut replay,
            alt_screen_enter(snap.alt_screen_mode),
            content.as_bytes(),
            false,
        );
        append_modes(&mut replay, &snap.terminal_modes);
    } else {
        append_snapshot_buffer(
            &mut replay,
            ANSI_ALT_SCREEN_EXIT,
            content.as_bytes(),
            full_history,
        );
        append_modes(&mut replay, &snap.terminal_modes);
    }
    append_optional(&mut replay, snap.saved_cursor_replay.as_deref());
    append_optional(&mut replay, snap.preceding_join_replay.as_deref());
    // Subsequent raw bytes continue the Runner parser's exact cut.
    replay.extend_from_slice(&snap.parser_prefix);
    replay
}

fn alt_screen_enter(mode: u16) -> &'static [u8] {
    match mode {
        47 => ANSI_ALT_SCREEN_47_ENTER,
        1047 => ANSI_ALT_SCREEN_1047_ENTER,
        _ => ANSI_ALT_SCREEN_ENTER,
    }
}

fn append_snapshot_buffer(
    replay: &mut Vec<u8>,
    screen_mode: &[u8],
    content: &[u8],
    clear_scrollback: bool,
) {
    replay.extend_from_slice(screen_mode);
    if clear_scrollback {
        replay.extend_from_slice(ANSI_CLEAR_SCROLLBACK);
    }
    replay.extend_from_slice(ANSI_RESET_SGR);
    replay.extend_from_slice(ANSI_CLEAR);
    replay.extend_from_slice(ANSI_AUTOWRAP_ENABLE);
    replay.extend_from_slice(content);
}

fn append_modes(replay: &mut Vec<u8>, modes: &[String]) {
    for mode in modes {
        replay.extend_from_slice(mode.as_bytes());
    }
}

fn append_optional(replay: &mut Vec<u8>, value: Option<&str>) {
    if let Some(value) = value {
        replay.extend_from_slice(value.as_bytes());
    }
}
