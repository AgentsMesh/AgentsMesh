use agentsmesh_types::proto_repository_v1 as repo_proto;

use crate::core::AgentsMeshCore;
use crate::dto::{
    build_create_repository_proto_request, build_update_repository_proto_request,
    list_branches_from_proto, merge_request_list_from_proto, repository_list_from_proto, BranchDto,
    CreateRepositoryRequestDto, MergeRequestListResponseDto, RepositoryDto,
    RepositoryListResponseDto, UpdateRepositoryRequestDto, WebhookSecretDto, WebhookStatusDto,
};
use crate::error::CoreError;

#[uniffi::export(async_runtime = "tokio")]
impl AgentsMeshCore {
    pub async fn list_repositories(&self) -> Result<RepositoryListResponseDto, CoreError> {
        let req = repo_proto::ListRepositoriesRequest {
            org_slug: self.org_slug()?,
            offset: None,
            limit: None,
        };
        let resp = self.api.list_repositories_connect(&req).await?;
        Ok(repository_list_from_proto(resp))
    }

    pub async fn get_repository(&self, id: i64) -> Result<RepositoryDto, CoreError> {
        let req = repo_proto::GetRepositoryRequest { org_slug: self.org_slug()?, id };
        let repo = self.api.get_repository_connect(&req).await?;
        Ok(repo.into())
    }

    pub async fn create_repository(
        &self,
        req: CreateRepositoryRequestDto,
    ) -> Result<RepositoryDto, CoreError> {
        let proto_req = build_create_repository_proto_request(self.org_slug()?, req);
        let repo = self.api.create_repository_connect(&proto_req).await?;
        Ok(repo.into())
    }

    pub async fn update_repository(
        &self,
        id: i64,
        req: UpdateRepositoryRequestDto,
    ) -> Result<RepositoryDto, CoreError> {
        let proto_req = build_update_repository_proto_request(self.org_slug()?, id, req);
        let repo = self.api.update_repository_connect(&proto_req).await?;
        Ok(repo.into())
    }

    pub async fn delete_repository(&self, id: i64) -> Result<(), CoreError> {
        let req = repo_proto::DeleteRepositoryRequest { org_slug: self.org_slug()?, id };
        self.api.delete_repository_connect(&req).await?;
        Ok(())
    }

    pub async fn list_repository_branches(&self, id: i64) -> Result<Vec<BranchDto>, CoreError> {
        let req = repo_proto::ListRepositoryBranchesRequest {
            org_slug: self.org_slug()?,
            id,
            access_token: String::new(),
        };
        let resp = self.api.list_repository_branches_connect(&req).await?;
        Ok(list_branches_from_proto(resp))
    }

    pub async fn sync_repository_branches(
        &self,
        id: i64,
        access_token: Option<String>,
    ) -> Result<Vec<BranchDto>, CoreError> {
        let req = repo_proto::SyncRepositoryBranchesRequest {
            org_slug: self.org_slug()?,
            id,
            access_token: access_token.unwrap_or_default(),
        };
        let resp = self.api.sync_repository_branches_connect(&req).await?;
        Ok(list_branches_from_proto(resp))
    }

    pub async fn register_repository_webhook(&self, id: i64) -> Result<(), CoreError> {
        let req = repo_proto::RegisterRepositoryWebhookRequest { org_slug: self.org_slug()?, id };
        self.api.register_repository_webhook_connect(&req).await?;
        Ok(())
    }

    pub async fn delete_repository_webhook(&self, id: i64) -> Result<(), CoreError> {
        let req = repo_proto::DeleteRepositoryWebhookRequest { org_slug: self.org_slug()?, id };
        self.api.delete_repository_webhook_connect(&req).await?;
        Ok(())
    }

    pub async fn get_repository_webhook_status(
        &self,
        id: i64,
    ) -> Result<WebhookStatusDto, CoreError> {
        let req = repo_proto::GetRepositoryWebhookStatusRequest {
            org_slug: self.org_slug()?,
            id,
        };
        let status = self.api.get_repository_webhook_status_connect(&req).await?;
        Ok(status.into())
    }

    pub async fn get_repository_webhook_secret(
        &self,
        id: i64,
    ) -> Result<WebhookSecretDto, CoreError> {
        let req = repo_proto::GetRepositoryWebhookSecretRequest {
            org_slug: self.org_slug()?,
            id,
        };
        let secret = self.api.get_repository_webhook_secret_connect(&req).await?;
        Ok(secret.into())
    }

    pub async fn list_repository_merge_requests(
        &self,
        id: i64,
        branch: Option<String>,
        state: Option<String>,
    ) -> Result<MergeRequestListResponseDto, CoreError> {
        let req = repo_proto::ListRepositoryMergeRequestsRequest {
            org_slug: self.org_slug()?,
            id,
            branch,
            state,
        };
        let resp = self.api.list_repository_merge_requests_connect(&req).await?;
        Ok(merge_request_list_from_proto(resp))
    }
}
