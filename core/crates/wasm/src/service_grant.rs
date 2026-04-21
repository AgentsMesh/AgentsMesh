use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::GrantService;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmGrantService(pub(crate) GrantService);

#[wasm_bindgen]
impl WasmGrantService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self(GrantService::new(client))
    }

    pub async fn list(&self, resource_type: String, resource_id: String) -> Result<String, String> {
        self.0.list(&resource_type, &resource_id).await
    }

    pub async fn grant(
        &self,
        resource_type: String,
        resource_id: String,
        user_id: i64,
    ) -> Result<String, String> {
        self.0.grant(&resource_type, &resource_id, user_id).await
    }

    pub async fn revoke(
        &self,
        resource_type: String,
        resource_id: String,
        grant_id: i64,
    ) -> Result<(), String> {
        self.0.revoke(&resource_type, &resource_id, grant_id).await
    }
}
