use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::*;
use agentsmesh_types::proto_runner_api_v1 as runner_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// These methods call the Connect handlers in
// backend/internal/api/connect/runner/. Procedure paths derive from
// `proto.runner_api.v1.RunnerService.<Method>` (conventions §12).

impl ApiClient {
    pub async fn list_runners_connect(
        &self,
        req: &runner_proto::ListRunnersRequest,
    ) -> Result<runner_proto::ListRunnersResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/ListRunners",
            req,
        )
        .await
    }

    pub async fn list_available_runners_connect(
        &self,
        req: &runner_proto::ListAvailableRunnersRequest,
    ) -> Result<runner_proto::ListAvailableRunnersResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/ListAvailableRunners",
            req,
        )
        .await
    }

    pub async fn get_runner_connect(
        &self,
        req: &runner_proto::GetRunnerRequest,
    ) -> Result<runner_proto::GetRunnerResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/GetRunner",
            req,
        )
        .await
    }

    pub async fn update_runner_connect(
        &self,
        req: &runner_proto::UpdateRunnerRequest,
    ) -> Result<runner_proto::Runner, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/UpdateRunner",
            req,
        )
        .await
    }

    pub async fn delete_runner_connect(
        &self,
        req: &runner_proto::DeleteRunnerRequest,
    ) -> Result<runner_proto::DeleteRunnerResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/DeleteRunner",
            req,
        )
        .await
    }

    pub async fn upgrade_runner_connect(
        &self,
        req: &runner_proto::UpgradeRunnerRequest,
    ) -> Result<runner_proto::UpgradeRunnerResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/UpgradeRunner",
            req,
        )
        .await
    }

    pub async fn upgrade_agent_connect(
        &self,
        req: &runner_proto::UpgradeAgentRequest,
    ) -> Result<runner_proto::UpgradeAgentResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/UpgradeAgent",
            req,
        )
        .await
    }

    pub async fn request_log_upload_connect(
        &self,
        req: &runner_proto::RequestLogUploadRequest,
    ) -> Result<runner_proto::RequestLogUploadResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/RequestLogUpload",
            req,
        )
        .await
    }

    pub async fn list_runner_logs_connect(
        &self,
        req: &runner_proto::ListRunnerLogsRequest,
    ) -> Result<runner_proto::ListRunnerLogsResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/ListRunnerLogs",
            req,
        )
        .await
    }

    pub async fn query_sandboxes_connect(
        &self,
        req: &runner_proto::QuerySandboxesRequest,
    ) -> Result<runner_proto::QuerySandboxesResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/QuerySandboxes",
            req,
        )
        .await
    }

    pub async fn create_runner_token_connect(
        &self,
        req: &runner_proto::CreateRunnerTokenRequest,
    ) -> Result<runner_proto::RunnerToken, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/CreateRunnerToken",
            req,
        )
        .await
    }

    pub async fn list_runner_tokens_connect(
        &self,
        req: &runner_proto::ListRunnerTokensRequest,
    ) -> Result<runner_proto::ListRunnerTokensResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/ListRunnerTokens",
            req,
        )
        .await
    }

    pub async fn delete_runner_token_connect(
        &self,
        req: &runner_proto::DeleteRunnerTokenRequest,
    ) -> Result<runner_proto::DeleteRunnerTokenResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/DeleteRunnerToken",
            req,
        )
        .await
    }
}

// =============================================================================
// Legacy REST — carve-outs without proto coverage.
// =============================================================================
//
// Each method below has no Connect-RPC equivalent and is preserved as REST:
//   - list_runner_pods: proto.pod.v1.ListPods has no runner_id filter yet;
//     runner-detail view (web + desktop) still drives the per-runner list
//     via this REST path.
//   - get_runner_auth_status / authorize_runner: registration bootstrap
//     (Tailscale-style interactive auth) lives outside the org-scoped
//     RunnerService — REST stays until the runner-mgmt RPCs land.

impl ApiClient {
    /// Lists pods owned by a specific runner. Implemented on top of
    /// `proto.pod.v1.PodService.ListPods` with the `runner_id` filter —
    /// the dedicated REST endpoint was removed.
    pub async fn list_runner_pods(
        &self,
        id: i64,
        status: Option<&str>,
        limit: Option<u32>,
        offset: Option<u32>,
    ) -> Result<agentsmesh_types::proto_pod_v1::ListPodsResponse, ApiError> {
        let req = agentsmesh_types::proto_pod_v1::ListPodsRequest {
            org_slug: self.current_org_slug(),
            status: status.map(|s| s.to_string()),
            created_by_id: None,
            limit: limit.map(|v| v as i32),
            offset: offset.map(|v| v as i32),
            runner_id: Some(id),
        };
        self.list_pods_connect(&req).await
    }

    pub async fn get_runner_auth_status(
        &self,
        auth_key: &str,
    ) -> Result<RunnerAuthStatus, ApiError> {
        let req = runner_proto::GetRunnerAuthStatusRequest {
            auth_key: auth_key.to_string(),
        };
        let resp: runner_proto::RunnerAuthStatus = connect_call(
            self,
            "/proto.runner_api.v1.RunnerPublicService/GetRunnerAuthStatus",
            &req,
        )
        .await?;
        Ok(RunnerAuthStatus {
            status: resp.status,
            runner_id: resp.runner_id,
            organization_slug: resp.org_slug,
        })
    }

    // Proto-bytes flavour of get_runner_auth_status — used by the wasm/NAPI
    // bridge so the renderer can issue/decode wire-aligned proto rather than
    // the legacy serde DTO. Backend path is identical to the JSON helper above.
    pub async fn get_runner_auth_status_connect(
        &self,
        req: &runner_proto::GetRunnerAuthStatusRequest,
    ) -> Result<runner_proto::RunnerAuthStatus, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerPublicService/GetRunnerAuthStatus",
            req,
        )
        .await
    }

    pub async fn authorize_runner(
        &self,
        data: &AuthorizeRunnerRequest,
    ) -> Result<serde_json::Value, ApiError> {
        let req = runner_proto::AuthorizeRunnerRequest {
            org_slug: self.current_org_slug(),
            auth_key: data.auth_key.clone(),
            node_id: data.node_id.clone().unwrap_or_default(),
        };
        let resp: runner_proto::AuthorizeRunnerResponse = connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/AuthorizeRunner",
            &req,
        )
        .await?;
        Ok(serde_json::json!({
            "runner_id": resp.runner_id,
            "node_id": resp.node_id,
            "message": resp.message,
        }))
    }

    // Proto-bytes flavour of authorize_runner. Note: the proto request
    // carries `org_slug` explicitly, so the caller (wasm bridge) is
    // responsible for filling it from the current session — matches the
    // pattern used by the other *_connect methods on this client.
    pub async fn authorize_runner_connect(
        &self,
        req: &runner_proto::AuthorizeRunnerRequest,
    ) -> Result<runner_proto::AuthorizeRunnerResponse, ApiError> {
        connect_call(
            self,
            "/proto.runner_api.v1.RunnerService/AuthorizeRunner",
            req,
        )
        .await
    }
}
