use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::*;
use agentsmesh_types::proto_apikey_v1 as apikey_proto;
use prost::Message;

pub struct ApiKeyService {
    client: Arc<ApiClient>,
}

impl ApiKeyService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Each method accepts a prost-encoded request body (`Vec<u8>`) and returns
    // a prost-encoded response body — matching the wasm bridge's
    // `Result<Vec<u8>, String>` surface (conventions §2.5).
    //
    // org_slug is sourced from the caller-supplied request, not from
    // AuthManager — keeps these methods unit-testable. The TS adapter
    // populates org_slug before encoding (see clients/web/src/lib/api/apikey.ts).
    //
    // PR #345 lineage: `create_api_key_connect` surfaces `CreateApiKeyResponse`
    // — both api_key (tag 1) and raw_key (tag 2) round-trip through the
    // wire; the prost-encoded bytes are returned untouched to the bridge.

    pub async fn list_api_keys_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = apikey_proto::ListApiKeysRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_api_keys request: {e}"))?;
        let resp = self.client.list_api_keys_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_api_key_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = apikey_proto::GetApiKeyRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_api_key request: {e}"))?;
        let resp = self.client.get_api_key_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_api_key_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = apikey_proto::CreateApiKeyRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_api_key request: {e}"))?;
        let resp = self.client.create_api_key_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_api_key_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = apikey_proto::UpdateApiKeyRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_api_key request: {e}"))?;
        let resp = self.client.update_api_key_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn revoke_api_key_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = apikey_proto::RevokeApiKeyRequest::decode(request_bytes)
            .map_err(|e| format!("decode revoke_api_key request: {e}"))?;
        let resp = self.client.revoke_api_key_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_api_key_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = apikey_proto::DeleteApiKeyRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_api_key request: {e}"))?;
        let resp = self.client.delete_api_key_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    // -------- Legacy REST (JSON wire) — preserved during dual-track --------

    pub async fn list(&self) -> Result<String, String> {
        let resp = self.client.list_api_keys().await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn get(&self, id: i64) -> Result<String, String> {
        let resp = self.client.get_api_key(id).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn create(&self, json: &str) -> Result<String, String> {
        let req: CreateApiKeyRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client.create_api_key(&req).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn update(&self, id: i64, json: &str) -> Result<String, String> {
        let req: UpdateApiKeyRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client.update_api_key(id, &req).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn delete(&self, id: i64) -> Result<(), String> {
        self.client.delete_api_key(id).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn revoke(&self, id: i64) -> Result<(), String> {
        self.client.revoke_api_key(id).await.map_err(crate::wire)?;
        Ok(())
    }
}
