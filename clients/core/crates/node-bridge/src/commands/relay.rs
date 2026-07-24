use napi::threadsafe_function::ThreadsafeFunction;
use napi_derive::napi;

use crate::AppState;

mod listener_callbacks;

// Terminal data-plane relay surface over the shared `RelayConnectionPool` (the
// SSOT). The pool runs natively in the main process; `main/relay.ts` provides
// the `on_output`/`on_status`/`on_acp` ThreadsafeFunctions that fan bytes out to
// the renderer via `webContents.send`, and input/resize/acp come back as the
// `relay_*` commands below. Mirrors the WasmRelayManager surface (web) so the
// shared renderer relay adapter is platform-symmetric.
fn err(e: impl std::fmt::Display) -> napi::Error {
    napi::Error::from_reason(e.to_string())
}

#[napi]
impl AppState {
    #[napi]
    pub async fn relay_subscribe(
        &self,
        pod_key: String,
        subscription_id: String,
        relay_url: String,
        token: String,
        on_output: ThreadsafeFunction<Vec<u8>>,
        on_status: ThreadsafeFunction<String>,
        on_acp: ThreadsafeFunction<String>,
        on_bound: ThreadsafeFunction<u32>,
        listener_lease_id: String,
    ) -> napi::Result<()> {
        self.relay
            .subscribe_ready_with_listeners(
                &pod_key,
                &subscription_id,
                &relay_url,
                &token,
                listener_callbacks::output(on_output),
                &listener_lease_id,
                listener_callbacks::generation_status(on_status),
                listener_callbacks::generation_acp(on_acp),
                listener_callbacks::bound(on_bound),
            )
            .await
            .map_err(err)?;
        Ok(())
    }

    /// Rebind the desktop fan-out callbacks to the currently active driver.
    /// Returns 0 when a subscribe has not published that driver yet.
    #[napi]
    pub async fn relay_bind_pod_listeners(
        &self,
        pod_key: String,
        on_status: ThreadsafeFunction<String>,
        on_acp: ThreadsafeFunction<String>,
        listener_lease_id: String,
    ) -> u32 {
        self.relay.bind_listeners_if_active(
            &pod_key,
            &listener_lease_id,
            listener_callbacks::generation_status(on_status),
            listener_callbacks::generation_acp(on_acp),
        )
    }

    #[napi]
    pub async fn relay_unsubscribe(&self, pod_key: String, subscription_id: String) {
        self.relay.unsubscribe(&pod_key, &subscription_id).await;
    }

    #[napi]
    pub async fn relay_send(&self, pod_key: String, data: String) {
        self.relay.send(&pod_key, &data).await;
    }

    #[napi]
    pub async fn relay_send_resize(&self, pod_key: String, cols: u16, rows: u16) {
        self.relay.send_resize(&pod_key, cols, rows).await;
    }

    #[napi]
    pub async fn relay_force_resize(&self, pod_key: String, cols: u16, rows: u16) {
        self.relay.force_resize(&pod_key, cols, rows).await;
    }

    #[napi]
    pub async fn relay_send_acp_command(
        &self,
        pod_key: String,
        command: String,
    ) -> napi::Result<()> {
        let val: serde_json::Value = serde_json::from_str(&command).map_err(err)?;
        self.relay
            .send_acp_command(&pod_key, &val)
            .await
            .map_err(err)
    }

    #[napi]
    pub async fn relay_disconnect(&self, pod_key: String) {
        self.relay.disconnect(&pod_key).await;
    }

    #[napi]
    pub async fn relay_disconnect_all(&self) {
        self.relay.disconnect_all().await;
    }

    #[napi]
    pub async fn relay_get_status(&self, pod_key: String) -> String {
        self.relay.get_status(&pod_key).await.to_string()
    }

    #[napi]
    pub async fn relay_is_runner_disconnected(&self, pod_key: String) -> bool {
        self.relay.is_runner_disconnected(&pod_key).await
    }

    /// `[cols, rows]` or empty when the pod size is unknown.
    #[napi]
    pub async fn relay_get_pod_size(&self, pod_key: String) -> Vec<u16> {
        self.relay
            .get_pod_size(&pod_key)
            .await
            .map(|(c, r)| vec![c, r])
            .unwrap_or_default()
    }

    /// Pod-disconnected sink delivers `{"podKey","generation"}` JSON so main
    /// can reject teardown notifications from an older driver generation.
    #[napi]
    pub async fn relay_on_pod_disconnected(
        &self,
        on_disconnect: ThreadsafeFunction<String>,
    ) -> napi::Result<()> {
        self.relay
            .set_on_pod_generation_disconnected(listener_callbacks::generation_disconnect(
                on_disconnect,
            ));
        Ok(())
    }
}
