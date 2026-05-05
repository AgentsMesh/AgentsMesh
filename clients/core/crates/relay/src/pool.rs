use std::collections::HashMap;
use std::sync::Arc;
use web_time::Instant;

use agentsmesh_protocol::{encode_json_message, encode_message, MsgType};
use agentsmesh_transport::runtime::{PlatformRuntime, Runtime, TaskHandle};
use tokio::sync::{mpsc, RwLock};

use crate::error::RelayError;
use crate::retry;
use crate::types::{
    AcpCallback, ConnectionHandle, ConnectionState, OutputCallback, RelayStatus, StatusCallback,
};

#[derive(Clone)]
pub struct RelayConnectionPool<R: Runtime = PlatformRuntime> {
    pub(crate) inner: Arc<RwLock<PoolInner<R>>>,
    pub(crate) runtime: R,
}

pub(crate) struct PoolInner<R: Runtime = PlatformRuntime> {
    pub connections: HashMap<String, ConnectionState<R>>,
    pub status_listeners: HashMap<String, Vec<StatusCallback>>,
    pub acp_listeners: HashMap<String, Vec<AcpCallback>>,
    pub last_inputs: HashMap<String, (String, Instant)>,
    pub resize_debounce: HashMap<String, R::TaskHandle>,
    pub unsubscribe_tx: mpsc::UnboundedSender<(String, String)>,
}

impl RelayConnectionPool<PlatformRuntime> {
    pub fn new() -> (Self, mpsc::UnboundedReceiver<(String, String)>) {
        Self::with_runtime(PlatformRuntime)
    }
}

impl<R: Runtime> RelayConnectionPool<R> {
    pub fn with_runtime(runtime: R) -> (Self, mpsc::UnboundedReceiver<(String, String)>) {
        let (tx, rx) = mpsc::unbounded_channel();
        let inner = PoolInner {
            connections: HashMap::new(),
            status_listeners: HashMap::new(),
            acp_listeners: HashMap::new(),
            last_inputs: HashMap::new(),
            resize_debounce: HashMap::new(),
            unsubscribe_tx: tx,
        };
        (
            Self {
                inner: Arc::new(RwLock::new(inner)),
                runtime,
            },
            rx,
        )
    }

    pub async fn subscribe(
        &self,
        pod_key: &str,
        subscription_id: &str,
        relay_url: &str,
        relay_token: &str,
        callback: OutputCallback,
    ) -> ConnectionHandle {
        let needs_connect = {
            let mut inner = self.inner.write().await;
            let is_new = !inner.connections.contains_key(pod_key);
            let conn = inner
                .connections
                .entry(pod_key.to_string())
                .or_insert_with(|| ConnectionState::new(relay_url.into(), relay_token.into()));

            if let Some(h) = conn.disconnect_handle.take() {
                h.abort();
            }
            conn.subscribers
                .insert(subscription_id.to_string(), callback);

            if !is_new && conn.status == RelayStatus::Connected {
                if let Some(tx) = &conn.ws_write_tx {
                    let _ = tx.send(encode_message(MsgType::Resync, &[]));
                }
            }
            is_new
        };

        if needs_connect {
            let pool = self.clone();
            let pk = pod_key.to_string();
            self.runtime.spawn(Box::pin(async move {
                if let Err(e) = pool.connect_pod(&pk).await {
                    tracing::warn!("connect_pod failed for {pk}: {e}");
                    pool.schedule_reconnect(&pk);
                }
            }));
        }

        let inner = self.inner.read().await;
        let conn = inner.connections.get(pod_key);
        let write_tx = conn
            .and_then(|c| c.ws_write_tx.clone())
            .unwrap_or_else(|| mpsc::unbounded_channel().0);
        ConnectionHandle::new(
            pod_key.to_string(),
            subscription_id.to_string(),
            write_tx,
            inner.unsubscribe_tx.clone(),
        )
    }

    pub async fn unsubscribe(&self, pod_key: &str, subscription_id: &str) {
        let pool = self.inner.clone();
        let mut inner = pool.write().await;
        let Some(conn) = inner.connections.get_mut(pod_key) else {
            return;
        };
        conn.subscribers.remove(subscription_id);

        if conn.subscribers.is_empty() && conn.disconnect_handle.is_none() {
            let pool_ref = self.inner.clone();
            let pk = pod_key.to_string();
            let rt = self.runtime.clone();
            conn.disconnect_handle = Some(self.runtime.spawn(Box::pin(async move {
                rt.sleep(std::time::Duration::from_millis(retry::DISCONNECT_DELAY_MS))
                    .await;
                let mut inner = pool_ref.write().await;
                if let Some(c) = inner.connections.get(pk.as_str()) {
                    if c.subscribers.is_empty() {
                        Self::disconnect_inner(&mut inner, &pk);
                    }
                }
            })));
        }
    }

    pub async fn send(&self, pod_key: &str, data: &str) {
        let mut inner = self.inner.write().await;
        let Some(conn) = inner.connections.get(pod_key) else {
            return;
        };
        if conn.status != RelayStatus::Connected {
            return;
        }
        let tx = match &conn.ws_write_tx {
            Some(tx) => tx.clone(),
            None => return,
        };

        if data.len() > 1 {
            let now = Instant::now();
            if let Some((last_data, last_time)) = inner.last_inputs.get(pod_key) {
                if last_data == data
                    && now.duration_since(*last_time).as_millis()
                        < retry::INPUT_DEDUP_WINDOW_MS as u128
                {
                    return;
                }
            }
            inner
                .last_inputs
                .insert(pod_key.to_string(), (data.to_string(), now));
        }

        let msg = encode_message(MsgType::Input, data.as_bytes());
        let _ = tx.send(msg);
    }

    pub async fn send_resize(&self, pod_key: &str, cols: u16, rows: u16) {
        if cols == 0 || rows == 0 {
            return;
        }
        let mut inner = self.inner.write().await;
        if let Some(h) = inner.resize_debounce.remove(pod_key) {
            h.abort();
        }
        let pool = self.inner.clone();
        let pk = pod_key.to_string();
        let rt = self.runtime.clone();
        let handle = self.runtime.spawn(Box::pin(async move {
            rt.sleep(std::time::Duration::from_millis(retry::RESIZE_DEBOUNCE_MS))
                .await;
            let inner = pool.read().await;
            if let Some(conn) = inner.connections.get(&pk) {
                Self::do_send_resize(conn, cols, rows);
            }
        }));
        inner.resize_debounce.insert(pod_key.to_string(), handle);
    }

    pub async fn send_acp_command(
        &self,
        pod_key: &str,
        command: &serde_json::Value,
    ) -> Result<(), RelayError> {
        let inner = self.inner.read().await;
        let conn = inner
            .connections
            .get(pod_key)
            .ok_or_else(|| RelayError::NotConnected(pod_key.into()))?;
        let tx = conn
            .ws_write_tx
            .as_ref()
            .ok_or_else(|| RelayError::NotConnected(pod_key.into()))?;
        let msg = encode_json_message(MsgType::AcpCommand, command)?;
        tx.send(msg)
            .map_err(|e| RelayError::Send(e.to_string()))?;
        Ok(())
    }
}

impl Default for RelayConnectionPool<PlatformRuntime> {
    fn default() -> Self {
        Self::new().0
    }
}
