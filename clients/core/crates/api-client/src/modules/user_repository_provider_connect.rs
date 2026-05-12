use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_user_credential_v1 as uc_proto;

impl ApiClient {
    pub async fn list_repository_providers_connect(
        &self,
    ) -> Result<uc_proto::ListRepositoryProvidersResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserRepositoryProviderService/ListRepositoryProviders",
            &uc_proto::ListRepositoryProvidersRequest {},
        )
        .await
    }

    pub async fn get_repository_provider_connect(
        &self, req: &uc_proto::GetRepositoryProviderRequest,
    ) -> Result<uc_proto::RepositoryProvider, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserRepositoryProviderService/GetRepositoryProvider",
            req,
        )
        .await
    }

    pub async fn create_repository_provider_connect(
        &self, req: &uc_proto::CreateRepositoryProviderRequest,
    ) -> Result<uc_proto::RepositoryProvider, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserRepositoryProviderService/CreateRepositoryProvider",
            req,
        )
        .await
    }

    pub async fn update_repository_provider_connect(
        &self, req: &uc_proto::UpdateRepositoryProviderRequest,
    ) -> Result<uc_proto::RepositoryProvider, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserRepositoryProviderService/UpdateRepositoryProvider",
            req,
        )
        .await
    }

    pub async fn delete_repository_provider_connect(
        &self, req: &uc_proto::DeleteRepositoryProviderRequest,
    ) -> Result<uc_proto::DeleteRepositoryProviderResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserRepositoryProviderService/DeleteRepositoryProvider",
            req,
        )
        .await
    }

    pub async fn set_default_repository_provider_connect(
        &self, req: &uc_proto::SetDefaultRepositoryProviderRequest,
    ) -> Result<uc_proto::SetDefaultRepositoryProviderResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserRepositoryProviderService/SetDefaultRepositoryProvider",
            req,
        )
        .await
    }

    pub async fn test_repository_provider_connection_connect(
        &self, req: &uc_proto::TestRepositoryProviderConnectionRequest,
    ) -> Result<uc_proto::TestRepositoryProviderConnectionResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserRepositoryProviderService/TestRepositoryProviderConnection",
            req,
        )
        .await
    }

    pub async fn list_provider_repositories_connect(
        &self, req: &uc_proto::ListProviderRepositoriesRequest,
    ) -> Result<uc_proto::ListProviderRepositoriesResponse, ApiError> {
        connect_call(
            self,
            "/proto.user_credential.v1.UserRepositoryProviderService/ListProviderRepositories",
            req,
        )
        .await
    }
}
