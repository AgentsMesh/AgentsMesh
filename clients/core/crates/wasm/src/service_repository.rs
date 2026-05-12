use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::*;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmRepositoryService {
    client: Arc<ApiClient>,
}

#[wasm_bindgen]
impl WasmRepositoryService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn list(&self) -> Result<String, String> {
        let resp = self.client.list_repositories().await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn get(&self, id: i64) -> Result<String, String> {
        let resp = self.client.get_repository(id).await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn create(&self, json: &str) -> Result<String, String> {
        let req: CreateRepositoryRequest = serde_json::from_str(json).map_err(agentsmesh_services::wire)?;
        let resp = self.client.create_repository(&req).await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn update(&self, id: i64, json: &str) -> Result<String, String> {
        let req: UpdateRepositoryRequest = serde_json::from_str(json).map_err(agentsmesh_services::wire)?;
        let resp = self.client.update_repository(id, &req).await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn delete(&self, id: i64) -> Result<(), String> {
        self.client.delete_repository(id).await.map_err(agentsmesh_services::wire)?;
        Ok(())
    }

    pub async fn list_branches(&self, id: i64) -> Result<String, String> {
        let resp = self.client.list_repository_branches(id).await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn sync_branches(&self, id: i64, json: &str) -> Result<String, String> {
        let req: SyncBranchesRequest = serde_json::from_str(json).map_err(agentsmesh_services::wire)?;
        let resp = self.client
            .sync_repository_branches(id, &req)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn register_webhook(&self, id: i64) -> Result<(), String> {
        self.client.register_repository_webhook(id).await.map_err(agentsmesh_services::wire)?;
        Ok(())
    }

    pub async fn delete_webhook(&self, id: i64) -> Result<(), String> {
        self.client.delete_repository_webhook(id).await.map_err(agentsmesh_services::wire)?;
        Ok(())
    }

    pub async fn get_webhook_status(&self, id: i64) -> Result<String, String> {
        let resp = self.client
            .get_repository_webhook_status(id)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn get_webhook_secret(&self, id: i64) -> Result<String, String> {
        let resp = self.client
            .get_repository_webhook_secret(id)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn list_merge_requests(
        &self, id: i64, branch: Option<String>, state: Option<String>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_repository_merge_requests(id, branch.as_deref(), state.as_deref())
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn mark_webhook_configured(&self, id: i64) -> Result<(), String> {
        let path = self.client.org_path(&format!("/repositories/{id}/webhook/configured"));
        self.client
            .post::<agentsmesh_types::EmptyResponse>(&path, &serde_json::json!({}))
            .await.map_err(agentsmesh_services::wire)?;
        Ok(())
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Each `*_connect` method takes prost-encoded bytes (Uint8Array on the JS
    // side) and returns prost-encoded bytes — TS callers encode via
    // @bufbuild/protobuf .toBinary() and decode via .fromBinary().

    pub async fn list_repositories_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().list_repositories_connect(request_bytes).await
    }

    pub async fn get_repository_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().get_repository_connect(request_bytes).await
    }

    pub async fn create_repository_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().create_repository_connect(request_bytes).await
    }

    pub async fn update_repository_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().update_repository_connect(request_bytes).await
    }

    pub async fn delete_repository_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().delete_repository_connect(request_bytes).await
    }

    pub async fn list_repository_branches_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().list_repository_branches_connect(request_bytes).await
    }

    pub async fn sync_repository_branches_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().sync_repository_branches_connect(request_bytes).await
    }

    pub async fn list_repository_merge_requests_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().list_repository_merge_requests_connect(request_bytes).await
    }

    pub async fn register_repository_webhook_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().register_repository_webhook_connect(request_bytes).await
    }

    pub async fn delete_repository_webhook_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().delete_repository_webhook_connect(request_bytes).await
    }

    pub async fn get_repository_webhook_status_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().get_repository_webhook_status_connect(request_bytes).await
    }

    pub async fn get_repository_webhook_secret_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().get_repository_webhook_secret_connect(request_bytes).await
    }

    pub async fn mark_repository_webhook_configured_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.svc().mark_repository_webhook_configured_connect(request_bytes).await
    }

    // Lazily construct a services::RepositoryService for the Connect bridge.
    // The legacy JSON methods still call self.client directly (no state, no
    // bookkeeping) so wrapping them is unnecessary for the dual-track window.
    fn svc(&self) -> agentsmesh_services::RepositoryService {
        agentsmesh_services::RepositoryService::new(self.client.clone())
    }
}
