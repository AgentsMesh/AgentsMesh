use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_repository_v1 as repo_proto;
use agentsmesh_types::*;
use prost::Message;

pub struct RepositoryService {
    client: Arc<ApiClient>,
}

impl RepositoryService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn list(&self) -> Result<String, String> {
        let resp = self.client.list_repositories().await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn get(&self, id: i64) -> Result<String, String> {
        let resp = self.client.get_repository(id).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn create(&self, json: &str) -> Result<String, String> {
        let req: CreateRepositoryRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client.create_repository(&req).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn update(&self, id: i64, json: &str) -> Result<String, String> {
        let req: UpdateRepositoryRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client.update_repository(id, &req).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn delete(&self, id: i64) -> Result<(), String> {
        self.client.delete_repository(id).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn list_branches(&self, id: i64) -> Result<String, String> {
        let resp = self.client.list_repository_branches(id).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn sync_branches(&self, id: i64, json: &str) -> Result<String, String> {
        let req: SyncBranchesRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .sync_repository_branches(id, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn register_webhook(&self, id: i64) -> Result<(), String> {
        self.client.register_repository_webhook(id).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn delete_webhook(&self, id: i64) -> Result<(), String> {
        self.client.delete_repository_webhook(id).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn get_webhook_status(&self, id: i64) -> Result<String, String> {
        let resp = self.client
            .get_repository_webhook_status(id)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn get_webhook_secret(&self, id: i64) -> Result<String, String> {
        let resp = self.client
            .get_repository_webhook_secret(id)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn list_merge_requests(
        &self, id: i64, branch: Option<String>, state: Option<String>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_repository_merge_requests(id, branch.as_deref(), state.as_deref())
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn mark_webhook_configured(&self, id: i64) -> Result<(), String> {
        let path = self.client.org_path(&format!("/repositories/{id}/webhook/configured"));
        self.client
            .post::<agentsmesh_types::EmptyResponse>(&path, &serde_json::json!({}))
            .await.map_err(crate::wire)?;
        Ok(())
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
