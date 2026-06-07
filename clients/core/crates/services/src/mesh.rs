use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_mesh_state_v1::ReplaceTopologyRequest;
use agentsmesh_types::proto_mesh_v1 as mp;
use prost::Message;

pub struct MeshService {
    client: Arc<ApiClient>,
}

impl MeshService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    /// Crate-local accessor used by mesh_connect.rs to forward to the
    /// underlying api-client `*_connect` methods.
    pub(crate) fn client_ref(&self) -> &ApiClient {
        &self.client
    }

    /// Networking-only: fetch the topology and return it wrapped in a
    /// prost-encoded `ReplaceTopologyRequest`, ready to feed the state surface
    /// (WasmMeshState::replace_topology / app_mesh_replace_topology). The mesh
    /// cache SSOT is `runtime.state.mesh`; this service holds no state.
    pub async fn fetch_topology(&self) -> Result<Vec<u8>, String> {
        let req = mp::GetMeshTopologyRequest {
            org_slug: self.client.current_org_slug(),
        };
        tracing::debug!(target: "mesh", org_slug = %req.org_slug, "fetch topology");
        let topo = self.client
            .get_mesh_topology_connect(&req)
            .await.map_err(crate::wire)?;
        let replace = ReplaceTopologyRequest { topology: Some(topo) };
        Ok(replace.encode_to_vec())
    }
}

// =============================================================================
// Connect-RPC bridge methods. Binary in (prost-encoded), binary out — same wire
// the wasm/node-bridge layers speak.
// =============================================================================

macro_rules! connect_bridge {
    ($name:ident, $req:ident, $client_call:ident) => {
        pub async fn $name(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
            let req = mp::$req::decode(request_bytes)
                .map_err(|e| format!("decode {}: {e}", stringify!($req)))?;
            tracing::debug!(target: "mesh", rpc = stringify!($req));
            let resp = self.client_ref().$client_call(&req).await.map_err(crate::wire)?;
            Ok(resp.encode_to_vec())
        }
    };
}

impl MeshService {
    connect_bridge!(
        get_mesh_topology_connect,
        GetMeshTopologyRequest,
        get_mesh_topology_connect
    );
    connect_bridge!(
        get_ticket_pods_connect,
        GetTicketPodsRequest,
        get_ticket_pods_connect
    );
    connect_bridge!(
        batch_get_ticket_pods_connect,
        BatchGetTicketPodsRequest,
        batch_get_ticket_pods_connect
    );
    connect_bridge!(
        create_pod_for_ticket_connect,
        CreatePodForTicketRequest,
        create_pod_for_ticket_connect
    );
}
