use agentsmesh_transport::runtime::Runtime;

use super::RelayConnectionPool;
use crate::command::Command;
use crate::error::RelayError;
use crate::types::RelayStatus;

impl<R: Runtime> RelayConnectionPool<R> {
    pub async fn send(&self, pod_key: &str, data: &str) {
        self.send_command(
            pod_key,
            Command::Send {
                data: data.to_string(),
            },
        );
    }

    pub async fn send_resize(&self, pod_key: &str, cols: u16, rows: u16) {
        self.send_command(
            pod_key,
            Command::Resize {
                cols,
                rows,
                force: false,
            },
        );
    }

    pub async fn force_resize(&self, pod_key: &str, cols: u16, rows: u16) {
        self.send_command(
            pod_key,
            Command::Resize {
                cols,
                rows,
                force: true,
            },
        );
    }

    pub async fn send_acp_command(
        &self,
        pod_key: &str,
        command: &serde_json::Value,
    ) -> Result<(), RelayError> {
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
        if self.send_command(
            pod_key,
            Command::SendAcp {
                command: command.clone(),
            },
        ) {
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
}

// LCOV_EXCL_START: test-only code
#[cfg(all(test, not(target_arch = "wasm32")))]
mod tests {
    use agentsmesh_transport::runtime::PlatformRuntime;

    use super::*;

    #[tokio::test]
    async fn disconnect_all_notifies_every_active_driver() {
        let (pool, _rx) = RelayConnectionPool::with_runtime(PlatformRuntime);
        let (mut first, mut second) = {
            let mut router = pool.inner.write();
            (
                router.insert_test_pod("pod-1", 1, RelayStatus::Connected),
                router.insert_test_pod("pod-2", 2, RelayStatus::Connecting),
            )
        };

        pool.disconnect_all().await;
        assert!(matches!(first.try_recv(), Ok(Command::Disconnect)));
        assert!(matches!(second.try_recv(), Ok(Command::Disconnect)));
    }

    #[tokio::test]
    async fn connected_acp_send_reports_closed_driver_channel() {
        let (pool, _rx) = RelayConnectionPool::with_runtime(PlatformRuntime);
        let cmd_rx = pool
            .inner
            .write()
            .insert_test_pod("pod-1", 1, RelayStatus::Connected);
        drop(cmd_rx);

        let result = pool
            .send_acp_command("pod-1", &serde_json::json!({"command": "prompt"}))
            .await;
        assert!(matches!(result, Err(RelayError::NotConnected(_))));
    }

    #[tokio::test]
    async fn connected_acp_send_preserves_the_json_command() {
        let (pool, _rx) = RelayConnectionPool::with_runtime(PlatformRuntime);
        let mut cmd_rx = pool
            .inner
            .write()
            .insert_test_pod("pod-1", 1, RelayStatus::Connected);
        let command = serde_json::json!({"command": "prompt", "text": "hello"});

        pool.send_acp_command("pod-1", &command).await.unwrap();
        match cmd_rx.try_recv().unwrap() {
            Command::SendAcp { command: actual } => assert_eq!(actual, command),
            _ => panic!("expected SendAcp command"),
        }
    }
}
// LCOV_EXCL_STOP
