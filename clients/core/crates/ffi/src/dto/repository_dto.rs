use agentsmesh_types::proto_repository_v1 as repo_proto;
use agentsmesh_types::{
    Branch, CreateRepositoryRequest, MergeRequestListResponse, Repository, RepositoryListResponse,
    RepositoryMergeRequest, UpdateRepositoryRequest, WebhookSecret, WebhookStatus,
};

fn opt_str(s: String) -> Option<String> {
    if s.is_empty() { None } else { Some(s) }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct RepositoryDto {
    pub id: i64,
    pub name: String,
    pub slug: Option<String>,
    pub provider_type: Option<String>,
    pub provider_base_url: Option<String>,
    pub http_clone_url: Option<String>,
    pub ssh_clone_url: Option<String>,
    pub external_id: Option<String>,
    pub default_branch: Option<String>,
    pub ticket_prefix: Option<String>,
    pub visibility: Option<String>,
    pub is_active: Option<bool>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

impl From<Repository> for RepositoryDto {
    fn from(r: Repository) -> Self {
        Self {
            id: r.id,
            name: r.name,
            slug: r.slug,
            provider_type: r.provider_type,
            provider_base_url: r.provider_base_url,
            http_clone_url: r.http_clone_url,
            ssh_clone_url: r.ssh_clone_url,
            external_id: r.external_id,
            default_branch: r.default_branch,
            ticket_prefix: r.ticket_prefix,
            visibility: r.visibility,
            is_active: r.is_active,
            created_at: r.created_at,
            updated_at: r.updated_at,
        }
    }
}

impl From<repo_proto::Repository> for RepositoryDto {
    fn from(r: repo_proto::Repository) -> Self {
        Self {
            id: r.id,
            name: r.name,
            slug: opt_str(r.slug),
            provider_type: opt_str(r.provider_type),
            provider_base_url: opt_str(r.provider_base_url),
            http_clone_url: opt_str(r.http_clone_url),
            ssh_clone_url: opt_str(r.ssh_clone_url),
            external_id: opt_str(r.external_id),
            default_branch: opt_str(r.default_branch),
            ticket_prefix: r.ticket_prefix,
            visibility: opt_str(r.visibility),
            is_active: Some(r.is_active),
            created_at: opt_str(r.created_at),
            updated_at: opt_str(r.updated_at),
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct RepositoryListResponseDto {
    pub repositories: Vec<RepositoryDto>,
}

impl From<RepositoryListResponse> for RepositoryListResponseDto {
    fn from(r: RepositoryListResponse) -> Self {
        Self {
            repositories: r.repositories.into_iter().map(RepositoryDto::from).collect(),
        }
    }
}

pub(crate) fn repository_list_from_proto(
    resp: repo_proto::ListRepositoriesResponse,
) -> RepositoryListResponseDto {
    RepositoryListResponseDto {
        repositories: resp.items.into_iter().map(RepositoryDto::from).collect(),
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateRepositoryRequestDto {
    pub provider_type: Option<String>,
    pub provider_base_url: Option<String>,
    pub http_clone_url: Option<String>,
    pub ssh_clone_url: Option<String>,
    pub external_id: Option<String>,
    pub name: String,
    pub slug: Option<String>,
    pub default_branch: Option<String>,
    pub ticket_prefix: Option<String>,
    pub visibility: Option<String>,
}

impl From<CreateRepositoryRequestDto> for CreateRepositoryRequest {
    fn from(d: CreateRepositoryRequestDto) -> Self {
        Self {
            provider_type: d.provider_type,
            provider_base_url: d.provider_base_url,
            http_clone_url: d.http_clone_url,
            ssh_clone_url: d.ssh_clone_url,
            external_id: d.external_id,
            name: d.name,
            slug: d.slug,
            default_branch: d.default_branch,
            ticket_prefix: d.ticket_prefix,
            visibility: d.visibility,
        }
    }
}

pub(crate) fn build_create_repository_proto_request(
    org_slug: String,
    d: CreateRepositoryRequestDto,
) -> repo_proto::CreateRepositoryRequest {
    repo_proto::CreateRepositoryRequest {
        org_slug,
        provider_type: d.provider_type.unwrap_or_default(),
        provider_base_url: d.provider_base_url.unwrap_or_default(),
        http_clone_url: d.http_clone_url,
        ssh_clone_url: d.ssh_clone_url,
        external_id: d.external_id.unwrap_or_default(),
        name: d.name,
        slug: d.slug.unwrap_or_default(),
        default_branch: d.default_branch,
        ticket_prefix: d.ticket_prefix,
        visibility: d.visibility,
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct UpdateRepositoryRequestDto {
    pub name: Option<String>,
    pub default_branch: Option<String>,
    pub ticket_prefix: Option<String>,
    pub is_active: Option<bool>,
    pub http_clone_url: Option<String>,
    pub ssh_clone_url: Option<String>,
}

impl From<UpdateRepositoryRequestDto> for UpdateRepositoryRequest {
    fn from(d: UpdateRepositoryRequestDto) -> Self {
        Self {
            name: d.name,
            default_branch: d.default_branch,
            ticket_prefix: d.ticket_prefix,
            is_active: d.is_active,
            http_clone_url: d.http_clone_url,
            ssh_clone_url: d.ssh_clone_url,
        }
    }
}

pub(crate) fn build_update_repository_proto_request(
    org_slug: String,
    id: i64,
    d: UpdateRepositoryRequestDto,
) -> repo_proto::UpdateRepositoryRequest {
    repo_proto::UpdateRepositoryRequest {
        org_slug,
        id,
        name: d.name,
        default_branch: d.default_branch,
        ticket_prefix: d.ticket_prefix,
        is_active: d.is_active,
        http_clone_url: d.http_clone_url,
        ssh_clone_url: d.ssh_clone_url,
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct BranchDto {
    pub name: String,
    pub is_default: Option<bool>,
    pub last_commit: Option<String>,
}

impl From<Branch> for BranchDto {
    fn from(b: Branch) -> Self {
        Self {
            name: b.name,
            is_default: b.is_default,
            last_commit: b.last_commit,
        }
    }
}

// Proto Branch only carries the name — is_default and last_commit are not
// part of the .proto contract (PR #329). Both surface as None.
impl From<repo_proto::Branch> for BranchDto {
    fn from(b: repo_proto::Branch) -> Self {
        Self { name: b.name, is_default: None, last_commit: None }
    }
}

pub(crate) fn list_branches_from_proto(
    resp: repo_proto::ListRepositoryBranchesResponse,
) -> Vec<BranchDto> {
    resp.items.into_iter().map(BranchDto::from).collect()
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct WebhookStatusDto {
    pub is_configured: Option<bool>,
    pub url: Option<String>,
    pub events: Option<Vec<String>>,
}

impl From<WebhookStatus> for WebhookStatusDto {
    fn from(w: WebhookStatus) -> Self {
        Self {
            is_configured: w.is_configured,
            url: w.url,
            events: w.events,
        }
    }
}

impl From<repo_proto::WebhookStatus> for WebhookStatusDto {
    fn from(w: repo_proto::WebhookStatus) -> Self {
        Self {
            is_configured: Some(w.registered),
            url: w.webhook_url,
            events: Some(w.events),
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct WebhookSecretDto {
    pub secret: String,
}

impl From<WebhookSecret> for WebhookSecretDto {
    fn from(w: WebhookSecret) -> Self {
        Self { secret: w.secret }
    }
}

impl From<repo_proto::WebhookSecret> for WebhookSecretDto {
    fn from(w: repo_proto::WebhookSecret) -> Self {
        Self { secret: w.webhook_secret }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct RepositoryMergeRequestDto {
    pub id: i64,
    pub title: Option<String>,
    pub state: Option<String>,
    pub source_branch: Option<String>,
    pub target_branch: Option<String>,
    pub author: Option<String>,
    pub url: Option<String>,
    pub created_at: Option<String>,
}

impl From<RepositoryMergeRequest> for RepositoryMergeRequestDto {
    fn from(m: RepositoryMergeRequest) -> Self {
        Self {
            id: m.id,
            title: m.title,
            state: m.state,
            source_branch: m.source_branch,
            target_branch: m.target_branch,
            author: m.author,
            url: m.url,
            created_at: m.created_at,
        }
    }
}

// Proto MergeRequest has no `author` or `created_at` — both stay None.
impl From<repo_proto::MergeRequest> for RepositoryMergeRequestDto {
    fn from(m: repo_proto::MergeRequest) -> Self {
        Self {
            id: m.id,
            title: opt_str(m.title),
            state: opt_str(m.state),
            source_branch: opt_str(m.source_branch),
            target_branch: opt_str(m.target_branch),
            author: None,
            url: opt_str(m.mr_url),
            created_at: None,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct MergeRequestListResponseDto {
    pub merge_requests: Vec<RepositoryMergeRequestDto>,
}

impl From<MergeRequestListResponse> for MergeRequestListResponseDto {
    fn from(r: MergeRequestListResponse) -> Self {
        Self {
            merge_requests: r
                .merge_requests
                .into_iter()
                .map(RepositoryMergeRequestDto::from)
                .collect(),
        }
    }
}

pub(crate) fn merge_request_list_from_proto(
    resp: repo_proto::ListRepositoryMergeRequestsResponse,
) -> MergeRequestListResponseDto {
    MergeRequestListResponseDto {
        merge_requests: resp.items.into_iter().map(RepositoryMergeRequestDto::from).collect(),
    }
}
