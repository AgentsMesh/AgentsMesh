use std::sync::Arc;

use agentsmesh_transport::runtime::Runtime;
use futures::channel::{mpsc, oneshot};
use parking_lot::RwLock;

use super::{PodHandle, RelayConnectionPool};
use crate::command::Command;
use crate::driver::Driver;
use crate::types::{
    ConnectionHandle, GenerationAcpCallback, GenerationStatusCallback, OutputCallback, RelayStatus,
    RelayStatusInfo, StatusSnapshot,
};

impl<R: Runtime> RelayConnectionPool<R> {
    pub(super) fn add_subscriber(
        &self,
        pod_key: &str,
        subscription_id: &str,
        relay_url: &str,
        relay_token: &str,
        callback: OutputCallback,
        ready: Option<oneshot::Sender<()>>,
        listeners: Option<(String, GenerationStatusCallback, GenerationAcpCallback)>,
    ) -> (ConnectionHandle, u32) {
        tracing::info!(target: "relay", pod_key, %subscription_id, "subscribe");
        let (cmd_tx, generation, initial_status) = {
            let mut router = self.inner.write();
            let existing = router.pods.get(pod_key).map(|handle| {
                (
                    handle.cmd_tx.clone(),
                    handle.generation,
                    Arc::clone(&handle.snapshot),
                )
            });
            let (tx, generation, snapshot) = if let Some((tx, generation, snapshot)) = existing {
                let _ = tx.unbounded_send(Command::AddSubscriber {
                    sub_id: subscription_id.to_string(),
                    cb: callback,
                    ready,
                });
                (tx, generation, snapshot)
            } else {
                let (cmd_tx, cmd_rx) = mpsc::unbounded();
                let snapshot = Arc::new(RwLock::new(StatusSnapshot {
                    status: RelayStatus::Connecting,
                    runner_disconnected: false,
                    pod_size: None,
                    revision: 0,
                }));
                router.next_generation = router.next_generation.wrapping_add(1);
                if router.next_generation == 0 {
                    router.next_generation = 1;
                }
                let generation = router.next_generation;
                router.pods.insert(
                    pod_key.to_string(),
                    PodHandle {
                        cmd_tx: cmd_tx.clone(),
                        snapshot: Arc::clone(&snapshot),
                        generation,
                    },
                );
                Driver::spawn(
                    self.runtime.clone(),
                    Arc::clone(&self.inner),
                    pod_key.to_string(),
                    relay_url.to_string(),
                    relay_token.to_string(),
                    snapshot,
                    generation,
                    cmd_rx,
                    (subscription_id.to_string(), callback, ready),
                );
                let snapshot = router
                    .pods
                    .get(pod_key)
                    .map(|handle| Arc::clone(&handle.snapshot))
                    .expect("new relay driver handle");
                (cmd_tx, generation, snapshot)
            };

            let initial_status = listeners.and_then(|(lease_id, status, acp)| {
                router.bind_generation_listeners(pod_key, generation, lease_id, status, acp)
            });
            let status_info = {
                let state = snapshot.read();
                RelayStatusInfo {
                    status: state.status,
                    runner_disconnected: state.runner_disconnected,
                    revision: state.revision,
                }
            };
            (
                tx,
                generation,
                initial_status.map(|listener| (listener, status_info)),
            )
        };
        if let Some((listener, status_info)) = initial_status {
            listener.deliver(status_info);
        }
        (
            ConnectionHandle::new(
                pod_key.to_string(),
                subscription_id.to_string(),
                cmd_tx,
                self.unsubscribe_tx.clone(),
            ),
            generation,
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
}

// LCOV_EXCL_START: test-only code
#[cfg(all(test, not(target_arch = "wasm32")))]
mod tests {
    use std::sync::Arc;

    use agentsmesh_transport::runtime::PlatformRuntime;

    use super::*;

    #[tokio::test]
    async fn generation_counter_wraps_to_one_instead_of_publishing_zero() {
        let (pool, _rx) = RelayConnectionPool::with_runtime(PlatformRuntime);
        pool.inner.write().next_generation = u32::MAX;

        let (_handle, generation) = pool.add_subscriber(
            "pod-wrap",
            "sub-1",
            "not-a-relay-url",
            "token",
            Arc::new(|_| {}),
            None,
            None,
        );

        assert_eq!(generation, 1);
        assert_eq!(pool.inner.read().next_generation, 1);
        pool.disconnect("pod-wrap").await;
    }
}
// LCOV_EXCL_STOP
