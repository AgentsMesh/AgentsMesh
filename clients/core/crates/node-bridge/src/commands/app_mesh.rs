use napi_derive::napi;

use agentsmesh_types::proto_mesh_state_v1::ReplaceTopologyRequest;
use prost::Message as _;

use crate::AppState;

// Mesh state surface over the shared `runtime.state` (SSOT), mirroring
// app_autopilot.rs / app_loop.rs. fetch_topology fills the full topology;
// pod status/agent events patch individual nodes via event_dispatch
// (mesh_state.update_node_status), and app_get_mesh_node_json reads those
// patched nodes for the desktop realtime mirror.
#[napi]
impl AppState {
    // Single mesh node for the desktop realtime mirror (mirrors app_get_pod_json);
    // empty string = not a topology node → renderer skips.
    #[napi]
    pub fn app_get_mesh_node_json(&self, pod_key: String) -> String {
        match self.runtime.state.read().mesh.get_node_by_key(&pod_key) {
            Some(node) => serde_json::to_string(node).unwrap_or_default(),
            None => String::new(),
        }
    }

    #[napi]
    pub fn app_mesh_replace_topology(&self, req_bytes: Vec<u8>) -> napi::Result<()> {
        let req = ReplaceTopologyRequest::decode(&req_bytes[..])
            .map_err(|e| napi::Error::from_reason(format!("decode: {e}")))?;
        if let Some(topology) = req.topology {
            self.runtime.state.write().mesh.set_topology(topology);
        }
        Ok(())
    }
}
