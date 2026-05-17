use crate::ApiClient;
use crate::error::ApiError;
use agentsmesh_types::*;

impl ApiClient {
    pub async fn list_user_agent_credentials(
        &self,
    ) -> Result<AgentCredentialProfileListResponse, ApiError> {
        self.get("/api/v1/users/agent-credentials").await
    }

    pub async fn list_user_agent_credentials_for_agent(
        &self,
        agent_slug: &str,
    ) -> Result<AgentCredentialProfilesForAgentResponse, ApiError> {
        self.get(&format!(
            "/api/v1/users/agent-credentials/agents/{agent_slug}"
        ))
        .await
    }

    pub async fn create_user_agent_credential(
        &self,
        agent_slug: &str,
        data: &CreateAgentCredentialProfileRequest,
    ) -> Result<AgentCredentialProfile, ApiError> {
        self.post_resource(
            &format!("/api/v1/users/agent-credentials/agents/{agent_slug}"),
            data, "profile",
        ).await
    }

    pub async fn get_user_agent_credential(
        &self,
        profile_id: i64,
    ) -> Result<AgentCredentialProfile, ApiError> {
        self.get_resource(
            &format!("/api/v1/users/agent-credentials/profiles/{profile_id}"),
            "profile",
        ).await
    }

    pub async fn update_user_agent_credential(
        &self,
        profile_id: i64,
        data: &UpdateAgentCredentialProfileRequest,
    ) -> Result<AgentCredentialProfile, ApiError> {
        self.put_resource(
            &format!("/api/v1/users/agent-credentials/profiles/{profile_id}"),
            data, "profile",
        ).await
    }

    pub async fn delete_user_agent_credential(
        &self,
        profile_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.delete(&format!(
            "/api/v1/users/agent-credentials/profiles/{profile_id}"
        ))
        .await
    }

    pub async fn set_default_agent_credential(
        &self,
        profile_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(
            &format!("/api/v1/users/agent-credentials/profiles/{profile_id}/set-default"),
            &serde_json::json!({}),
        )
        .await
    }
}
