use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::RepositoryService;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmRepositoryService {
    svc: RepositoryService,
}

#[wasm_bindgen]
impl WasmRepositoryService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self { svc: RepositoryService::new(client) }
    }

    // -------- Connect-RPC (binary wire) --------

    #[wasm_bindgen(js_name = listRepositoriesConnect)]
    pub async fn list_repositories_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.list_repositories_connect(request).await
    }
    #[wasm_bindgen(js_name = getRepositoryConnect)]
    pub async fn get_repository_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.get_repository_connect(request).await
    }
    #[wasm_bindgen(js_name = createRepositoryConnect)]
    pub async fn create_repository_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.create_repository_connect(request).await
    }
    #[wasm_bindgen(js_name = updateRepositoryConnect)]
    pub async fn update_repository_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.update_repository_connect(request).await
    }
    #[wasm_bindgen(js_name = deleteRepositoryConnect)]
    pub async fn delete_repository_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.delete_repository_connect(request).await
    }
    #[wasm_bindgen(js_name = listRepositoryBranchesConnect)]
    pub async fn list_repository_branches_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.list_repository_branches_connect(request).await
    }
    #[wasm_bindgen(js_name = syncRepositoryBranchesConnect)]
    pub async fn sync_repository_branches_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.sync_repository_branches_connect(request).await
    }
    #[wasm_bindgen(js_name = listRepositoryMergeRequestsConnect)]
    pub async fn list_repository_merge_requests_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.list_repository_merge_requests_connect(request).await
    }
    #[wasm_bindgen(js_name = registerRepositoryWebhookConnect)]
    pub async fn register_repository_webhook_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.register_repository_webhook_connect(request).await
    }
    #[wasm_bindgen(js_name = deleteRepositoryWebhookConnect)]
    pub async fn delete_repository_webhook_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.delete_repository_webhook_connect(request).await
    }
    #[wasm_bindgen(js_name = getRepositoryWebhookStatusConnect)]
    pub async fn get_repository_webhook_status_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.get_repository_webhook_status_connect(request).await
    }
    #[wasm_bindgen(js_name = getRepositoryWebhookSecretConnect)]
    pub async fn get_repository_webhook_secret_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.get_repository_webhook_secret_connect(request).await
    }
    #[wasm_bindgen(js_name = markRepositoryWebhookConfiguredConnect)]
    pub async fn mark_repository_webhook_configured_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.svc.mark_repository_webhook_configured_connect(request).await
    }
}
