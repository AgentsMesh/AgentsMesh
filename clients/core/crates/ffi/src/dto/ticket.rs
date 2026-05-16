use agentsmesh_types::{
    AddAssigneeRequest, AddTicketLabelRequest, BoardColumn, BoardResponse, CreateLabelRequest,
    CreateTicketCommentRequest, CreateTicketRelationRequest, CreateTicketRequest, Label,
    LabelListResponse, LinkTicketCommitRequest, Ticket, TicketComment, TicketCommentListResponse,
    TicketCommit, TicketCommitListResponse, TicketListResponse, TicketPriority, TicketRelation,
    TicketRelationListResponse, TicketStatus, UpdateLabelRequest, UpdateTicketCommentRequest,
    UpdateTicketRequest, UpdateTicketStatusRequest,
};

use super::UserDto;

// ── Enums ─────────────────────────────────────────────────

#[derive(Clone, Copy, Debug, uniffi::Enum)]
pub enum TicketStatusDto {
    Backlog,
    Todo,
    InProgress,
    InReview,
    Done,
    Unknown,
}

impl From<TicketStatus> for TicketStatusDto {
    fn from(s: TicketStatus) -> Self {
        match s {
            TicketStatus::Backlog => Self::Backlog,
            TicketStatus::Todo => Self::Todo,
            TicketStatus::InProgress => Self::InProgress,
            TicketStatus::InReview => Self::InReview,
            TicketStatus::Done => Self::Done,
            TicketStatus::Unknown => Self::Unknown,
        }
    }
}

impl From<TicketStatusDto> for TicketStatus {
    fn from(s: TicketStatusDto) -> Self {
        match s {
            TicketStatusDto::Backlog => Self::Backlog,
            TicketStatusDto::Todo => Self::Todo,
            TicketStatusDto::InProgress => Self::InProgress,
            TicketStatusDto::InReview => Self::InReview,
            TicketStatusDto::Done => Self::Done,
            TicketStatusDto::Unknown => Self::Unknown,
        }
    }
}

#[derive(Clone, Copy, Debug, uniffi::Enum)]
pub enum TicketPriorityDto {
    None,
    Low,
    Medium,
    High,
    Urgent,
    Unknown,
}

impl From<TicketPriority> for TicketPriorityDto {
    fn from(p: TicketPriority) -> Self {
        match p {
            TicketPriority::None => Self::None,
            TicketPriority::Low => Self::Low,
            TicketPriority::Medium => Self::Medium,
            TicketPriority::High => Self::High,
            TicketPriority::Urgent => Self::Urgent,
            TicketPriority::Unknown => Self::Unknown,
        }
    }
}

impl From<TicketPriorityDto> for TicketPriority {
    fn from(p: TicketPriorityDto) -> Self {
        match p {
            TicketPriorityDto::None => Self::None,
            TicketPriorityDto::Low => Self::Low,
            TicketPriorityDto::Medium => Self::Medium,
            TicketPriorityDto::High => Self::High,
            TicketPriorityDto::Urgent => Self::Urgent,
            TicketPriorityDto::Unknown => Self::Unknown,
        }
    }
}

// ── Ticket ────────────────────────────────────────────────

#[derive(Clone, Debug, uniffi::Record)]
pub struct TicketDto {
    pub id: i64,
    pub slug: String,
    pub title: String,
    pub content: Option<String>,
    pub status: TicketStatusDto,
    pub priority: TicketPriorityDto,
    pub repository_id: Option<i64>,
    pub parent_slug: Option<String>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

impl From<Ticket> for TicketDto {
    fn from(t: Ticket) -> Self {
        Self {
            id: t.id,
            slug: t.slug,
            title: t.title,
            content: t.content,
            status: t.status.into(),
            priority: t.priority.into(),
            repository_id: t.repository_id,
            parent_slug: t.parent_slug,
            created_at: t.created_at,
            updated_at: t.updated_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct TicketListResponseDto {
    pub tickets: Vec<TicketDto>,
    pub total: Option<i64>,
    pub limit: Option<i64>,
    pub offset: Option<i64>,
}

impl From<TicketListResponse> for TicketListResponseDto {
    fn from(r: TicketListResponse) -> Self {
        Self {
            tickets: r.tickets.into_iter().map(TicketDto::from).collect(),
            total: r.total,
            limit: r.limit,
            offset: r.offset,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct BoardColumnDto {
    pub status: TicketStatusDto,
    pub tickets: Vec<TicketDto>,
    pub total_count: i64,
}

impl From<BoardColumn> for BoardColumnDto {
    fn from(c: BoardColumn) -> Self {
        Self {
            status: c.status.into(),
            tickets: c.tickets.into_iter().map(TicketDto::from).collect(),
            total_count: c.total_count,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct BoardResponseDto {
    pub columns: Vec<BoardColumnDto>,
    /// Opaque JSON of per-priority counts (server-side aggregated).
    pub priority_counts_json: Option<String>,
}

impl From<BoardResponse> for BoardResponseDto {
    fn from(r: BoardResponse) -> Self {
        Self {
            columns: r.columns.into_iter().map(BoardColumnDto::from).collect(),
            priority_counts_json: r
                .priority_counts
                .and_then(|v| serde_json::to_string(&v).ok()),
        }
    }
}

// ── Label ─────────────────────────────────────────────────

#[derive(Clone, Debug, uniffi::Record)]
pub struct LabelDto {
    pub id: i64,
    pub name: String,
    pub color: String,
}

impl From<Label> for LabelDto {
    fn from(l: Label) -> Self {
        Self {
            id: l.id,
            name: l.name,
            color: l.color,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct LabelListResponseDto {
    pub labels: Vec<LabelDto>,
}

impl From<LabelListResponse> for LabelListResponseDto {
    fn from(r: LabelListResponse) -> Self {
        Self {
            labels: r.labels.into_iter().map(LabelDto::from).collect(),
        }
    }
}

// ── Ticket Comment ────────────────────────────────────────

#[derive(Clone, Debug, uniffi::Record)]
pub struct TicketCommentDto {
    pub id: i64,
    pub ticket_slug: Option<String>,
    pub content: String,
    pub parent_id: Option<i64>,
    pub author: Option<UserDto>,
    pub mentions: Option<Vec<String>>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

impl From<TicketComment> for TicketCommentDto {
    fn from(c: TicketComment) -> Self {
        Self {
            id: c.id,
            ticket_slug: c.ticket_slug,
            content: c.content,
            parent_id: c.parent_id,
            author: c.author.map(UserDto::from),
            mentions: c.mentions,
            created_at: c.created_at,
            updated_at: c.updated_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct TicketCommentListResponseDto {
    pub comments: Vec<TicketCommentDto>,
    pub total: Option<i64>,
    pub limit: Option<i64>,
    pub offset: Option<i64>,
}

impl From<TicketCommentListResponse> for TicketCommentListResponseDto {
    fn from(r: TicketCommentListResponse) -> Self {
        Self {
            comments: r.comments.into_iter().map(TicketCommentDto::from).collect(),
            total: r.total,
            limit: r.limit,
            offset: r.offset,
        }
    }
}

// ── Ticket Relation ───────────────────────────────────────

#[derive(Clone, Debug, uniffi::Record)]
pub struct TicketRelationDto {
    pub id: i64,
    pub source_slug: Option<String>,
    pub target_slug: Option<String>,
    pub relation_type: Option<String>,
    pub created_at: Option<String>,
}

impl From<TicketRelation> for TicketRelationDto {
    fn from(r: TicketRelation) -> Self {
        Self {
            id: r.id,
            source_slug: r.source_slug,
            target_slug: r.target_slug,
            relation_type: r.relation_type,
            created_at: r.created_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct TicketRelationListResponseDto {
    pub relations: Vec<TicketRelationDto>,
}

impl From<TicketRelationListResponse> for TicketRelationListResponseDto {
    fn from(r: TicketRelationListResponse) -> Self {
        Self {
            relations: r.relations.into_iter().map(TicketRelationDto::from).collect(),
        }
    }
}

// ── Ticket Commit ─────────────────────────────────────────

#[derive(Clone, Debug, uniffi::Record)]
pub struct TicketCommitDto {
    pub id: i64,
    pub ticket_slug: Option<String>,
    pub commit_sha: String,
    pub commit_message: Option<String>,
    pub commit_url: Option<String>,
    pub author_name: Option<String>,
    pub author_email: Option<String>,
    pub committed_at: Option<String>,
}

impl From<TicketCommit> for TicketCommitDto {
    fn from(c: TicketCommit) -> Self {
        Self {
            id: c.id,
            ticket_slug: c.ticket_slug,
            commit_sha: c.commit_sha,
            commit_message: c.commit_message,
            commit_url: c.commit_url,
            author_name: c.author_name,
            author_email: c.author_email,
            committed_at: c.committed_at,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct TicketCommitListResponseDto {
    pub commits: Vec<TicketCommitDto>,
}

impl From<TicketCommitListResponse> for TicketCommitListResponseDto {
    fn from(r: TicketCommitListResponse) -> Self {
        Self {
            commits: r.commits.into_iter().map(TicketCommitDto::from).collect(),
        }
    }
}

// ── Request DTOs ──────────────────────────────────────────

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateTicketRequestDto {
    pub title: String,
    pub content: Option<String>,
    pub priority: Option<TicketPriorityDto>,
    pub severity: Option<String>,
    pub estimate: Option<f64>,
    pub repository_id: Option<i64>,
    pub assignee_ids: Option<Vec<i64>>,
    pub labels: Option<Vec<i64>>,
    pub parent_slug: Option<String>,
}

impl From<CreateTicketRequestDto> for CreateTicketRequest {
    fn from(d: CreateTicketRequestDto) -> Self {
        Self {
            title: d.title,
            content: d.content,
            priority: d.priority.map(Into::into),
            severity: d.severity,
            estimate: d.estimate,
            repository_id: d.repository_id,
            assignee_ids: d.assignee_ids,
            labels: d.labels,
            parent_slug: d.parent_slug,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct UpdateTicketRequestDto {
    pub title: Option<String>,
    pub content: Option<String>,
    pub priority: Option<TicketPriorityDto>,
    pub severity: Option<String>,
    pub estimate: Option<f64>,
    pub repository_id: Option<i64>,
}

impl From<UpdateTicketRequestDto> for UpdateTicketRequest {
    fn from(d: UpdateTicketRequestDto) -> Self {
        Self {
            title: d.title,
            content: d.content,
            priority: d.priority.map(Into::into),
            severity: d.severity,
            estimate: d.estimate,
            repository_id: d.repository_id,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateLabelRequestDto {
    pub name: String,
    pub color: String,
}

impl From<CreateLabelRequestDto> for CreateLabelRequest {
    fn from(d: CreateLabelRequestDto) -> Self {
        Self {
            name: d.name,
            color: d.color,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct UpdateLabelRequestDto {
    pub name: Option<String>,
    pub color: Option<String>,
}

impl From<UpdateLabelRequestDto> for UpdateLabelRequest {
    fn from(d: UpdateLabelRequestDto) -> Self {
        Self {
            name: d.name,
            color: d.color,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateTicketCommentRequestDto {
    pub content: String,
    pub parent_id: Option<i64>,
    pub mentions: Option<Vec<String>>,
}

impl From<CreateTicketCommentRequestDto> for CreateTicketCommentRequest {
    fn from(d: CreateTicketCommentRequestDto) -> Self {
        Self {
            content: d.content,
            parent_id: d.parent_id,
            mentions: d.mentions,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct UpdateTicketCommentRequestDto {
    pub content: String,
}

impl From<UpdateTicketCommentRequestDto> for UpdateTicketCommentRequest {
    fn from(d: UpdateTicketCommentRequestDto) -> Self {
        Self { content: d.content }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreateTicketRelationRequestDto {
    pub target_slug: String,
    pub relation_type: String,
}

impl From<CreateTicketRelationRequestDto> for CreateTicketRelationRequest {
    fn from(d: CreateTicketRelationRequestDto) -> Self {
        Self {
            target_slug: d.target_slug,
            relation_type: d.relation_type,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct LinkTicketCommitRequestDto {
    pub commit_sha: String,
    pub commit_message: Option<String>,
    pub commit_url: Option<String>,
    pub author_name: Option<String>,
    pub author_email: Option<String>,
    pub committed_at: Option<String>,
}

impl From<LinkTicketCommitRequestDto> for LinkTicketCommitRequest {
    fn from(d: LinkTicketCommitRequestDto) -> Self {
        Self {
            commit_sha: d.commit_sha,
            commit_message: d.commit_message,
            commit_url: d.commit_url,
            author_name: d.author_name,
            author_email: d.author_email,
            committed_at: d.committed_at,
        }
    }
}

pub(crate) fn update_ticket_status_req(status: TicketStatusDto) -> UpdateTicketStatusRequest {
    UpdateTicketStatusRequest {
        status: status.into(),
    }
}

pub(crate) fn add_assignee_req(user_id: i64) -> AddAssigneeRequest {
    AddAssigneeRequest { user_id }
}

pub(crate) fn add_ticket_label_req(label_id: i64) -> AddTicketLabelRequest {
    AddTicketLabelRequest { label_id }
}
