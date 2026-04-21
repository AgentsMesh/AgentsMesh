use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::*;

impl ApiClient {
    pub async fn list_autopilots(&self) -> Result<AutopilotListResponse, ApiError> {
        let list: Vec<AutopilotController> =
            self.get(&self.org_path("/autopilot-controllers")).await?;
        Ok(AutopilotListResponse { controllers: list })
    }

    pub async fn get_autopilot(&self, key: &str) -> Result<AutopilotController, ApiError> {
        self.get(&self.org_path(&format!("/autopilot-controllers/{key}")))
            .await
    }

    pub async fn create_autopilot(
        &self,
        data: &CreateAutopilotRequest,
    ) -> Result<AutopilotController, ApiError> {
        self.post_resource(&self.org_path("/autopilot-controllers"), data, "controller").await
    }

    pub async fn pause_autopilot(&self, key: &str) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/autopilot-controllers/{key}/pause")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn resume_autopilot(&self, key: &str) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/autopilot-controllers/{key}/resume")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn stop_autopilot(&self, key: &str) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/autopilot-controllers/{key}/stop")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn approve_autopilot(
        &self,
        key: &str,
        data: &ApproveAutopilotRequest,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/autopilot-controllers/{key}/approve")),
            data,
        )
        .await
    }

    pub async fn takeover_autopilot(&self, key: &str) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/autopilot-controllers/{key}/takeover")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn handback_autopilot(&self, key: &str) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/autopilot-controllers/{key}/handback")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn get_autopilot_iterations(
        &self,
        key: &str,
    ) -> Result<AutopilotIterationListResponse, ApiError> {
        self.get(&self.org_path(&format!(
            "/autopilot-controllers/{key}/iterations"
        )))
        .await
    }
}
