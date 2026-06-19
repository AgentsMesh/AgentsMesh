use std::sync::Arc;

use agentsmesh_state::app_state::AppState;
use agentsmesh_types::proto_pod_v1::{ListPodsResponse, Pod};
use agentsmesh_types::proto_pod_state_v1::{
    InsertCreatedPodRequest, PatchPodPerpetualRequest,
    ApplyPodStatusEventRequest, ApplyPodTitleEventRequest,
    ApplyPodAliasEventRequest, ApplyAgentStatusEventRequest,
    ReplaceCachedPodsRequest,
    MarkPodTerminatedRequest,
};
use parking_lot::RwLock;
use prost::Message;
use wasm_bindgen::prelude::*;

/// View into `AppState.pods` exposed to JavaScript. Created via
/// `WasmApiClient::get_pod_state()` — the inner `Arc<RwLock<AppState>>`
/// is shared with the events dispatch hook + every other service, so
/// writes through any path are visible to readers through any path.
#[wasm_bindgen]
pub struct WasmPodState {
    state: Arc<RwLock<AppState>>,
}

fn decode_err<E: std::fmt::Display>(e: E) -> JsValue {
    JsValue::from_str(&format!("decode: {e}"))
}

impl WasmPodState {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmPodState {
    /// Stand-alone constructor for tests that don't have a full runtime.
    /// Creates an isolated `AppState` — events won't reach this instance.
    /// Production code MUST use `WasmApiClient::get_pod_state()`.
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::with_storage(crate::new_memory_backend()))),
        }
    }

    pub fn pods_json(&self) -> String {
        serde_json::to_string(self.state.read().pods.pods()).unwrap_or_default()
    }

    // Read side, zero-JSON: prost-encode state pods (reuse the cache wrapper).
    // Web/desktop decode via fromBinary + podToCache. Replaces pods_json — the
    // wire Pod and the cached Pod are the same proto, so this is identity.
    pub fn pods_bytes(&self) -> Vec<u8> {
        let pods = self.state.read().pods.pods().to_vec();
        ReplaceCachedPodsRequest { pods }.encode_to_vec()
    }

    pub fn current_pod_bytes(&self) -> Vec<u8> {
        match self.state.read().pods.current_pod() {
            Some(pod) => pod.encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn get_pod_bytes(&self, pod_key: &str) -> Vec<u8> {
        match self.state.read().pods.get_pod(pod_key) {
            Some(pod) => pod.encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn current_pod_json(&self) -> JsValue {
        match self.state.read().pods.current_pod() {
            Some(pod) => JsValue::from_str(
                &serde_json::to_string(pod).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn get_pod_json(&self, pod_key: &str) -> JsValue {
        match self.state.read().pods.get_pod(pod_key) {
            Some(pod) => JsValue::from_str(
                &serde_json::to_string(pod).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn insert_created_pod(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = InsertCreatedPodRequest::decode(req_bytes).map_err(decode_err)?;
        let pod = req.pod.ok_or_else(|| JsValue::from_str("missing pod"))?;
        let ts = if req.client_timestamp_ms == 0 { None } else { Some(req.client_timestamp_ms) };
        self.state.write().pods.upsert_pod(pod, ts);
        Ok(())
    }

    pub fn patch_pod_perpetual(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = PatchPodPerpetualRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().pods.patch_perpetual(&req.pod_key, req.perpetual);
        Ok(())
    }

    pub fn apply_pod_status_event(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ApplyPodStatusEventRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().pods.update_pod_status(
            &req.pod_key,
            &req.status,
            req.agent_status.as_deref(),
            req.error_code.as_deref(),
            req.error_message.as_deref(),
            None,
        );
        Ok(())
    }

    pub fn apply_pod_title_event(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ApplyPodTitleEventRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().pods.update_pod_title(&req.pod_key, &req.title, None);
        Ok(())
    }

    pub fn apply_pod_alias_event(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ApplyPodAliasEventRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().pods.update_pod_alias(&req.pod_key, req.alias.as_deref().unwrap_or(""));
        Ok(())
    }

    pub fn apply_agent_status_event(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ApplyAgentStatusEventRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().pods.update_agent_status(&req.pod_key, &req.agent_status);
        Ok(())
    }

    // Fetch→state: decode wire ListPodsResponse + fold into state. The wire Pod
    // IS the cache Pod (proto.pod.v1.Pod), so this is pure identity — no TS
    // fromProtoPod + podToProtoPod round-trip. Web fetch hands raw wire bytes
    // straight to Rust.
    pub fn apply_fetched_pods(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListPodsResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().pods.set_pods(resp.items);
        Ok(())
    }

    // Fetch→state for pagination load-more: upsert each fetched pod (append).
    pub fn apply_appended_pods(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListPodsResponse::decode(resp_bytes).map_err(decode_err)?;
        for pod in resp.items {
            self.state.write().pods.upsert_pod(pod, None);
        }
        Ok(())
    }

    pub fn mark_pod_terminated(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = MarkPodTerminatedRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().pods.update_pod_status(&req.pod_key, "terminated", None, None, None, None);
        Ok(())
    }

    pub fn remove_pod(&self, pod_key: &str) {
        self.state.write().pods.remove_pod(pod_key);
    }

    pub fn update_init_progress(
        &self, pod_key: &str, phase: &str, progress: f64, message: Option<String>,
    ) {
        self.state.write().pods.update_init_progress(pod_key, phase, progress, message.as_deref());
    }

    pub fn clear_init_progress(&self, pod_key: &str) {
        self.state.write().pods.clear_init_progress(pod_key);
    }
}
