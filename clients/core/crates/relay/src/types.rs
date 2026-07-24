use std::sync::Arc;

use agentsmesh_protocol::MsgType;
use futures::channel::mpsc;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RelayStatus {
    Connecting,
    Connected,
    Disconnected,
    Error,
}

impl std::fmt::Display for RelayStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Connecting => write!(f, "connecting"),
            Self::Connected => write!(f, "connected"),
            Self::Disconnected => write!(f, "disconnected"),
            Self::Error => write!(f, "error"),
        }
    }
}

pub type OutputCallback = Arc<dyn Fn(Vec<u8>) + Send + Sync>;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RelayStatusInfo {
    pub status: RelayStatus,
    pub runner_disconnected: bool,
    pub revision: u64,
}

/// Driver-owned, pool-readable status mirror. The driver task is the single
/// writer (under its own lock); the pool's `get_status` / `is_runner_disconnected`
/// / `get_pod_size` read it directly instead of round-tripping a command.
#[derive(Debug, Clone)]
pub(crate) struct StatusSnapshot {
    pub status: RelayStatus,
    pub runner_disconnected: bool,
    pub pod_size: Option<(u16, u16)>,
    pub revision: u64,
}

impl Default for StatusSnapshot {
    fn default() -> Self {
        Self {
            status: RelayStatus::Disconnected,
            runner_disconnected: false,
            pod_size: None,
            revision: 0,
        }
    }
}

pub type StatusCallback = Arc<dyn Fn(RelayStatusInfo) + Send + Sync>;
pub type AcpCallback = Arc<dyn Fn(MsgType, serde_json::Value) + Send + Sync>;
pub type GenerationStatusCallback = Arc<dyn Fn(u32, RelayStatusInfo) + Send + Sync>;
pub type GenerationAcpCallback = Arc<dyn Fn(u32, MsgType, serde_json::Value) + Send + Sync>;
// Fired once when a pod connection is fully torn down (disconnect_inner) so
// adapters can drop their register-once guard and re-register listeners on the
// next subscribe. Carries the pod_key.
pub type DisconnectCallback = Arc<dyn Fn(String) + Send + Sync>;
/// Desktop-only lifecycle notification carrying the exact driver generation
/// that was retired. Other adapters keep the pod-scoped callback above for
/// backward compatibility.
pub type GenerationDisconnectCallback = Arc<dyn Fn(String, u32) + Send + Sync>;

pub struct ConnectionHandle {
    pub pod_key: String,
    pub subscription_id: String,
    cmd_tx: mpsc::UnboundedSender<crate::command::Command>,
    unsubscribe_tx: mpsc::UnboundedSender<(String, String)>,
}

impl ConnectionHandle {
    pub(crate) fn new(
        pod_key: String,
        subscription_id: String,
        cmd_tx: mpsc::UnboundedSender<crate::command::Command>,
        unsubscribe_tx: mpsc::UnboundedSender<(String, String)>,
    ) -> Self {
        Self {
            pod_key,
            subscription_id,
            cmd_tx,
            unsubscribe_tx,
        }
    }

    pub fn send(&self, data: Vec<u8>) {
        let _ = self.cmd_tx.unbounded_send(crate::command::Command::Send {
            data: String::from_utf8_lossy(&data).into_owned(),
        });
    }

    pub fn unsubscribe(&self) {
        let _ = self
            .unsubscribe_tx
            .unbounded_send((self.pod_key.clone(), self.subscription_id.clone()));
    }
}

// LCOV_EXCL_START: test-only code
#[cfg(test)]
mod tests {
    use super::*;
    use crate::command::Command;

    #[test]
    fn relay_status_strings_are_stable() {
        assert_eq!(RelayStatus::Connecting.to_string(), "connecting");
        assert_eq!(RelayStatus::Connected.to_string(), "connected");
        assert_eq!(RelayStatus::Disconnected.to_string(), "disconnected");
        assert_eq!(RelayStatus::Error.to_string(), "error");
    }

    #[test]
    fn status_snapshot_default_is_fully_disconnected() {
        let snapshot = StatusSnapshot::default();
        assert_eq!(snapshot.status, RelayStatus::Disconnected);
        assert!(!snapshot.runner_disconnected);
        assert_eq!(snapshot.pod_size, None);
        assert_eq!(snapshot.revision, 0);
    }

    #[test]
    fn connection_handle_forwards_input_and_unsubscribe_identity() {
        let (cmd_tx, mut cmd_rx) = mpsc::unbounded();
        let (unsubscribe_tx, mut unsubscribe_rx) = mpsc::unbounded();
        let handle = ConnectionHandle::new(
            "pod-1".to_string(),
            "sub-7".to_string(),
            cmd_tx,
            unsubscribe_tx,
        );

        handle.send(vec![b'h', b'i', 0xff]);
        match cmd_rx.try_recv().unwrap() {
            Command::Send { data } => assert_eq!(data, "hi\u{fffd}"),
            _ => panic!("expected Send command"),
        }

        handle.unsubscribe();
        assert_eq!(
            unsubscribe_rx.try_recv().unwrap(),
            ("pod-1".to_string(), "sub-7".to_string())
        );
    }
}
// LCOV_EXCL_STOP
