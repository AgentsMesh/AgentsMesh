use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_repository_v1 as repo_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// These methods call the Connect handlers in
// backend/internal/api/connect/repository/. Procedure paths derive from
// `proto.repository.v1.RepositoryService.<Method>` (conventions §12).

impl ApiClient {
    pub async fn list_repositories_connect(
        &self,
        req: &repo_proto::ListRepositoriesRequest,
    ) -> Result<repo_proto::ListRepositoriesResponse, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/ListRepositories",
            req,
        )
        .await
    }

    pub async fn get_repository_connect(
        &self,
        req: &repo_proto::GetRepositoryRequest,
    ) -> Result<repo_proto::Repository, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/GetRepository",
            req,
        )
        .await
    }

    pub async fn create_repository_connect(
        &self,
        req: &repo_proto::CreateRepositoryRequest,
    ) -> Result<repo_proto::Repository, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/CreateRepository",
            req,
        )
        .await
    }

    pub async fn update_repository_connect(
        &self,
        req: &repo_proto::UpdateRepositoryRequest,
    ) -> Result<repo_proto::Repository, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/UpdateRepository",
            req,
        )
        .await
    }

    pub async fn delete_repository_connect(
        &self,
        req: &repo_proto::DeleteRepositoryRequest,
    ) -> Result<repo_proto::DeleteRepositoryResponse, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/DeleteRepository",
            req,
        )
        .await
    }

    pub async fn list_repository_branches_connect(
        &self,
        req: &repo_proto::ListRepositoryBranchesRequest,
    ) -> Result<repo_proto::ListRepositoryBranchesResponse, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/ListRepositoryBranches",
            req,
        )
        .await
    }

    pub async fn sync_repository_branches_connect(
        &self,
        req: &repo_proto::SyncRepositoryBranchesRequest,
    ) -> Result<repo_proto::ListRepositoryBranchesResponse, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/SyncRepositoryBranches",
            req,
        )
        .await
    }

    pub async fn list_repository_merge_requests_connect(
        &self,
        req: &repo_proto::ListRepositoryMergeRequestsRequest,
    ) -> Result<repo_proto::ListRepositoryMergeRequestsResponse, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/ListRepositoryMergeRequests",
            req,
        )
        .await
    }

    pub async fn register_repository_webhook_connect(
        &self,
        req: &repo_proto::RegisterRepositoryWebhookRequest,
    ) -> Result<repo_proto::RegisterRepositoryWebhookResponse, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/RegisterRepositoryWebhook",
            req,
        )
        .await
    }

    pub async fn delete_repository_webhook_connect(
        &self,
        req: &repo_proto::DeleteRepositoryWebhookRequest,
    ) -> Result<repo_proto::DeleteRepositoryWebhookResponse, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/DeleteRepositoryWebhook",
            req,
        )
        .await
    }

    pub async fn get_repository_webhook_status_connect(
        &self,
        req: &repo_proto::GetRepositoryWebhookStatusRequest,
    ) -> Result<repo_proto::WebhookStatus, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/GetRepositoryWebhookStatus",
            req,
        )
        .await
    }

    pub async fn get_repository_webhook_secret_connect(
        &self,
        req: &repo_proto::GetRepositoryWebhookSecretRequest,
    ) -> Result<repo_proto::WebhookSecret, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/GetRepositoryWebhookSecret",
            req,
        )
        .await
    }

    pub async fn mark_repository_webhook_configured_connect(
        &self,
        req: &repo_proto::MarkRepositoryWebhookConfiguredRequest,
    ) -> Result<repo_proto::MarkRepositoryWebhookConfiguredResponse, ApiError> {
        connect_call(
            self,
            "/proto.repository.v1.RepositoryService/MarkRepositoryWebhookConfigured",
            req,
        )
        .await
    }
}

