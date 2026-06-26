use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_services::RunnerService;
use wasm_bindgen::prelude::*;

// Networking-only wasm handle for the runner domain. The runner cache lives in
// the shared `AppState.runners` (reached via `WasmRunnerState`); this service
// exposes only the Connect-RPC surface (+ registration-bootstrap auth RPCs).
#[wasm_bindgen]
pub struct WasmRunnerService(pub(crate) RunnerService);

#[wasm_bindgen]
impl WasmRunnerService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self(RunnerService::new(client))
    }

    pub async fn get_auth_status(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_auth_status_connect(request_bytes).await
    }

    pub async fn authorize_runner(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        self.0.authorize_runner_connect(request_bytes).await
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // TS encodes the request via @bufbuild/protobuf .toBinary(), passes the
    // Uint8Array in, receives a Uint8Array back, decodes via .fromBinary().

    #[wasm_bindgen(js_name = listRunnersConnect)]
    pub async fn list_runners_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_runners_connect(request).await
    }

    #[wasm_bindgen(js_name = listAvailableRunnersConnect)]
    pub async fn list_available_runners_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_available_runners_connect(request).await
    }

    #[wasm_bindgen(js_name = getRunnerConnect)]
    pub async fn get_runner_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.get_runner_connect(request).await
    }

    #[wasm_bindgen(js_name = updateRunnerConnect)]
    pub async fn update_runner_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.update_runner_connect(request).await
    }

    #[wasm_bindgen(js_name = deleteRunnerConnect)]
    pub async fn delete_runner_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.delete_runner_connect(request).await
    }

    #[wasm_bindgen(js_name = upgradeRunnerConnect)]
    pub async fn upgrade_runner_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.upgrade_runner_connect(request).await
    }

    #[wasm_bindgen(js_name = upgradeAgentConnect)]
    pub async fn upgrade_agent_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.upgrade_agent_connect(request).await
    }

    #[wasm_bindgen(js_name = requestLogUploadConnect)]
    pub async fn request_log_upload_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.request_log_upload_connect(request).await
    }

    #[wasm_bindgen(js_name = listRunnerLogsConnect)]
    pub async fn list_runner_logs_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_runner_logs_connect(request).await
    }

    #[wasm_bindgen(js_name = querySandboxesConnect)]
    pub async fn query_sandboxes_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.query_sandboxes_connect(request).await
    }

    #[wasm_bindgen(js_name = createRunnerTokenConnect)]
    pub async fn create_runner_token_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.create_runner_token_connect(request).await
    }

    #[wasm_bindgen(js_name = listRunnerTokensConnect)]
    pub async fn list_runner_tokens_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.list_runner_tokens_connect(request).await
    }

    #[wasm_bindgen(js_name = deleteRunnerTokenConnect)]
    pub async fn delete_runner_token_connect(&self, request: &[u8]) -> Result<Vec<u8>, String> {
        self.0.delete_runner_token_connect(request).await
    }
}
