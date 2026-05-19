use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmUserCredentialService {
    inner: agentsmesh_services::UserCredentialService,
}

#[wasm_bindgen]
impl WasmUserCredentialService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self { inner: agentsmesh_services::UserCredentialService::new(client) }
    }

    // -------- Connect-RPC (binary wire) — see proto-naming-conventions.md §2.5
    //
    // TS encodes the request via @bufbuild/protobuf .toBinary(), passes the
    // Uint8Array in, receives a Uint8Array back, decodes via .fromBinary().
    // No JSON intermediate.

    // UserGitCredentialService
    #[wasm_bindgen(js_name = listGitCredentialsConnect)]
    pub async fn list_git_credentials_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.list_git_credentials_connect(request).await
    }
    #[wasm_bindgen(js_name = getGitCredentialConnect)]
    pub async fn get_git_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.get_git_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = createGitCredentialConnect)]
    pub async fn create_git_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.create_git_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = updateGitCredentialConnect)]
    pub async fn update_git_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.update_git_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = deleteGitCredentialConnect)]
    pub async fn delete_git_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.delete_git_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = getDefaultGitCredentialConnect)]
    pub async fn get_default_git_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.get_default_git_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = setDefaultGitCredentialConnect)]
    pub async fn set_default_git_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.set_default_git_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = clearDefaultGitCredentialConnect)]
    pub async fn clear_default_git_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.clear_default_git_credential_connect(request).await
    }

    // UserAgentCredentialService
    #[wasm_bindgen(js_name = listAgentCredentialsConnect)]
    pub async fn list_agent_credentials_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.list_agent_credentials_connect(request).await
    }
    #[wasm_bindgen(js_name = listAgentCredentialsForAgentConnect)]
    pub async fn list_agent_credentials_for_agent_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.list_agent_credentials_for_agent_connect(request).await
    }
    #[wasm_bindgen(js_name = getAgentCredentialConnect)]
    pub async fn get_agent_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.get_agent_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = createAgentCredentialConnect)]
    pub async fn create_agent_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.create_agent_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = updateAgentCredentialConnect)]
    pub async fn update_agent_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.update_agent_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = deleteAgentCredentialConnect)]
    pub async fn delete_agent_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.delete_agent_credential_connect(request).await
    }
    #[wasm_bindgen(js_name = setDefaultAgentCredentialConnect)]
    pub async fn set_default_agent_credential_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.set_default_agent_credential_connect(request).await
    }

    // UserRepositoryProviderService
    #[wasm_bindgen(js_name = listRepositoryProvidersConnect)]
    pub async fn list_repository_providers_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.list_repository_providers_connect(request).await
    }
    #[wasm_bindgen(js_name = getRepositoryProviderConnect)]
    pub async fn get_repository_provider_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.get_repository_provider_connect(request).await
    }
    #[wasm_bindgen(js_name = createRepositoryProviderConnect)]
    pub async fn create_repository_provider_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.create_repository_provider_connect(request).await
    }
    #[wasm_bindgen(js_name = updateRepositoryProviderConnect)]
    pub async fn update_repository_provider_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.update_repository_provider_connect(request).await
    }
    #[wasm_bindgen(js_name = deleteRepositoryProviderConnect)]
    pub async fn delete_repository_provider_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.delete_repository_provider_connect(request).await
    }
    #[wasm_bindgen(js_name = setDefaultRepositoryProviderConnect)]
    pub async fn set_default_repository_provider_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.set_default_repository_provider_connect(request).await
    }
    #[wasm_bindgen(js_name = testRepositoryProviderConnectionConnect)]
    pub async fn test_repository_provider_connection_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.test_repository_provider_connection_connect(request).await
    }
    #[wasm_bindgen(js_name = listProviderRepositoriesConnect)]
    pub async fn list_provider_repositories_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.inner.list_provider_repositories_connect(request).await
    }
}
