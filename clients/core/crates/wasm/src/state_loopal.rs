use std::sync::Arc;

use agentsmesh_state::app_state::AppState;
use agentsmesh_state::loopal_dispatch;
use parking_lot::RwLock;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmLoopalManager {
    state: Arc<RwLock<AppState>>,
}

impl WasmLoopalManager {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmLoopalManager {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::new())),
        }
    }

    pub fn dispatch_event(&self, pod_key: &str, event_type: &str, data_json: &str) {
        let data = serde_json::from_str(data_json).unwrap_or(serde_json::Value::Null);
        loopal_dispatch::dispatch_event(&mut self.state.write().loopal, pod_key, event_type, &data);
    }

    pub fn dispatch_snapshot(&self, pod_key: &str, snapshot_json: &str) {
        let snap = serde_json::from_str(snapshot_json).unwrap_or(serde_json::Value::Null);
        loopal_dispatch::dispatch_snapshot(&mut self.state.write().loopal, pod_key, &snap);
    }

    pub fn get_session_json(&self, pod_key: &str) -> JsValue {
        match self.state.read().loopal.get(pod_key) {
            Some(s) => JsValue::from_str(&serde_json::to_string(s).unwrap_or_default()),
            None => JsValue::NULL,
        }
    }

    pub fn clear_session(&self, pod_key: &str) {
        self.state.write().loopal.clear(pod_key);
    }
}
