use std::sync::Arc;

use agentsmesh_transport::runtime::Runtime;
use futures::channel::oneshot;

use super::RelayConnectionPool;
use crate::error::RelayError;
use crate::types::{
    ConnectionHandle, GenerationAcpCallback, GenerationStatusCallback, OutputCallback,
};

impl<R: Runtime> RelayConnectionPool<R> {
    pub async fn subscribe(
        &self,
        pod_key: &str,
        subscription_id: &str,
        relay_url: &str,
        relay_token: &str,
        callback: OutputCallback,
    ) -> ConnectionHandle {
        self.add_subscriber(
            pod_key,
            subscription_id,
            relay_url,
            relay_token,
            callback,
            None,
            None,
        )
        .0
    }

    pub async fn subscribe_ready(
        &self,
        pod_key: &str,
        subscription_id: &str,
        relay_url: &str,
        relay_token: &str,
        callback: OutputCallback,
    ) -> Result<ConnectionHandle, RelayError> {
        let (ready_tx, ready_rx) = oneshot::channel();
        let (handle, _) = self.add_subscriber(
            pod_key,
            subscription_id,
            relay_url,
            relay_token,
            callback,
            Some(ready_tx),
            None,
        );
        wait_until_ready(ready_rx, pod_key, subscription_id).await?;
        Ok(handle)
    }

    pub async fn subscribe_ready_with_listeners(
        &self,
        pod_key: &str,
        subscription_id: &str,
        relay_url: &str,
        relay_token: &str,
        callback: OutputCallback,
        listener_lease_id: &str,
        status_listener: GenerationStatusCallback,
        acp_listener: GenerationAcpCallback,
        on_bound: Arc<dyn Fn(u32) + Send + Sync>,
    ) -> Result<ConnectionHandle, RelayError> {
        let (ready_tx, ready_rx) = oneshot::channel();
        let (handle, generation) = self.add_subscriber(
            pod_key,
            subscription_id,
            relay_url,
            relay_token,
            callback,
            Some(ready_tx),
            Some((listener_lease_id.to_string(), status_listener, acp_listener)),
        );
        on_bound(generation);
        wait_until_ready(ready_rx, pod_key, subscription_id).await?;
        Ok(handle)
    }
}

async fn wait_until_ready(
    ready: oneshot::Receiver<()>,
    pod_key: &str,
    subscription_id: &str,
) -> Result<(), RelayError> {
    ready.await.map_err(|_| {
        RelayError::NotConnected(format!(
            "{pod_key} subscriber {subscription_id} closed before baseline"
        ))
    })
}
