use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_transport::runtime::{PlatformRuntime, Runtime};
use futures::channel::mpsc;
use parking_lot::RwLock;

use crate::command::Command;
use crate::driver::Driver;
use crate::error::RelayError;
use crate::types::{
    AcpCallback, ConnectionHandle, DisconnectCallback, OutputCallback, RelayStatus, RelayStatusInfo,
    StatusCallback, StatusSnapshot,
};

/// Thin routing table over per-pod driver actors. Holds only each pod's command
/// sender + status mirror, plus pool-scoped listeners (which may be registered
/// before a driver exists). All connection state lives inside the drivers.
#[derive(Clone)]
pub struct RelayConnectionPool<R: Runtime = PlatformRuntime> {
    inner: Arc<RwLock<PoolRouter>>,
    runtime: R,
    unsubscribe_tx: mpsc::UnboundedSender<(String, String)>,
}

pub(crate) struct PoolRouter {
    pub pods: HashMap<String, PodHandle>,
    pub status_listeners: HashMap<String, Vec<StatusCallback>>,
    pub acp_listeners: HashMap<String, Vec<AcpCallback>>,
    pub on_pod_disconnected: Option<DisconnectCallback>,
}

pub(crate) struct PodHandle {
    cmd_tx: mpsc::UnboundedSender<Command>,
    snapshot: Arc<RwLock<StatusSnapshot>>,
}

impl RelayConnectionPool<PlatformRuntime> {
    pub fn new() -> (Self, mpsc::UnboundedReceiver<(String, String)>) {
        Self::with_runtime(PlatformRuntime)
    }
}

impl<R: Runtime> RelayConnectionPool<R> {
    pub fn with_runtime(runtime: R) -> (Self, mpsc::UnboundedReceiver<(String, String)>) {
        let (tx, rx) = mpsc::unbounded();
        let inner = PoolRouter {
            pods: HashMap::new(),
            status_listeners: HashMap::new(),
            acp_listeners: HashMap::new(),
            on_pod_disconnected: None,
        };
        (
            Self {
                inner: Arc::new(RwLock::new(inner)),
                runtime,
                unsubscribe_tx: tx,
            },
            rx,
        )
    }

    pub fn set_on_pod_disconnected(&self, callback: DisconnectCallback) {
        self.inner.write().on_pod_disconnected = Some(callback);
    }

    pub async fn subscribe(
        &self,
        pod_key: &str,
        subscription_id: &str,
        relay_url: &str,
        relay_token: &str,
        callback: OutputCallback,
    ) -> ConnectionHandle {
        tracing::info!(target: "relay", pod_key, %subscription_id, "subscribe");
        let cmd_tx = {
            let mut router = self.inner.write();
            if let Some(handle) = router.pods.get(pod_key) {
                let tx = handle.cmd_tx.clone();
                let _ = tx.unbounded_send(Command::AddSubscriber {
                    sub_id: subscription_id.to_string(),
                    cb: callback,
                });
                tx
            } else {
                let (cmd_tx, cmd_rx) = mpsc::unbounded();
                // Mirror starts at Connecting (matching the driver's initial
                // state), so get_status during the first connect window doesn't
                // read the StatusSnapshot::default() Disconnected.
                let snapshot = Arc::new(RwLock::new(StatusSnapshot {
                    status: RelayStatus::Connecting,
                    runner_disconnected: false,
                    pod_size: None,
                }));
                router.pods.insert(
                    pod_key.to_string(),
                    PodHandle {
                        cmd_tx: cmd_tx.clone(),
                        snapshot: Arc::clone(&snapshot),
                    },
                );
                tracing::debug!(target: "relay", pod_key, "no live driver — spawning");
                Driver::spawn(
                    self.runtime.clone(),
                    Arc::clone(&self.inner),
                    pod_key.to_string(),
                    relay_url.to_string(),
                    relay_token.to_string(),
                    snapshot,
                    cmd_rx,
                    (subscription_id.to_string(), callback),
                );
                cmd_tx
            }
        };
        ConnectionHandle::new(
            pod_key.to_string(),
            subscription_id.to_string(),
            cmd_tx,
            self.unsubscribe_tx.clone(),
        )
    }

    pub async fn unsubscribe(&self, pod_key: &str, subscription_id: &str) {
        self.send_command(
            pod_key,
            Command::RemoveSubscriber {
                sub_id: subscription_id.to_string(),
            },
        );
    }

    pub async fn send(&self, pod_key: &str, data: &str) {
        self.send_command(pod_key, Command::Send { data: data.to_string() });
    }

    pub async fn send_resize(&self, pod_key: &str, cols: u16, rows: u16) {
        self.send_command(pod_key, Command::Resize { cols, rows, force: false });
    }

    pub async fn force_resize(&self, pod_key: &str, cols: u16, rows: u16) {
        self.send_command(pod_key, Command::Resize { cols, rows, force: true });
    }

    pub async fn send_acp_command(
        &self,
        pod_key: &str,
        command: &serde_json::Value,
    ) -> Result<(), RelayError> {
        // Only a data-ready link can carry ACP. Check the mirror (not just driver
        // existence): a driver that's connecting/backing-off returns NotConnected
        // instead of returning Ok while the command is silently dropped offline.
        let ready = self
            .inner
            .read()
            .pods
            .get(pod_key)
            .map(|h| h.snapshot.read().status == RelayStatus::Connected)
            .unwrap_or(false);
        if !ready {
            return Err(RelayError::NotConnected(pod_key.into()));
        }
        if self.send_command(pod_key, Command::SendAcp { command: command.clone() }) {
            Ok(())
        } else {
            Err(RelayError::NotConnected(pod_key.into()))
        }
    }

    pub async fn disconnect(&self, pod_key: &str) {
        tracing::info!(target: "relay", pod_key, "disconnect");
        self.send_command(pod_key, Command::Disconnect);
    }

    pub async fn disconnect_all(&self) {
        let txs: Vec<_> = self
            .inner
            .read()
            .pods
            .values()
            .map(|h| h.cmd_tx.clone())
            .collect();
        for tx in txs {
            let _ = tx.unbounded_send(Command::Disconnect);
        }
    }

    pub async fn on_status_change(&self, pod_key: &str, listener: StatusCallback) {
        {
            let mut router = self.inner.write();
            router
                .status_listeners
                .entry(pod_key.to_string())
                .or_default()
                .push(Arc::clone(&listener));
        }
        // Fire current status AFTER registering, reading the LATEST snapshot (not
        // one captured before the push): if the driver changes status between the
        // push and this fire, the listener still ends on the freshest value rather
        // than a stale captured one.
        let info = {
            let router = self.inner.read();
            router
                .pods
                .get(pod_key)
                .map(|h| {
                    let s = h.snapshot.read();
                    RelayStatusInfo {
                        status: s.status,
                        runner_disconnected: s.runner_disconnected,
                    }
                })
                .unwrap_or(RelayStatusInfo {
                    status: RelayStatus::Disconnected,
                    runner_disconnected: false,
                })
        };
        listener(info);
    }

    pub async fn on_acp_message(&self, pod_key: &str, listener: AcpCallback) {
        self.inner
            .write()
            .acp_listeners
            .entry(pod_key.to_string())
            .or_default()
            .push(listener);
    }

    pub async fn get_status(&self, pod_key: &str) -> RelayStatus {
        self.inner
            .read()
            .pods
            .get(pod_key)
            .map(|h| h.snapshot.read().status)
            .unwrap_or(RelayStatus::Disconnected)
    }

    pub async fn is_runner_disconnected(&self, pod_key: &str) -> bool {
        self.inner
            .read()
            .pods
            .get(pod_key)
            .map(|h| h.snapshot.read().runner_disconnected)
            .unwrap_or(false)
    }

    pub async fn get_pod_size(&self, pod_key: &str) -> Option<(u16, u16)> {
        self.inner
            .read()
            .pods
            .get(pod_key)
            .and_then(|h| h.snapshot.read().pod_size)
    }

    /// Forward a command to a live driver; false if the pod has no driver.
    fn send_command(&self, pod_key: &str, cmd: Command) -> bool {
        match self.inner.read().pods.get(pod_key) {
            Some(h) => h.cmd_tx.unbounded_send(cmd).is_ok(),
            None => false,
        }
    }
}

impl Default for RelayConnectionPool<PlatformRuntime> {
    fn default() -> Self {
        Self::new().0
    }
}
