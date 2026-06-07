use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_grant_v1 as gp;
use prost::Message;

pub struct GrantService {
    client: Arc<ApiClient>,
}

impl GrantService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // TS encodes via @bufbuild/protobuf .toBinary() → wasm bridge → here →
    // ApiClient.*_connect (binary in / binary out, conventions §2.5). No
    // JSON path on the client.

    pub async fn list_grants_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = gp::ListGrantsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_grants request: {e}"))?;
        tracing::debug!(target: "grant", org_slug = %req.org_slug, resource_type = %req.resource_type, resource_id = %req.resource_id, "list grants");
        let resp = self.client.list_grants_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_grant_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = gp::CreateGrantRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_grant request: {e}"))?;
        tracing::info!(target: "grant", org_slug = %req.org_slug, resource_type = %req.resource_type, resource_id = %req.resource_id, user_id = req.user_id, "create grant");
        let resp = self.client.create_grant_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_grant_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = gp::DeleteGrantRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_grant request: {e}"))?;
        tracing::info!(target: "grant", org_slug = %req.org_slug, resource_type = %req.resource_type, resource_id = %req.resource_id, grant_id = req.grant_id, "delete grant");
        let resp = self.client.delete_grant_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
