// Proto → legacy serde-shape conversions for the dual-track Connect
// migration. Service public method signatures (e.g., `fetch_pods`,
// `terminate_pod`) preserve the legacy JSON wire shape so wasm bridge / iOS
// FFI / node-bridge consumers don't change. The internal implementation
// calls Connect-RPC and routes the prost response through these converters
// back to the legacy shape.

use agentsmesh_types::proto_pod_v1 as pod_proto;
use agentsmesh_types::proto_repository_v1 as repo_proto;
use agentsmesh_types::proto_runner_api_v1 as runner_proto;
use agentsmesh_types::{
    Pod, PodAgentInfo, PodConnectionInfo, PodCreatedByInfo, PodLoopInfo,
    PodRepositoryInfo, PodRunnerInfo, PodTicketInfo, Repository, Runner,
};

pub mod pod {
    use super::*;

    pub fn from_proto(p: pod_proto::Pod) -> Pod {
        Pod {
            id: Some(p.id),
            key: p.pod_key,
            status: super::parse_status(&p.status),
            agent_status: super::option_string(&p.agent_status),
            alias: p.alias,
            title: p.title,
            agent_slug: p.agent_slug,
            runner_id: p.runner_id,
            runner_name: p.runner.as_ref().and_then(|r| r.node_id.clone()),
            user_id: p.created_by_id,
            ticket_slug: p.ticket.as_ref().and_then(|t| t.slug.clone()),
            channel_id: None,
            runner: p.runner.map(|r| PodRunnerInfo {
                id: r.id,
                node_id: r.node_id,
                status: r.status,
            }),
            agent: p.agent.map(|a| PodAgentInfo { name: a.name, slug: a.slug }),
            repository: p.repository.map(|r| PodRepositoryInfo {
                id: r.id,
                name: r.name,
                slug: r.slug,
                provider_type: r.provider_type,
            }),
            ticket: p.ticket.map(|t| PodTicketInfo {
                id: t.id,
                slug: t.slug,
                title: t.title,
            }),
            loop_info: p.r#loop.map(|l| PodLoopInfo { id: l.id, name: l.name, slug: l.slug }),
            created_by: p.created_by.map(|c| PodCreatedByInfo {
                id: c.id,
                username: c.username,
                name: c.name,
            }),
            prompt: p.prompt,
            branch_name: p.branch_name,
            sandbox_path: p.sandbox_path,
            started_at: p.started_at,
            finished_at: p.finished_at,
            last_activity: p.last_activity,
            created_at: super::option_string(&p.created_at),
            updated_at: super::option_string(&p.updated_at),
            interaction_mode: super::option_string(&p.interaction_mode),
            perpetual: Some(p.perpetual),
            restart_count: Some(p.restart_count),
            last_restart_at: p.last_restart_at,
            error_code: p.error_code,
            error_message: p.error_message,
        }
    }

    pub fn connection_info(info: pod_proto::PodConnectionInfo) -> PodConnectionInfo {
        PodConnectionInfo {
            relay_url: info.relay_url,
            token: info.token,
            pod_key: info.pod_key,
            local_relay_url: info.local_relay_url,
            local_token: info.local_token,
            local_relay_node_id: info.local_relay_node_id,
        }
    }
}

pub mod runner {
    use super::*;

    pub fn from_proto(r: runner_proto::Runner) -> Runner {
        Runner {
            id: r.id,
            // Legacy `name` is best-effort: REST emits the `name` column from
            // backend `runners`. The proto sheds the legacy `name` in favor of
            // `node_id`, so we surface node_id as the display name here — keeps
            // existing state cache consumers (RunnerListItem) rendering.
            name: r.node_id.clone(),
            node_id: super::option_string(&r.node_id),
            description: super::option_string(&r.description),
            status: super::parse_status(&r.status),
            version: r.runner_version,
            max_concurrent_pods: r.max_concurrent_pods,
            active_pod_count: r.current_pods,
            is_enabled: r.is_enabled,
            host_info: if r.host_info_json.is_empty() {
                None
            } else {
                serde_json::from_str(&r.host_info_json).ok()
            },
            last_heartbeat: r.last_heartbeat,
            available_agents: if r.available_agents.is_empty() {
                None
            } else {
                Some(r.available_agents)
            },
            created_at: super::option_string(&r.created_at),
            updated_at: super::option_string(&r.updated_at),
        }
    }
}

pub mod repository {
    use super::*;

    pub fn from_proto(r: repo_proto::Repository) -> Repository {
        Repository {
            id: r.id,
            name: r.name,
            slug: super::option_string(&r.slug),
            provider_type: super::option_string(&r.provider_type),
            provider_base_url: super::option_string(&r.provider_base_url),
            http_clone_url: super::option_string(&r.http_clone_url),
            ssh_clone_url: super::option_string(&r.ssh_clone_url),
            external_id: super::option_string(&r.external_id),
            default_branch: super::option_string(&r.default_branch),
            ticket_prefix: r.ticket_prefix,
            visibility: super::option_string(&r.visibility),
            is_active: Some(r.is_active),
            created_at: super::option_string(&r.created_at),
            updated_at: super::option_string(&r.updated_at),
        }
    }
}

fn option_string(s: &str) -> Option<String> {
    if s.is_empty() { None } else { Some(s.to_string()) }
}

fn parse_status<T: serde::de::DeserializeOwned + Default>(s: &str) -> T {
    serde_json::from_value(serde_json::Value::String(s.to_string())).unwrap_or_default()
}
