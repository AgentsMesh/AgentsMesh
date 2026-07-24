use std::collections::HashMap;
use std::sync::Arc;

use agentsmesh_transport::runtime::{PlatformRuntime, Runtime};
use futures::channel::mpsc;
use parking_lot::RwLock;

use crate::command::Command;
use crate::types::{DisconnectCallback, GenerationDisconnectCallback, RelayStatus, StatusSnapshot};

mod commands;
mod listener_entries;
mod listeners;
mod subscription_readiness;
mod subscriptions;

use listener_entries::{AcpListenerEntry, StatusListenerEntry};

#[derive(Clone)]
pub struct RelayConnectionPool<R: Runtime = PlatformRuntime> {
    inner: Arc<RwLock<PoolRouter>>,
    runtime: R,
    unsubscribe_tx: mpsc::UnboundedSender<(String, String)>,
}

pub(crate) struct PoolRouter {
    pub pods: HashMap<String, PodHandle>,
    pub next_generation: u32,
    pub status_listeners: HashMap<String, Arc<StatusListenerEntry>>,
    pub acp_listeners: HashMap<String, Arc<AcpListenerEntry>>,
    pub on_pod_disconnected: Option<DisconnectCallback>,
    pub on_pod_generation_disconnected: Option<GenerationDisconnectCallback>,
}

pub(crate) struct PodHandle {
    cmd_tx: mpsc::UnboundedSender<Command>,
    snapshot: Arc<RwLock<StatusSnapshot>>,
    pub(crate) generation: u32,
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
            next_generation: 0,
            status_listeners: HashMap::new(),
            acp_listeners: HashMap::new(),
            on_pod_disconnected: None,
            on_pod_generation_disconnected: None,
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

    pub async fn get_status(&self, pod_key: &str) -> RelayStatus {
        self.inner
            .read()
            .pods
            .get(pod_key)
            .map(|handle| handle.snapshot.read().status)
            .unwrap_or(RelayStatus::Disconnected)
    }

    pub async fn is_runner_disconnected(&self, pod_key: &str) -> bool {
        self.inner
            .read()
            .pods
            .get(pod_key)
            .map(|handle| handle.snapshot.read().runner_disconnected)
            .unwrap_or(false)
    }

    pub async fn get_pod_size(&self, pod_key: &str) -> Option<(u16, u16)> {
        self.inner
            .read()
            .pods
            .get(pod_key)
            .and_then(|handle| handle.snapshot.read().pod_size)
    }

    fn send_command(&self, pod_key: &str, command: Command) -> bool {
        match self.inner.read().pods.get(pod_key) {
            Some(handle) => handle.cmd_tx.unbounded_send(command).is_ok(),
            None => false,
        }
    }
}

// LCOV_EXCL_START: test-only code
#[cfg(test)]
impl PoolRouter {
    pub(crate) fn insert_test_pod(
        &mut self,
        pod_key: &str,
        generation: u32,
        status: RelayStatus,
    ) -> mpsc::UnboundedReceiver<Command> {
        let (cmd_tx, cmd_rx) = mpsc::unbounded();
        self.pods.insert(
            pod_key.to_string(),
            PodHandle {
                cmd_tx,
                snapshot: Arc::new(RwLock::new(StatusSnapshot {
                    status,
                    runner_disconnected: false,
                    pod_size: None,
                    revision: 0,
                })),
                generation,
            },
        );
        cmd_rx
    }
}
// LCOV_EXCL_STOP

impl Default for RelayConnectionPool<PlatformRuntime> {
    fn default() -> Self {
        Self::new().0
    }
}
