use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_loop_v1 as lp;
use prost::Message;

pub struct LoopService {
    client: Arc<ApiClient>,
}

impl LoopService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // The loop + run cache is the AppState SSOT (runtime.state.loops), fed by
    // the LoopRun* dispatch arms + the app_loop_* napi/wasm surface; this
    // service is networking-only.

    pub async fn list_loops_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = lp::ListLoopsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_loops request: {e}"))?;
        tracing::debug!(target: "loop", org_slug = %req.org_slug, "list loops");
        let resp = self.client.list_loops_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = lp::GetLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_loop request: {e}"))?;
        tracing::debug!(target: "loop", org_slug = %req.org_slug, loop_slug = %req.loop_slug, "get loop");
        let resp = self.client.get_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = lp::CreateLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_loop request: {e}"))?;
        tracing::info!(target: "loop", org_slug = %req.org_slug, slug = %req.slug, agent_slug = %req.agent_slug, "create loop");
        let resp = self.client.create_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = lp::UpdateLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_loop request: {e}"))?;
        tracing::info!(target: "loop", org_slug = %req.org_slug, loop_slug = %req.loop_slug, "update loop");
        let resp = self.client.update_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = lp::DeleteLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_loop request: {e}"))?;
        tracing::info!(target: "loop", org_slug = %req.org_slug, loop_slug = %req.loop_slug, "delete loop");
        let resp = self.client.delete_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn loop_action_connect(
        &self, action: &str, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = lp::LoopActionRequest::decode(request_bytes)
            .map_err(|e| format!("decode loop_action request: {e}"))?;
        tracing::info!(target: "loop", org_slug = %req.org_slug, loop_slug = %req.loop_slug, action, "loop action");
        let resp = match action {
            "enable" => self.client.enable_loop_connect(&req).await,
            "disable" => self.client.disable_loop_connect(&req).await,
            other => return Err(format!("unknown loop action: {other}")),
        }.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn trigger_loop_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = lp::TriggerLoopRequest::decode(request_bytes)
            .map_err(|e| format!("decode trigger_loop request: {e}"))?;
        tracing::info!(target: "loop", org_slug = %req.org_slug, loop_slug = %req.loop_slug, "trigger loop");
        let resp = self.client.trigger_loop_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_runs_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = lp::ListRunsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_runs request: {e}"))?;
        tracing::debug!(target: "loop", org_slug = %req.org_slug, loop_slug = %req.loop_slug, "list runs");
        let resp = self.client.list_loop_runs_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn cancel_run_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = lp::CancelRunRequest::decode(request_bytes)
            .map_err(|e| format!("decode cancel_run request: {e}"))?;
        tracing::info!(target: "loop", org_slug = %req.org_slug, loop_slug = %req.loop_slug, run_id = req.run_id, "cancel run");
        let resp = self.client.cancel_loop_run_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
