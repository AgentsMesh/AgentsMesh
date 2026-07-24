use agentsmesh_protocol::MsgType;
use serde::Deserialize;
use tracing::warn;

use crate::dispatch_snapshot::decode_snapshot;
pub use crate::dispatch_snapshot::SnapshotPayload;
use crate::types::OutputCallback;

#[derive(Debug, PartialEq)]
pub enum DispatchAction {
    None,
    Snapshot(SnapshotPayload),
    PodResized {
        cols: u16,
        rows: u16,
    },
    RunnerDisconnected,
    RunnerReconnected,
    AcpMessage {
        msg_type: MsgType,
        payload: serde_json::Value,
    },
}

#[derive(Deserialize)]
struct ControlJson {
    #[serde(rename = "type")]
    msg_type: Option<String>,
    #[serde(default)]
    cols: u16,
    #[serde(default)]
    rows: u16,
}

pub fn dispatch_message(
    msg_type: MsgType,
    payload: &[u8],
    subscribers: &[&OutputCallback],
) -> DispatchAction {
    match msg_type {
        MsgType::Output => {
            broadcast(subscribers, payload);
            DispatchAction::None
        }
        MsgType::Snapshot => decode_snapshot(payload)
            .map(DispatchAction::Snapshot)
            .unwrap_or(DispatchAction::None),
        MsgType::Control => handle_control(payload),
        MsgType::RunnerDisconnected => {
            let msg = b"\r\n\x1b[33mRunner disconnected. Waiting for reconnection...\x1b[0m\r\n";
            broadcast(subscribers, msg);
            DispatchAction::RunnerDisconnected
        }
        MsgType::RunnerReconnected => {
            let msg = b"\r\n\x1b[32mRunner reconnected.\x1b[0m\r\n";
            broadcast(subscribers, msg);
            DispatchAction::RunnerReconnected
        }
        MsgType::AcpEvent | MsgType::AcpSnapshot | MsgType::AcpCommand => {
            match serde_json::from_slice::<serde_json::Value>(payload) {
                Ok(val) => DispatchAction::AcpMessage {
                    msg_type,
                    payload: val,
                },
                Err(e) => {
                    warn!("failed to parse ACP payload: {e}");
                    DispatchAction::None
                }
            }
        }
        MsgType::Pong | MsgType::Ping => DispatchAction::None,
        _ => {
            warn!("unhandled relay message type: {msg_type:?}");
            DispatchAction::None
        }
    }
}

fn handle_control(payload: &[u8]) -> DispatchAction {
    match serde_json::from_slice::<ControlJson>(payload) {
        Ok(ctrl) if ctrl.msg_type.as_deref() == Some("pod_resized") => DispatchAction::PodResized {
            cols: ctrl.cols,
            rows: ctrl.rows,
        },
        Ok(_) => DispatchAction::None,
        Err(e) => {
            warn!("failed to parse control message: {e}");
            DispatchAction::None
        }
    }
}

fn broadcast(subscribers: &[&OutputCallback], data: &[u8]) {
    // Each callback takes an owned Vec, so N copies are unavoidable — but skip the
    // extra up-front `owned` clone (that made it N+1). Copy straight per callback.
    for cb in subscribers {
        cb(data.to_vec());
    }
}
