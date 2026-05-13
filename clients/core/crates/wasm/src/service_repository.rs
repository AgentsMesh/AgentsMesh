use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
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

    // Legacy JSON-shaped methods routed through the Rust `RepositoryService`
    // (Connect-RPC under the hood) so the web/iOS/desktop bridge keeps the
    // legacy wire shape during the dual-track window.

    pub async fn list(&self) -> Result<String, String> {
        self.svc().list().await
    }

    pub async fn get(&self, id: i64) -> Result<String, String> {
        self.svc().get(id).await
    }

    pub async fn create(&self, json: &str) -> Result<String, String> {
        self.svc().create(json).await
    }

    pub async fn update(&self, id: i64, json: &str) -> Result<String, String> {
        self.svc().update(id, json).await
    }

    pub async fn delete(&self, id: i64) -> Result<(), String> {
        self.svc().delete(id).await
    }

    pub async fn list_branches(&self, id: i64) -> Result<String, String> {
        self.svc().list_branches(id).await
    }

    pub async fn sync_branches(&self, id: i64, json: &str) -> Result<String, String> {
        self.svc().sync_branches(id, json).await
    }

    pub async fn register_webhook(&self, id: i64) -> Result<(), String> {
        self.svc().register_webhook(id).await
    }

    pub async fn delete_webhook(&self, id: i64) -> Result<(), String> {
        self.svc().delete_webhook(id).await
    }

    pub async fn get_webhook_status(&self, id: i64) -> Result<String, String> {
        self.svc().get_webhook_status(id).await
    }

    pub async fn get_webhook_secret(&self, id: i64) -> Result<String, String> {
        self.svc().get_webhook_secret(id).await
    }

    pub async fn list_merge_requests(
        &self, id: i64, branch: Option<String>, state: Option<String>,
    ) -> Result<String, String> {
        self.svc().list_merge_requests(id, branch, state).await
    }

    pub async fn mark_webhook_configured(&self, id: i64) -> Result<(), String> {
        self.svc().mark_webhook_configured(id).await
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

    fn svc(&self) -> agentsmesh_services::RepositoryService {
        agentsmesh_services::RepositoryService::new(self.client.clone())
    }
}
