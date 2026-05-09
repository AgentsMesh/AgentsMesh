use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::*;

impl ApiClient {
    pub async fn list_runners(
        &self,
        status: Option<&str>,
    ) -> Result<RunnerListResponse, ApiError> {
        let mut path = self.org_path("/runners");
        if let Some(s) = status {
            path = format!("{path}?status={s}");
        }
        self.get(&path).await
    }

    pub async fn list_available_runners(&self) -> Result<RunnerListResponse, ApiError> {
        self.get(&self.org_path("/runners/available")).await
    }

    pub async fn get_runner(&self, id: i64) -> Result<RunnerDetailResponse, ApiError> {
        self.get(&self.org_path(&format!("/runners/{id}"))).await
    }

    pub async fn update_runner(
        &self,
        id: i64,
        data: &UpdateRunnerRequest,
    ) -> Result<Runner, ApiError> {
        self.put_resource(&self.org_path(&format!("/runners/{id}")), data, "runner").await
    }

    pub async fn delete_runner(&self, id: i64) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!("/runners/{id}"))).await
    }

    pub async fn create_runner_token(
        &self,
        data: &CreateRunnerTokenRequest,
    ) -> Result<GRPCRegistrationToken, ApiError> {
        self.post(&self.org_path("/runners/grpc/tokens"), data)
            .await
    }

    pub async fn list_runner_tokens(&self) -> Result<RunnerTokenListResponse, ApiError> {
        self.get(&self.org_path("/runners/grpc/tokens")).await
    }

    pub async fn delete_runner_token(&self, id: i64) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!("/runners/grpc/tokens/{id}")))
            .await
    }

    pub async fn list_runner_pods(
        &self,
        id: i64,
        status: Option<&str>,
        limit: Option<u32>,
        offset: Option<u32>,
    ) -> Result<PodListResponse, ApiError> {
        let mut path = self.org_path(&format!("/runners/{id}/pods"));
        let mut params = Vec::new();
        if let Some(s) = status {
            params.push(format!("status={s}"));
        }
        if let Some(l) = limit {
            params.push(format!("limit={l}"));
        }
        if let Some(o) = offset {
            params.push(format!("offset={o}"));
        }
        if !params.is_empty() {
            path = format!("{path}?{}", params.join("&"));
        }
        self.get(&path).await
    }

    pub async fn query_runner_sandboxes(
        &self,
        id: i64,
        data: &SandboxQueryRequest,
    ) -> Result<SandboxQueryResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/runners/{id}/sandboxes/query")),
            data,
        )
        .await
    }

    pub async fn upgrade_runner(
        &self,
        id: i64,
        data: &UpgradeRunnerRequest,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/runners/{id}/upgrade")),
            data,
        )
        .await
    }

    pub async fn request_runner_log_upload(
        &self,
        id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/runners/{id}/logs/upload")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn list_runner_logs(&self, id: i64) -> Result<RunnerLogListResponse, ApiError> {
        self.get(&self.org_path(&format!("/runners/{id}/logs")))
            .await
    }

    pub async fn get_runner_auth_status(
        &self,
        auth_key: &str,
    ) -> Result<RunnerAuthStatus, ApiError> {
        self.get(&self.org_path(&format!("/runners/auth/{auth_key}")))
            .await
    }

    pub async fn authorize_runner(
        &self,
        data: &AuthorizeRunnerRequest,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(&self.org_path("/runners/grpc/authorize"), data).await
    }
}
