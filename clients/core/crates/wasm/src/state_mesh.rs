use std::sync::Arc;

use agentsmesh_state::app_state::AppState;
use agentsmesh_types::proto_mesh_state_v1::ReplaceTopologyRequest;
use parking_lot::RwLock;
use prost::Message;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmMeshState {
    state: Arc<RwLock<AppState>>,
}

fn decode_err<E: std::fmt::Display>(e: E) -> JsValue {
    JsValue::from_str(&format!("decode: {e}"))
}

impl WasmMeshState {
    pub(crate) fn from_runtime(state: Arc<RwLock<AppState>>) -> Self {
        Self { state }
    }
}

#[wasm_bindgen]
impl WasmMeshState {
    #[wasm_bindgen(constructor)]
    pub fn new() -> Self {
        Self {
            state: Arc::new(RwLock::new(AppState::new())),
        }
    }

    pub fn topology_json(&self) -> JsValue {
        match self.state.read().mesh.topology() {
            Some(t) => JsValue::from_str(
                &serde_json::to_string(t).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    // Read side (B, zero-JSON): prost-encode the state topology (wire==state).
    // Web/desktop decode via fromBinary(MeshTopologySchema) + topologyToCache and
    // derive the per-node queries in TS. Empty bytes = no topology.
    pub fn topology_bytes(&self) -> Vec<u8> {
        match self.state.read().mesh.topology() {
            Some(t) => t.encode_to_vec(),
            None => Vec::new(),
        }
    }

    pub fn selected_node(&self) -> JsValue {
        match self.state.read().mesh.selected_node() {
            Some(key) => JsValue::from_str(key),
            None => JsValue::NULL,
        }
    }

    pub fn replace_topology(&self, req_bytes: &[u8]) -> Result<(), JsValue> {
        let req = ReplaceTopologyRequest::decode(req_bytes).map_err(decode_err)?;
        if let Some(topology) = req.topology {
            self.state.write().mesh.set_topology(topology);
        }
        Ok(())
    }

    pub fn clear_topology(&self) {
        self.state.write().mesh.clear_topology();
    }

    pub fn select_node(&self, pod_key: Option<String>) {
        self.state.write().mesh.select_node(pod_key);
    }

    pub fn get_node_json(&self, pod_key: &str) -> JsValue {
        match self.state.read().mesh.get_node_by_key(pod_key) {
            Some(n) => JsValue::from_str(
                &serde_json::to_string(n).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }

    pub fn get_edges_for_node_json(&self, pod_key: &str) -> String {
        let guard = self.state.read();
        let edges = guard.mesh.get_edges_for_node(pod_key);
        serde_json::to_string(&edges).unwrap_or_default()
    }

    pub fn get_channels_for_node_json(&self, pod_key: &str) -> String {
        let guard = self.state.read();
        let channels = guard.mesh.get_channels_for_node(pod_key);
        serde_json::to_string(&channels).unwrap_or_default()
    }

    pub fn get_nodes_by_runner_json(&self, runner_id: i64) -> String {
        let guard = self.state.read();
        let nodes = guard.mesh.get_nodes_by_runner(runner_id);
        serde_json::to_string(&nodes).unwrap_or_default()
    }

    pub fn get_active_nodes_json(&self) -> String {
        let guard = self.state.read();
        let nodes = guard.mesh.get_active_nodes();
        serde_json::to_string(&nodes).unwrap_or_default()
    }

    pub fn get_runner_info_json(&self, runner_id: i64) -> JsValue {
        match self.state.read().mesh.get_runner_info(runner_id) {
            Some(r) => JsValue::from_str(
                &serde_json::to_string(r).unwrap_or_default(),
            ),
            None => JsValue::NULL,
        }
    }
}
