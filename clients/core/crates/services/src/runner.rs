use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_runner_api_v1 as runner_proto;
use prost::Message;

// Networking-only service for the runner domain. The runner cache lives in the
// shared `AppState.runners` (dispatch-hook SSOT), reached via the wasm/napi
// `app_runner*` surface — this service speaks only the Connect-RPC wire
// (including the registration-bootstrap auth RPCs).
pub struct RunnerService {
    client: Arc<ApiClient>,
}

impl RunnerService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn get_auth_status_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        // No proto coverage on the wire for /runners/auth/<key> per se, but
        // GetRunnerAuthStatus is a Connect RPC — same path the legacy JSON
        // helper used, just expressed as wire-aligned proto bytes.
        let req = runner_proto::GetRunnerAuthStatusRequest::decode(request_bytes)
            .map_err(|e| format!("decode GetRunnerAuthStatusRequest: {e}"))?;
        tracing::debug!(target: "runner", "get runner auth status");
        let resp = self.client
            .get_runner_auth_status_connect(&req)
            .await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn authorize_runner_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        // Same reason as get_auth_status_connect — registration bootstrap.
        let mut req = runner_proto::AuthorizeRunnerRequest::decode(request_bytes)
            .map_err(|e| format!("decode AuthorizeRunnerRequest: {e}"))?;
        // The renderer can't always populate org_slug (registration happens
        // before the session knows which org the runner will land in). Fill
        // it from the session here for parity with the legacy helper.
        if req.org_slug.is_empty() {
            req.org_slug = self.client.current_org_slug();
        }
        tracing::info!(target: "runner", org_slug = %req.org_slug, node_id = %req.node_id, "authorize runner");
        let resp = self.client
            .authorize_runner_connect(&req)
            .await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Each `*_connect` method takes prost-encoded bytes and returns prost-encoded
    // bytes — matching the wasm bridge's `Result<Vec<u8>, String>` surface
    // (conventions §2.5). Caller (TS) encodes via @bufbuild/protobuf .toBinary()
    // and decodes via .fromBinary().

    pub async fn list_runners_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::ListRunnersRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_runners request: {e}"))?;
        tracing::debug!(target: "runner", org_slug = %req.org_slug, "list runners");
        let resp = self.client.list_runners_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_available_runners_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::ListAvailableRunnersRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_available_runners request: {e}"))?;
        tracing::debug!(target: "runner", org_slug = %req.org_slug, "list available runners");
        let resp = self.client.list_available_runners_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_runner_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::GetRunnerRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_runner request: {e}"))?;
        tracing::debug!(target: "runner", org_slug = %req.org_slug, runner_id = req.id, "get runner");
        let resp = self.client.get_runner_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_runner_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::UpdateRunnerRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_runner request: {e}"))?;
        tracing::info!(target: "runner", org_slug = %req.org_slug, runner_id = req.id, "update runner");
        let resp = self.client.update_runner_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_runner_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::DeleteRunnerRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_runner request: {e}"))?;
        tracing::info!(target: "runner", org_slug = %req.org_slug, runner_id = req.id, "delete runner");
        let resp = self.client.delete_runner_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn upgrade_runner_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::UpgradeRunnerRequest::decode(request_bytes)
            .map_err(|e| format!("decode upgrade_runner request: {e}"))?;
        tracing::info!(target: "runner", org_slug = %req.org_slug, runner_id = req.id, target_version = %req.target_version, "upgrade runner");
        let resp = self.client.upgrade_runner_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn request_log_upload_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::RequestLogUploadRequest::decode(request_bytes)
            .map_err(|e| format!("decode request_log_upload request: {e}"))?;
        tracing::info!(target: "runner", org_slug = %req.org_slug, runner_id = req.id, "request log upload");
        let resp = self.client.request_log_upload_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_runner_logs_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::ListRunnerLogsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_runner_logs request: {e}"))?;
        tracing::debug!(target: "runner", org_slug = %req.org_slug, runner_id = req.id, "list runner logs");
        let resp = self.client.list_runner_logs_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn query_sandboxes_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::QuerySandboxesRequest::decode(request_bytes)
            .map_err(|e| format!("decode query_sandboxes request: {e}"))?;
        tracing::debug!(target: "runner", org_slug = %req.org_slug, runner_id = req.id, "query sandboxes");
        let resp = self.client.query_sandboxes_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_runner_token_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::CreateRunnerTokenRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_runner_token request: {e}"))?;
        tracing::info!(target: "runner", org_slug = %req.org_slug, "create runner token");
        let resp = self.client.create_runner_token_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_runner_tokens_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::ListRunnerTokensRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_runner_tokens request: {e}"))?;
        tracing::debug!(target: "runner", org_slug = %req.org_slug, "list runner tokens");
        let resp = self.client.list_runner_tokens_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_runner_token_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = runner_proto::DeleteRunnerTokenRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_runner_token request: {e}"))?;
        tracing::info!(target: "runner", org_slug = %req.org_slug, token_id = req.id, "delete runner token");
        let resp = self.client.delete_runner_token_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
