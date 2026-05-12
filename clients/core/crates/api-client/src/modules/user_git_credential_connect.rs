use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_user_credential_v1 as uc_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// Three user-scoped services share the proto.user_credential.v1 package:
// UserGitCredentialService, UserAgentCredentialService,
// UserRepositoryProviderService. The Connect handler at
// backend/internal/api/connect/user_credential/ implements all three.
//
// Methods are split across three files (one per sub-service) to stay
// under the 200-line file limit.

impl ApiClient {
    pub async fn list_git_credentials_connect(
        &self,
    ) -> Result<uc_proto::ListGitCredentialsResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserGitCredentialService/ListGitCredentials",
            &uc_proto::ListGitCredentialsRequest {},
        )
        .await
    }

    pub async fn get_git_credential_connect(
        &self, req: &uc_proto::GetGitCredentialRequest,
    ) -> Result<uc_proto::GitCredential, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserGitCredentialService/GetGitCredential",
            req,
        )
        .await
    }

    pub async fn create_git_credential_connect(
        &self, req: &uc_proto::CreateGitCredentialRequest,
    ) -> Result<uc_proto::GitCredential, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserGitCredentialService/CreateGitCredential",
            req,
        )
        .await
    }

    pub async fn update_git_credential_connect(
        &self, req: &uc_proto::UpdateGitCredentialRequest,
    ) -> Result<uc_proto::GitCredential, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserGitCredentialService/UpdateGitCredential",
            req,
        )
        .await
    }

    pub async fn delete_git_credential_connect(
        &self, req: &uc_proto::DeleteGitCredentialRequest,
    ) -> Result<uc_proto::DeleteGitCredentialResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserGitCredentialService/DeleteGitCredential",
            req,
        )
        .await
    }

    pub async fn get_default_git_credential_connect(
        &self,
    ) -> Result<uc_proto::GetDefaultGitCredentialResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserGitCredentialService/GetDefaultGitCredential",
            &uc_proto::GetDefaultGitCredentialRequest {},
        )
        .await
    }

    pub async fn set_default_git_credential_connect(
        &self, req: &uc_proto::SetDefaultGitCredentialRequest,
    ) -> Result<uc_proto::SetDefaultGitCredentialResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserGitCredentialService/SetDefaultGitCredential",
            req,
        )
        .await
    }

    pub async fn clear_default_git_credential_connect(
        &self,
    ) -> Result<uc_proto::ClearDefaultGitCredentialResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserGitCredentialService/ClearDefaultGitCredential",
            &uc_proto::ClearDefaultGitCredentialRequest {},
        )
        .await
    }
}
