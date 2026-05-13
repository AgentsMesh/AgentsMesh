use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_agent_v1 as agent_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// These methods call the Connect handlers in
// backend/internal/api/connect/agent/. Procedure paths derive from
// `proto.agent.v1.<Service>.<Method>` (conventions §12).

impl ApiClient {
    pub async fn list_agents_connect(
        &self,
        req: &agent_proto::ListAgentsRequest,
    ) -> Result<agent_proto::AgentListResponse, ApiError> {
        connect_call(self, "/proto.agent.v1.AgentService/ListAgents", req).await
    }

    pub async fn get_agent_connect(
        &self,
        req: &agent_proto::GetAgentRequest,
    ) -> Result<agent_proto::Agent, ApiError> {
        connect_call(self, "/proto.agent.v1.AgentService/GetAgent", req).await
    }

    pub async fn get_agent_config_schema_connect(
        &self,
        req: &agent_proto::GetAgentConfigSchemaRequest,
    ) -> Result<agent_proto::ConfigSchema, ApiError> {
        connect_call(
            self,
            "/proto.agent.v1.AgentService/GetAgentConfigSchema",
            req,
        )
        .await
    }

    pub async fn create_custom_agent_connect(
        &self,
        req: &agent_proto::CreateCustomAgentRequest,
    ) -> Result<agent_proto::Agent, ApiError> {
        connect_call(self, "/proto.agent.v1.AgentService/CreateCustomAgent", req).await
    }

    pub async fn update_custom_agent_connect(
        &self,
        req: &agent_proto::UpdateCustomAgentRequest,
    ) -> Result<agent_proto::Agent, ApiError> {
        connect_call(self, "/proto.agent.v1.AgentService/UpdateCustomAgent", req).await
    }

    pub async fn delete_custom_agent_connect(
        &self,
        req: &agent_proto::DeleteCustomAgentRequest,
    ) -> Result<agent_proto::DeleteCustomAgentResponse, ApiError> {
        connect_call(self, "/proto.agent.v1.AgentService/DeleteCustomAgent", req).await
    }

    pub async fn list_user_agent_configs_connect(
        &self,
    ) -> Result<agent_proto::UserAgentConfigListResponse, ApiError> {
        connect_call(
            self,
            "/proto.agent.v1.UserAgentConfigService/ListUserAgentConfigs",
            &agent_proto::ListUserAgentConfigsRequest {},
        )
        .await
    }

    pub async fn get_user_agent_config_connect(
        &self,
        req: &agent_proto::GetUserAgentConfigRequest,
    ) -> Result<agent_proto::UserAgentConfig, ApiError> {
        connect_call(
            self,
            "/proto.agent.v1.UserAgentConfigService/GetUserAgentConfig",
            req,
        )
        .await
    }

    pub async fn set_user_agent_config_connect(
        &self,
        req: &agent_proto::SetUserAgentConfigRequest,
    ) -> Result<agent_proto::UserAgentConfig, ApiError> {
        connect_call(
            self,
            "/proto.agent.v1.UserAgentConfigService/SetUserAgentConfig",
            req,
        )
        .await
    }

    pub async fn delete_user_agent_config_connect(
        &self,
        req: &agent_proto::DeleteUserAgentConfigRequest,
    ) -> Result<agent_proto::DeleteUserAgentConfigResponse, ApiError> {
        connect_call(
            self,
            "/proto.agent.v1.UserAgentConfigService/DeleteUserAgentConfig",
            req,
        )
        .await
    }
}
