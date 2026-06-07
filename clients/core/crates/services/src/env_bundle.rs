// EnvBundleService — thin Connect-RPC wrapper. The renderer drives this
// through binary wire-bytes (Uint8Array in TS / `Vec<u8>` here). The legacy
// JSON facade was removed alongside the REST backend in R6 — the wasm
// surface now mirrors the user_credential pattern (one entry per RPC,
// caller encodes/decodes the prost messages on the TS side).

use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_env_bundle_v1 as eb_proto;
use prost::Message;

pub struct EnvBundleService {
    client: Arc<ApiClient>,
}

impl EnvBundleService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn list_env_bundles_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = eb_proto::ListEnvBundlesRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_env_bundles request: {e}"))?;
        tracing::debug!(target: "env_bundle", "list env bundles");
        let resp = self
            .client
            .list_user_env_bundles_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_env_bundle_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = eb_proto::GetEnvBundleRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_env_bundle request: {e}"))?;
        tracing::debug!(target: "env_bundle", env_bundle_id = req.id, "get env bundle");
        let resp = self
            .client
            .get_user_env_bundle_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_env_bundle_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = eb_proto::CreateEnvBundleRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_env_bundle request: {e}"))?;
        tracing::info!(target: "env_bundle", kind = %req.kind, "create env bundle");
        let resp = self
            .client
            .create_user_env_bundle_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_env_bundle_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = eb_proto::UpdateEnvBundleRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_env_bundle request: {e}"))?;
        tracing::info!(target: "env_bundle", env_bundle_id = req.id, "update env bundle");
        let resp = self
            .client
            .update_user_env_bundle_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_env_bundle_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = eb_proto::DeleteEnvBundleRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_env_bundle request: {e}"))?;
        tracing::info!(target: "env_bundle", env_bundle_id = req.id, "delete env bundle");
        let resp = self
            .client
            .delete_user_env_bundle_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn set_primary_env_bundle_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = eb_proto::SetPrimaryEnvBundleRequest::decode(request_bytes)
            .map_err(|e| format!("decode set_primary_env_bundle request: {e}"))?;
        tracing::info!(target: "env_bundle", env_bundle_id = req.id, "set primary env bundle");
        let resp = self
            .client
            .set_primary_env_bundle_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
