use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::FileService;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmFileService(pub(crate) FileService);

#[wasm_bindgen]
impl WasmFileService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self(FileService::new(client))
    }

    pub async fn presign_upload(&self, json: &str) -> Result<String, String> {
        self.0.presign_upload(json).await
    }

    pub async fn upload_file(
        &self, file_data: js_sys::Uint8Array, filename: &str, content_type: &str,
    ) -> Result<String, String> {
        let bytes = file_data.to_vec();
        self.0.upload_file(bytes, filename, content_type).await
    }

    // -------- Connect-RPC (binary wire) --------

    #[wasm_bindgen(js_name = presignUploadConnect)]
    pub async fn presign_upload_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.presign_upload_connect(request).await
    }
}
