use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::SSOService;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmSSOService(pub(crate) SSOService);

#[wasm_bindgen]
impl WasmSSOService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self(SSOService::new(client))
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // TS encodes the request via @bufbuild/protobuf .toBinary(), passes the
    // Uint8Array in, receives a Uint8Array back, decodes via .fromBinary().
    // No JSON intermediate; conventions §2.5 forbids it on the client.
    //
    // js_name is camelCase to match the existing JS-side conventions; the
    // `_connect` suffix marks the migration lane so the legacy JSON methods
    // can coexist until all 26 services flip.

    #[wasm_bindgen(js_name = discoverConnect)]
    pub async fn discover_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.discover_connect(request).await
    }

    #[wasm_bindgen(js_name = ldapAuthConnect)]
    pub async fn ldap_auth_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.ldap_auth_connect(request).await
    }

    // -------- Legacy REST JSON methods (preserved during dual-track) --------

    pub async fn discover(&self, email: &str) -> Result<String, String> {
        self.0.discover(email).await
    }

    pub async fn ldap_auth(&self, domain: &str, json: &str) -> Result<String, String> {
        self.0.ldap_auth(domain, json).await
    }
}
