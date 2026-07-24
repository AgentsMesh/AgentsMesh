use std::sync::Arc;

use agentsmesh_transport::runtime::Runtime;

use super::{AcpListenerEntry, PoolRouter, RelayConnectionPool, StatusListenerEntry};
use crate::types::{
    AcpCallback, DisconnectCallback, GenerationAcpCallback, GenerationDisconnectCallback,
    GenerationStatusCallback, RelayStatus, RelayStatusInfo, StatusCallback,
};

impl PoolRouter {
    pub(super) fn listeners_match(&self, pod_key: &str, generation: u32, lease_id: &str) -> bool {
        self.status_listeners.get(pod_key).is_some_and(|listener| {
            listener.generation() == Some(generation) && listener.lease_id() == Some(lease_id)
        }) && self.acp_listeners.get(pod_key).is_some_and(|listener| {
            listener.generation() == Some(generation) && listener.lease_id() == Some(lease_id)
        })
    }

    pub(super) fn bind_generation_listeners(
        &mut self,
        pod_key: &str,
        generation: u32,
        lease_id: String,
        status_listener: GenerationStatusCallback,
        acp_listener: GenerationAcpCallback,
    ) -> Option<Arc<StatusListenerEntry>> {
        if self.listeners_match(pod_key, generation, &lease_id) {
            return None;
        }
        let status: StatusCallback = Arc::new(move |info| status_listener(generation, info));
        let acp: AcpCallback =
            Arc::new(move |msg_type, payload| acp_listener(generation, msg_type, payload));
        let status_entry = Arc::new(StatusListenerEntry::new(
            Some(generation),
            Some(lease_id.clone()),
            status,
        ));
        self.status_listeners
            .insert(pod_key.to_string(), Arc::clone(&status_entry));
        self.acp_listeners.insert(
            pod_key.to_string(),
            Arc::new(AcpListenerEntry::new(Some(generation), Some(lease_id), acp)),
        );
        Some(status_entry)
    }
}

impl<R: Runtime> RelayConnectionPool<R> {
    pub fn set_on_pod_disconnected(&self, callback: DisconnectCallback) {
        self.inner.write().on_pod_disconnected = Some(callback);
    }

    pub fn set_on_pod_generation_disconnected(&self, callback: GenerationDisconnectCallback) {
        self.inner.write().on_pod_generation_disconnected = Some(callback);
    }

    pub fn bind_listeners_if_active(
        &self,
        pod_key: &str,
        lease_id: &str,
        status_listener: GenerationStatusCallback,
        acp_listener: GenerationAcpCallback,
    ) -> u32 {
        let (generation, initial) = {
            let mut router = self.inner.write();
            let Some(handle) = router.pods.get(pod_key) else {
                return 0;
            };
            let generation = handle.generation;
            if router.listeners_match(pod_key, generation, lease_id) {
                return generation;
            }
            let info = {
                let snapshot = handle.snapshot.read();
                RelayStatusInfo {
                    status: snapshot.status,
                    runner_disconnected: snapshot.runner_disconnected,
                    revision: snapshot.revision,
                }
            };
            let listener = router
                .bind_generation_listeners(
                    pod_key,
                    generation,
                    lease_id.to_string(),
                    status_listener,
                    acp_listener,
                )
                .expect("listener lease changed after match check");
            (generation, (listener, info))
        };
        initial.0.deliver(initial.1);
        generation
    }

    pub async fn on_status_change(&self, pod_key: &str, listener: StatusCallback) {
        let entry = Arc::new(StatusListenerEntry::new(None, None, listener));
        let info = {
            let mut router = self.inner.write();
            router
                .status_listeners
                .insert(pod_key.to_string(), Arc::clone(&entry));
            router
                .pods
                .get(pod_key)
                .map(|handle| {
                    let snapshot = handle.snapshot.read();
                    RelayStatusInfo {
                        status: snapshot.status,
                        runner_disconnected: snapshot.runner_disconnected,
                        revision: snapshot.revision,
                    }
                })
                .unwrap_or(RelayStatusInfo {
                    status: RelayStatus::Disconnected,
                    runner_disconnected: false,
                    revision: 0,
                })
        };
        entry.deliver(info);
    }

    pub async fn on_acp_message(&self, pod_key: &str, listener: AcpCallback) {
        self.inner.write().acp_listeners.insert(
            pod_key.to_string(),
            Arc::new(AcpListenerEntry::new(None, None, listener)),
        );
    }
}

// LCOV_EXCL_START: test-only code
#[cfg(all(test, not(target_arch = "wasm32")))]
#[path = "listeners_tests.rs"]
mod listeners_tests;
// LCOV_EXCL_STOP
