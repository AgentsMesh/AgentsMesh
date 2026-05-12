use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::*;
use agentsmesh_types::proto_pod_v1 as pod_proto;

// =============================================================================
// Connect-RPC (binary wire). User-scoped (no org_slug) — see conventions §3.5.
// =============================================================================

impl ApiClient {
    pub async fn get_agentpod_settings_connect(
        &self,
    ) -> Result<pod_proto::AgentPodSettings, ApiError> {
        connect_call(
            self,
            "/proto.pod.v1.AgentPodSettingsService/GetSettings",
            &pod_proto::GetSettingsRequest {},
        )
        .await
    }

    pub async fn update_agentpod_settings_connect(
        &self,
        req: &pod_proto::UpdateSettingsRequest,
    ) -> Result<pod_proto::AgentPodSettings, ApiError> {
        connect_call(
            self,
            "/proto.pod.v1.AgentPodSettingsService/UpdateSettings",
            req,
        )
        .await
    }

    pub async fn list_agentpod_providers_connect(
        &self,
    ) -> Result<pod_proto::ListProvidersResponse, ApiError> {
        connect_call(
            self,
            "/proto.pod.v1.AgentPodSettingsService/ListProviders",
            &pod_proto::ListProvidersRequest {},
        )
        .await
    }

    pub async fn create_agentpod_provider_connect(
        &self,
        req: &pod_proto::CreateProviderRequest,
    ) -> Result<pod_proto::AIProvider, ApiError> {
        connect_call(
            self,
            "/proto.pod.v1.AgentPodSettingsService/CreateProvider",
            req,
        )
        .await
    }

    pub async fn update_agentpod_provider_connect(
        &self,
        req: &pod_proto::UpdateProviderRequest,
    ) -> Result<pod_proto::AIProvider, ApiError> {
        connect_call(
            self,
            "/proto.pod.v1.AgentPodSettingsService/UpdateProvider",
            req,
        )
        .await
    }

    pub async fn delete_agentpod_provider_connect(
        &self,
        req: &pod_proto::DeleteProviderRequest,
    ) -> Result<pod_proto::DeleteProviderResponse, ApiError> {
        connect_call(
            self,
            "/proto.pod.v1.AgentPodSettingsService/DeleteProvider",
            req,
        )
        .await
    }

    pub async fn set_default_agentpod_provider_connect(
        &self,
        req: &pod_proto::SetDefaultProviderRequest,
    ) -> Result<pod_proto::SetDefaultProviderResponse, ApiError> {
        connect_call(
            self,
            "/proto.pod.v1.AgentPodSettingsService/SetDefaultProvider",
            req,
        )
        .await
    }
}

// =============================================================================
// Legacy REST methods — preserved for dual-track migration.
// =============================================================================

impl ApiClient {
    pub async fn get_agentpod_settings(&self) -> Result<AgentPodSettings, ApiError> {
        self.get_resource("/api/v1/users/me/agentpod/settings", "settings").await
    }

    pub async fn update_agentpod_settings(
        &self,
        data: &AgentPodSettings,
    ) -> Result<AgentPodSettings, ApiError> {
        self.put("/api/v1/users/me/agentpod/settings", data).await
    }

    pub async fn list_agentpod_providers(&self) -> Result<AIProviderListResponse, ApiError> {
        self.get("/api/v1/users/me/agentpod/providers").await
    }

    pub async fn create_agentpod_provider(
        &self,
        data: &CreateAIProviderRequest,
    ) -> Result<AIProvider, ApiError> {
        self.post("/api/v1/users/me/agentpod/providers", data)
            .await
    }

    pub async fn update_agentpod_provider(
        &self,
        id: i64,
        data: &UpdateAIProviderRequest,
    ) -> Result<AIProvider, ApiError> {
        self.put(
            &format!("/api/v1/users/me/agentpod/providers/{id}"),
            data,
        )
        .await
    }

    pub async fn delete_agentpod_provider(&self, id: i64) -> Result<EmptyResponse, ApiError> {
        self.delete(&format!("/api/v1/users/me/agentpod/providers/{id}"))
            .await
    }

    pub async fn set_default_agentpod_provider(
        &self,
        id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(
            &format!("/api/v1/users/me/agentpod/providers/{id}/default"),
            &serde_json::json!({}),
        )
        .await
    }
}
