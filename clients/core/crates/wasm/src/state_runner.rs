use std::sync::Arc;

use agentsmesh_state::app_state::AppState;
use agentsmesh_types::proto_runner_api_v1::{ListAvailableRunnersResponse, ListRunnersResponse};
use agentsmesh_types::proto_runner_state_v1::{
    PatchCachedRunnerRequest, RemoveCachedRunnerRequest, ReplaceAvailableRunnersRequest,
    ReplaceCachedRunnersRequest, SetCurrentRunnerRequest,
};
use parking_lot::RwLock;
use prost::Message;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmRunnerState {
    state: Arc<RwLock<AppState>>,
}

fn decode_err<E: std::fmt::Display>(e: E) -> JsValue {
    JsValue::from_str(&format!("decode: {e}"))
}

impl WasmRunnerState {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmRunnerState {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::with_storage(crate::new_memory_backend()))),
        }
    }

    pub fn runners_json(&self) -> String {
        serde_json::to_string(self.state.read().runners.runners()).unwrap_or_default()
    }

    pub fn available_runners_json(&self) -> String {
        serde_json::to_string(self.state.read().runners.available_runners())
            .unwrap_or_default()
    }

    pub fn current_runner_json(&self) -> JsValue {
        match self.state.read().runners.current_runner() {
            Some(r) => JsValue::from_str(
                &serde_json::to_string(r).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    // Read side (B, zero-JSON): prost-encode state into the same wrapper the
    // mutators decode, so the shared selectors decode bytes uniformly.
    pub fn runners_bytes(&self) -> Vec<u8> {
        let runners = self.state.read().runners.runners().to_vec();
        ReplaceCachedRunnersRequest { runners }.encode_to_vec()
    }

    pub fn available_runners_bytes(&self) -> Vec<u8> {
        let runners = self.state.read().runners.available_runners().to_vec();
        ReplaceAvailableRunnersRequest { runners }.encode_to_vec()
    }

    pub fn current_runner_bytes(&self) -> Vec<u8> {
        match self.state.read().runners.current_runner() {
            Some(r) => r.encode_to_vec(),
            None => Vec::new(),
        }
    }

    // Fetch→state (B): wire Runner == cache Runner, so decode the wire response
    // and fold into state directly — no TS fromProtoRunner/xToProto.
    pub fn apply_fetched_runners(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListRunnersResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().runners.set_runners(resp.items);
        Ok(())
    }

    pub fn apply_fetched_available_runners(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = ListAvailableRunnersResponse::decode(resp_bytes).map_err(decode_err)?;
        self.state.write().runners.set_available_runners(resp.items);
        Ok(())
    }

    pub fn set_current_runner_proto(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = SetCurrentRunnerRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().runners.set_current_runner(req.runner);
        Ok(())
    }

    // Fetch→state (B): decode the wire GetRunnerResponse + set current from its
    // runner field — no TS runnerToProtoRunner round-trip. relay_connections /
    // latest_runner_version are detail-page-only (read off a separate fetch).
    pub fn apply_fetched_current_runner(&self, resp_bytes: &[u8]) -> Result<(), JsValue> {
        let resp = agentsmesh_types::proto_runner_api_v1::GetRunnerResponse::decode(resp_bytes)
            .map_err(decode_err)?;
        self.state.write().runners.set_current_runner(resp.runner);
        Ok(())
    }

    pub fn patch_cached_runner(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = PatchCachedRunnerRequest::decode(req_bytes).map_err(decode_err)?;
        if let Some(runner) = req.runner {
            self.state.write().runners.upsert_runner(runner);
        }
        Ok(())
    }

    pub fn remove_cached_runner(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = RemoveCachedRunnerRequest::decode(req_bytes).map_err(decode_err)?;
        self.state.write().runners.remove_runner(req.runner_id);
        Ok(())
    }
}
