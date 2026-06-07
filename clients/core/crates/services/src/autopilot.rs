use std::sync::Arc;

use agentsmesh_api_client::ApiClient;

pub struct AutopilotService {
    client: Arc<ApiClient>,
}

impl AutopilotService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Connect lanes — request bytes in, response bytes out. The autopilot
    // controller cache is the AppState SSOT (runtime.state.autopilot), fed via
    // the dispatch hook + the app_autopilot_* napi/wasm surface; this service
    // is networking-only.

    pub async fn list_autopilots_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_autopilot_v1 as ap;
        use prost::Message;
        let req = ap::ListAutopilotControllersRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_autopilots request: {e}"))?;
        tracing::debug!(target: "autopilot", org_slug = %req.org_slug, "list autopilots");
        let resp = self.client.list_autopilots_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_autopilot_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_autopilot_v1 as ap;
        use prost::Message;
        let req = ap::GetAutopilotControllerRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_autopilot request: {e}"))?;
        tracing::debug!(target: "autopilot", org_slug = %req.org_slug, key = %req.key, "get autopilot");
        let resp = self.client.get_autopilot_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_autopilot_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_autopilot_v1 as ap;
        use prost::Message;
        let req = ap::CreateAutopilotControllerRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_autopilot request: {e}"))?;
        tracing::info!(target: "autopilot", org_slug = %req.org_slug, pod_key = %req.pod_key, "create autopilot");
        let resp = self.client.create_autopilot_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn action_autopilot_connect(
        &self, procedure: &str, request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_autopilot_v1 as ap;
        use prost::Message;
        let req = ap::ActionRequest::decode(request_bytes)
            .map_err(|e| format!("decode action_autopilot request: {e}"))?;
        tracing::info!(target: "autopilot", org_slug = %req.org_slug, key = %req.key, procedure, "action autopilot");
        let resp = match procedure {
            "pause" => self.client.pause_autopilot_connect(&req).await,
            "resume" => self.client.resume_autopilot_connect(&req).await,
            "stop" => self.client.stop_autopilot_connect(&req).await,
            "takeover" => self.client.takeover_autopilot_connect(&req).await,
            "handback" => self.client.handback_autopilot_connect(&req).await,
            other => return Err(format!("unknown autopilot action: {other}")),
        }.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn approve_autopilot_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_autopilot_v1 as ap;
        use prost::Message;
        let req = ap::ApproveRequest::decode(request_bytes)
            .map_err(|e| format!("decode approve_autopilot request: {e}"))?;
        tracing::info!(target: "autopilot", org_slug = %req.org_slug, key = %req.key, "approve autopilot");
        let resp = self.client.approve_autopilot_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_iterations_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        use agentsmesh_types::proto_autopilot_v1 as ap;
        use prost::Message;
        let req = ap::GetIterationsRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_iterations request: {e}"))?;
        tracing::debug!(target: "autopilot", org_slug = %req.org_slug, key = %req.key, "get iterations");
        let resp = self.client.get_autopilot_iterations_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
