use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_repository_v1 as repo_proto;
use prost::Message;

pub struct RepositoryService {
    client: Arc<ApiClient>,
}

impl RepositoryService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }
    // -------- Connect-RPC (binary wire) --------
    //
    // Each `*_connect` method takes prost-encoded bytes and returns
    // prost-encoded bytes — matching the wasm bridge's `Result<Vec<u8>, String>`
    // surface (conventions §2.5). Caller (TS) encodes via
    // @bufbuild/protobuf .toBinary() and decodes via .fromBinary().

    pub async fn list_repositories_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::ListRepositoriesRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_repositories request: {e}"))?;
        let resp = self.client.list_repositories_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_repository_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::GetRepositoryRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_repository request: {e}"))?;
        let resp = self.client.get_repository_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_repository_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::CreateRepositoryRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_repository request: {e}"))?;
        let resp = self.client.create_repository_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_repository_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::UpdateRepositoryRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_repository request: {e}"))?;
        let resp = self.client.update_repository_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_repository_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::DeleteRepositoryRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_repository request: {e}"))?;
        let resp = self.client.delete_repository_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_repository_branches_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::ListRepositoryBranchesRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_repository_branches request: {e}"))?;
        let resp = self.client.list_repository_branches_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn sync_repository_branches_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::SyncRepositoryBranchesRequest::decode(request_bytes)
            .map_err(|e| format!("decode sync_repository_branches request: {e}"))?;
        let resp = self.client.sync_repository_branches_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_repository_merge_requests_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::ListRepositoryMergeRequestsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_repository_merge_requests request: {e}"))?;
        let resp = self.client.list_repository_merge_requests_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn register_repository_webhook_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::RegisterRepositoryWebhookRequest::decode(request_bytes)
            .map_err(|e| format!("decode register_repository_webhook request: {e}"))?;
        let resp = self.client.register_repository_webhook_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_repository_webhook_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::DeleteRepositoryWebhookRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_repository_webhook request: {e}"))?;
        let resp = self.client.delete_repository_webhook_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_repository_webhook_status_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::GetRepositoryWebhookStatusRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_repository_webhook_status request: {e}"))?;
        let resp = self.client.get_repository_webhook_status_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_repository_webhook_secret_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::GetRepositoryWebhookSecretRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_repository_webhook_secret request: {e}"))?;
        let resp = self.client.get_repository_webhook_secret_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn mark_repository_webhook_configured_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = repo_proto::MarkRepositoryWebhookConfiguredRequest::decode(request_bytes)
            .map_err(|e| format!("decode mark_repository_webhook_configured request: {e}"))?;
        let resp = self.client.mark_repository_webhook_configured_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
