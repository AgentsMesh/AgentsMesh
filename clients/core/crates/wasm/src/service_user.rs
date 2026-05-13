use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::UserApiService;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmUserApiService {
    inner: UserApiService,
}

#[wasm_bindgen]
impl WasmUserApiService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self { inner: UserApiService::new(client) }
    }

    // -------- Connect-RPC (binary wire) --------

    #[wasm_bindgen(js_name = getMeConnect)]
    pub async fn get_me_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.get_me_connect(request).await
    }

    #[wasm_bindgen(js_name = updateMeConnect)]
    pub async fn update_me_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.update_me_connect(request).await
    }

    #[wasm_bindgen(js_name = changePasswordConnect)]
    pub async fn change_password_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.change_password_connect(request).await
    }

    #[wasm_bindgen(js_name = listIdentitiesConnect)]
    pub async fn list_identities_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.list_identities_connect(request).await
    }

    #[wasm_bindgen(js_name = deleteIdentityConnect)]
    pub async fn delete_identity_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.delete_identity_connect(request).await
    }

    #[wasm_bindgen(js_name = searchUsersConnect)]
    pub async fn search_users_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.search_users_connect(request).await
    }
}
