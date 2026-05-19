use agentsmesh_types::proto_pod_v1 as pod_proto;
use agentsmesh_types::{
    CreatePodRequest, CreatePodResponse, Pod, PodAgentInfo, PodConnectionInfo, PodCreatedByInfo,
    PodListResponse,
    PodLoopInfo, PodRepositoryInfo, PodRunnerInfo, PodStatus, PodTicketInfo,
};

#[derive(Clone, Copy, Debug, uniffi::Enum)]
pub enum PodStatusDto {
    Pending,
    Creating,
    Initializing,
    Running,
    Paused,
    Stopping,
    Disconnected,
    Orphaned,
    Completed,
    Terminated,
    Error,
    Failed,
    Unknown,
}

impl From<PodStatus> for PodStatusDto {
    fn from(s: PodStatus) -> Self {
        match s {
            PodStatus::Pending => Self::Pending,
            PodStatus::Creating => Self::Creating,
            PodStatus::Initializing => Self::Initializing,
            PodStatus::Running => Self::Running,
            PodStatus::Paused => Self::Paused,
            PodStatus::Stopping => Self::Stopping,
            PodStatus::Disconnected => Self::Disconnected,
            PodStatus::Orphaned => Self::Orphaned,
            PodStatus::Completed => Self::Completed,
            PodStatus::Terminated => Self::Terminated,
            PodStatus::Error => Self::Error,
            PodStatus::Failed => Self::Failed,
            PodStatus::Unknown => Self::Unknown,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodRunnerInfoDto {
    pub id: Option<i64>,
    pub node_id: Option<String>,
    pub status: Option<String>,
}

impl From<PodRunnerInfo> for PodRunnerInfoDto {
    fn from(r: PodRunnerInfo) -> Self {
        Self {
            id: r.id,
            node_id: r.node_id,
            status: r.status,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodAgentInfoDto {
    pub name: Option<String>,
    pub slug: Option<String>,
}

impl From<PodAgentInfo> for PodAgentInfoDto {
    fn from(a: PodAgentInfo) -> Self {
        Self {
            name: a.name,
            slug: a.slug,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodRepositoryInfoDto {
    pub id: Option<i64>,
    pub name: Option<String>,
    pub slug: Option<String>,
    pub provider_type: Option<String>,
}

impl From<PodRepositoryInfo> for PodRepositoryInfoDto {
    fn from(r: PodRepositoryInfo) -> Self {
        Self {
            id: r.id,
            name: r.name,
            slug: r.slug,
            provider_type: r.provider_type,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodTicketInfoDto {
    pub id: Option<i64>,
    pub slug: Option<String>,
    pub title: Option<String>,
}

impl From<PodTicketInfo> for PodTicketInfoDto {
    fn from(t: PodTicketInfo) -> Self {
        Self {
            id: t.id,
            slug: t.slug,
            title: t.title,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodLoopInfoDto {
    pub id: Option<i64>,
    pub name: Option<String>,
    pub slug: Option<String>,
}

impl From<PodLoopInfo> for PodLoopInfoDto {
    fn from(l: PodLoopInfo) -> Self {
        Self {
            id: l.id,
            name: l.name,
            slug: l.slug,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodCreatedByInfoDto {
    pub id: Option<i64>,
    pub username: Option<String>,
    pub name: Option<String>,
}

impl From<PodCreatedByInfo> for PodCreatedByInfoDto {
    fn from(u: PodCreatedByInfo) -> Self {
        Self {
            id: u.id,
            username: u.username,
            name: u.name,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodDto {
    pub key: String,
    pub id: Option<i64>,
    pub status: PodStatusDto,
    pub agent_status: Option<String>,
    pub alias: Option<String>,
    pub title: Option<String>,
    pub agent_slug: String,
    pub runner_id: Option<i64>,
    pub runner_name: Option<String>,
    pub user_id: Option<i64>,
    pub ticket_slug: Option<String>,
    pub channel_id: Option<i64>,
    pub runner: Option<PodRunnerInfoDto>,
    pub agent: Option<PodAgentInfoDto>,
    pub repository: Option<PodRepositoryInfoDto>,
    pub ticket: Option<PodTicketInfoDto>,
    pub loop_info: Option<PodLoopInfoDto>,
    pub created_by: Option<PodCreatedByInfoDto>,
    pub prompt: Option<String>,
    pub branch_name: Option<String>,
    pub sandbox_path: Option<String>,
    pub started_at: Option<String>,
    pub finished_at: Option<String>,
    pub last_activity: Option<String>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
    pub interaction_mode: Option<String>,
    pub perpetual: Option<bool>,
    pub restart_count: Option<i32>,
    pub last_restart_at: Option<String>,
    pub error_code: Option<String>,
    pub error_message: Option<String>,
}

impl From<Pod> for PodDto {
    fn from(p: Pod) -> Self {
        Self {
            key: p.key,
            id: p.id,
            status: p.status.into(),
            agent_status: p.agent_status,
            alias: p.alias,
            title: p.title,
            agent_slug: p.agent_slug,
            runner_id: p.runner_id,
            runner_name: p.runner_name,
            user_id: p.user_id,
            ticket_slug: p.ticket_slug,
            channel_id: p.channel_id,
            runner: p.runner.map(Into::into),
            agent: p.agent.map(Into::into),
            repository: p.repository.map(Into::into),
            ticket: p.ticket.map(Into::into),
            loop_info: p.loop_info.map(Into::into),
            created_by: p.created_by.map(Into::into),
            prompt: p.prompt,
            branch_name: p.branch_name,
            sandbox_path: p.sandbox_path,
            started_at: p.started_at,
            finished_at: p.finished_at,
            last_activity: p.last_activity,
            created_at: p.created_at,
            updated_at: p.updated_at,
            interaction_mode: p.interaction_mode,
            perpetual: p.perpetual,
            restart_count: p.restart_count,
            last_restart_at: p.last_restart_at,
            error_code: p.error_code,
            error_message: p.error_message,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodConnectionInfoDto {
    pub relay_url: String,
    pub token: String,
    pub pod_key: String,
}

impl From<PodConnectionInfo> for PodConnectionInfoDto {
    fn from(i: PodConnectionInfo) -> Self {
        Self {
            relay_url: i.relay_url,
            token: i.token,
            pod_key: i.pod_key,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreatePodRequestDto {
    pub agent_slug: String,
    pub agentfile_layer: Option<String>,
    pub runner_id: Option<i64>,
    pub alias: Option<String>,
    pub ticket_slug: Option<String>,
    pub cols: Option<u16>,
    pub rows: Option<u16>,
    pub source_pod_key: Option<String>,
    pub resume_agent_session: Option<bool>,
    pub perpetual: Option<bool>,
}

impl From<CreatePodRequestDto> for CreatePodRequest {
    fn from(d: CreatePodRequestDto) -> Self {
        Self {
            agent_slug: d.agent_slug,
            agentfile_layer: d.agentfile_layer,
            runner_id: d.runner_id,
            alias: d.alias,
            ticket_slug: d.ticket_slug,
            cols: d.cols,
            rows: d.rows,
            source_pod_key: d.source_pod_key,
            resume_agent_session: d.resume_agent_session,
            perpetual: d.perpetual,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct PodListResponseDto {
    pub pods: Vec<PodDto>,
    pub total: Option<i64>,
}

impl From<PodListResponse> for PodListResponseDto {
    fn from(r: PodListResponse) -> Self {
        Self {
            pods: r.pods.into_iter().map(PodDto::from).collect(),
            total: r.total,
        }
    }
}

#[derive(Clone, Debug, uniffi::Record)]
pub struct CreatePodResponseDto {
    pub pod: PodDto,
    pub warning: Option<String>,
}

impl From<CreatePodResponse> for CreatePodResponseDto {
    fn from(r: CreatePodResponse) -> Self {
        Self {
            pod: r.pod.into(),
            warning: r.warning,
        }
    }
}

// ── Connect / prost mappings ──
//
// Proto Pod uses required `status: string`; map it through the wire enum
// (PodStatus) so the conversion lives in one place — there's no separate
// dto-level parser.


fn opt_str(s: String) -> Option<String> {
    if s.is_empty() { None } else { Some(s) }
}

impl From<pod_proto::PodRunnerInfo> for PodRunnerInfoDto {
    fn from(r: pod_proto::PodRunnerInfo) -> Self {
        Self { id: r.id, node_id: r.node_id, status: r.status }
    }
}

impl From<pod_proto::PodAgentInfo> for PodAgentInfoDto {
    fn from(a: pod_proto::PodAgentInfo) -> Self {
        Self { name: a.name, slug: a.slug }
    }
}

impl From<pod_proto::PodRepositoryInfo> for PodRepositoryInfoDto {
    fn from(r: pod_proto::PodRepositoryInfo) -> Self {
        Self {
            id: r.id,
            name: r.name,
            slug: r.slug,
            provider_type: r.provider_type,
        }
    }
}

impl From<pod_proto::PodTicketInfo> for PodTicketInfoDto {
    fn from(t: pod_proto::PodTicketInfo) -> Self {
        Self { id: t.id, slug: t.slug, title: t.title }
    }
}

impl From<pod_proto::PodLoopInfo> for PodLoopInfoDto {
    fn from(l: pod_proto::PodLoopInfo) -> Self {
        Self { id: l.id, name: l.name, slug: l.slug }
    }
}

impl From<pod_proto::PodCreatedByInfo> for PodCreatedByInfoDto {
    fn from(u: pod_proto::PodCreatedByInfo) -> Self {
        Self { id: u.id, username: u.username, name: u.name }
    }
}

impl From<pod_proto::Pod> for PodDto {
    fn from(p: pod_proto::Pod) -> Self {
        Self {
            key: p.pod_key,
            id: Some(p.id),
            status: agentsmesh_services::parse_status::<PodStatus>(&p.status).into(),
            agent_status: opt_str(p.agent_status),
            alias: p.alias,
            title: p.title,
            agent_slug: p.agent_slug,
            runner_id: p.runner_id,
            runner_name: p.runner.as_ref().and_then(|r| r.node_id.clone()),
            user_id: p.created_by_id,
            ticket_slug: p.ticket.as_ref().and_then(|t| t.slug.clone()),
            channel_id: None,
            runner: p.runner.map(Into::into),
            agent: p.agent.map(Into::into),
            repository: p.repository.map(Into::into),
            ticket: p.ticket.map(Into::into),
            loop_info: p.r#loop.map(Into::into),
            created_by: p.created_by.map(Into::into),
            prompt: p.prompt,
            branch_name: p.branch_name,
            sandbox_path: p.sandbox_path,
            started_at: p.started_at,
            finished_at: p.finished_at,
            last_activity: p.last_activity,
            created_at: opt_str(p.created_at),
            updated_at: opt_str(p.updated_at),
            interaction_mode: opt_str(p.interaction_mode),
            perpetual: Some(p.perpetual),
            restart_count: Some(p.restart_count),
            last_restart_at: p.last_restart_at,
            error_code: p.error_code,
            error_message: p.error_message,
        }
    }
}

impl From<pod_proto::PodConnectionInfo> for PodConnectionInfoDto {
    fn from(i: pod_proto::PodConnectionInfo) -> Self {
        Self { relay_url: i.relay_url, token: i.token, pod_key: i.pod_key }
    }
}

impl From<pod_proto::ListPodsResponse> for PodListResponseDto {
    fn from(r: pod_proto::ListPodsResponse) -> Self {
        Self {
            pods: r.items.into_iter().map(PodDto::from).collect(),
            total: Some(r.total),
        }
    }
}

impl From<pod_proto::CreatePodResponse> for CreatePodResponseDto {
    fn from(r: pod_proto::CreatePodResponse) -> Self {
        Self {
            pod: r.pod.map(PodDto::from).unwrap_or_else(|| PodDto {
                key: String::new(),
                id: None,
                status: PodStatusDto::Unknown,
                agent_status: None,
                alias: None,
                title: None,
                agent_slug: String::new(),
                runner_id: None,
                runner_name: None,
                user_id: None,
                ticket_slug: None,
                channel_id: None,
                runner: None,
                agent: None,
                repository: None,
                ticket: None,
                loop_info: None,
                created_by: None,
                prompt: None,
                branch_name: None,
                sandbox_path: None,
                started_at: None,
                finished_at: None,
                last_activity: None,
                created_at: None,
                updated_at: None,
                interaction_mode: None,
                perpetual: None,
                restart_count: None,
                last_restart_at: None,
                error_code: None,
                error_message: None,
            }),
            warning: r.warning,
        }
    }
}

pub(crate) fn build_create_pod_proto_request(
    org_slug: String,
    d: CreatePodRequestDto,
) -> pod_proto::CreatePodRequest {
    pod_proto::CreatePodRequest {
        org_slug,
        agent_slug: d.agent_slug,
        runner_id: d.runner_id,
        ticket_slug: d.ticket_slug,
        alias: d.alias,
        agentfile_layer: d.agentfile_layer,
        repository_id: None,
        credential_profile_id: None,
        cols: d.cols.map(i32::from).unwrap_or(0),
        rows: d.rows.map(i32::from).unwrap_or(0),
        source_pod_key: d.source_pod_key,
        resume_agent_session: d.resume_agent_session,
        perpetual: d.perpetual,
    }
}
