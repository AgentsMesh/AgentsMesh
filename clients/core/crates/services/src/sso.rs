use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_sso_v1 as sso_proto;
use prost::Message;

pub struct SSOService {
    client: Arc<ApiClient>,
}

impl SSOService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Each method accepts a prost-encoded request body (`Vec<u8>`) and returns
    // a prost-encoded response body — matching the wasm bridge's
    // `Result<Vec<u8>, String>` surface (conventions §2.5).

    pub async fn discover_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = sso_proto::DiscoverRequest::decode(request_bytes)
            .map_err(|e| format!("decode discover request: {e}"))?;
        tracing::debug!(target: "sso", "discover");
        let resp = self.client.sso_discover_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn ldap_auth_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = sso_proto::LdapAuthRequest::decode(request_bytes)
            .map_err(|e| format!("decode ldap_auth request: {e}"))?;
        tracing::info!(target: "sso", domain = %req.domain, "ldap auth");
        let resp = self.client.sso_ldap_auth_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
