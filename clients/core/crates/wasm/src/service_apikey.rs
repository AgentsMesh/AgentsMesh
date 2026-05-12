use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::ApiKeyService;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmApiKeyService(pub(crate) ApiKeyService);

#[wasm_bindgen]
impl WasmApiKeyService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self(ApiKeyService::new(client))
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // TS encodes the request via @bufbuild/protobuf .toBinary(), passes the
    // Uint8Array in, receives a Uint8Array back, decodes via .fromBinary().
    // No JSON intermediate; conventions §2.5 forbids it on the client.
    //
    // js_name is camelCase to match JS conventions; the `_connect` suffix
    // marks the migration lane so the legacy JSON methods can coexist until
    // all 26 services flip.

    #[wasm_bindgen(js_name = listApiKeysConnect)]
    pub async fn list_api_keys_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_api_keys_connect(request).await
    }

    #[wasm_bindgen(js_name = getApiKeyConnect)]
    pub async fn get_api_key_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_api_key_connect(request).await
    }

    #[wasm_bindgen(js_name = createApiKeyConnect)]
    pub async fn create_api_key_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.create_api_key_connect(request).await
    }

    #[wasm_bindgen(js_name = updateApiKeyConnect)]
    pub async fn update_api_key_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.update_api_key_connect(request).await
    }

    #[wasm_bindgen(js_name = revokeApiKeyConnect)]
    pub async fn revoke_api_key_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.revoke_api_key_connect(request).await
    }

    #[wasm_bindgen(js_name = deleteApiKeyConnect)]
    pub async fn delete_api_key_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.delete_api_key_connect(request).await
    }

    // -------- Legacy REST JSON methods (preserved during dual-track) --------

    pub async fn list(&self) -> Result<String, String> {
        self.0.list().await
    }

    pub async fn get(&self, id: i64) -> Result<String, String> {
        self.0.get(id).await
    }

    pub async fn create(&self, json: &str) -> Result<String, String> {
        self.0.create(json).await
    }

    pub async fn update(&self, id: i64, json: &str) -> Result<String, String> {
        self.0.update(id, json).await
    }

    pub async fn delete(&self, id: i64) -> Result<(), String> {
        self.0.delete(id).await
    }

    pub async fn revoke(&self, id: i64) -> Result<(), String> {
        self.0.revoke(id).await
    }
}
