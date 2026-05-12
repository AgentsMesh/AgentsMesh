use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_user_credential_v1 as uc_proto;

impl ApiClient {
    pub async fn list_agent_credentials_connect(
        &self,
    ) -> Result<uc_proto::ListAgentCredentialProfilesResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserAgentCredentialService/ListAgentCredentialProfiles",
            &uc_proto::ListAgentCredentialProfilesRequest {},
        )
        .await
    }

    pub async fn list_agent_credentials_for_agent_connect(
        &self, req: &uc_proto::ListAgentCredentialProfilesForAgentRequest,
    ) -> Result<uc_proto::ListAgentCredentialProfilesForAgentResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserAgentCredentialService/ListAgentCredentialProfilesForAgent",
            req,
        )
        .await
    }

    pub async fn get_agent_credential_connect(
        &self, req: &uc_proto::GetAgentCredentialProfileRequest,
    ) -> Result<uc_proto::AgentCredentialProfile, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserAgentCredentialService/GetAgentCredentialProfile",
            req,
        )
        .await
    }

    pub async fn create_agent_credential_connect(
        &self, req: &uc_proto::CreateAgentCredentialProfileRequest,
    ) -> Result<uc_proto::AgentCredentialProfile, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserAgentCredentialService/CreateAgentCredentialProfile",
            req,
        )
        .await
    }

    pub async fn update_agent_credential_connect(
        &self, req: &uc_proto::UpdateAgentCredentialProfileRequest,
    ) -> Result<uc_proto::AgentCredentialProfile, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserAgentCredentialService/UpdateAgentCredentialProfile",
            req,
        )
        .await
    }

    pub async fn delete_agent_credential_connect(
        &self, req: &uc_proto::DeleteAgentCredentialProfileRequest,
    ) -> Result<uc_proto::DeleteAgentCredentialProfileResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserAgentCredentialService/DeleteAgentCredentialProfile",
            req,
        )
        .await
    }

    pub async fn set_default_agent_credential_connect(
        &self, req: &uc_proto::SetDefaultAgentCredentialProfileRequest,
    ) -> Result<uc_proto::AgentCredentialProfile, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserAgentCredentialService/SetDefaultAgentCredentialProfile",
            req,
        )
        .await
    }
}
