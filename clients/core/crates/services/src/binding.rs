// proto.binding.v1.BindingService Connect-RPC bridge. Thin owner of the
// shared ApiClient that forwards binary prost requests to the api-client
// `*_connect` methods.

use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_binding_v1 as bp;
use prost::Message;

use crate::wire;

pub struct BindingService {
    client: Arc<ApiClient>,
}

impl BindingService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub(crate) fn client(&self) -> &ApiClient {
        &self.client
    }
}

macro_rules! connect_bridge {
    ($name:ident, $req:ident, $client_call:ident) => {
        pub async fn $name(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
            let req = bp::$req::decode(request_bytes)
                .map_err(|e| format!("decode {}: {e}", stringify!($req)))?;
            tracing::debug!(target: "binding", rpc = stringify!($req));
            let resp = self.client().$client_call(&req).await.map_err(wire)?;
            Ok(resp.encode_to_vec())
        }
    };
}

impl BindingService {
    connect_bridge!(
        request_binding_connect,
        RequestBindingRequest,
        request_binding_connect
    );
    connect_bridge!(
        accept_binding_connect,
        AcceptBindingRequest,
        accept_binding_connect
    );
    connect_bridge!(
        reject_binding_connect,
        RejectBindingRequest,
        reject_binding_connect
    );
    connect_bridge!(unbind_connect, UnbindRequest, unbind_connect);
    connect_bridge!(
        request_scopes_connect,
        RequestScopesRequest,
        request_binding_scopes_connect
    );
    connect_bridge!(
        approve_scopes_connect,
        ApproveScopesRequest,
        approve_binding_scopes_connect
    );
    connect_bridge!(
        list_bindings_connect,
        ListBindingsRequest,
        list_bindings_connect
    );
    connect_bridge!(
        get_pending_bindings_connect,
        GetPendingBindingsRequest,
        get_pending_bindings_connect
    );
    connect_bridge!(
        get_bound_pods_connect,
        GetBoundPodsRequest,
        get_bound_pods_connect
    );
    connect_bridge!(
        check_binding_connect,
        CheckBindingRequest,
        check_binding_connect
    );
}
