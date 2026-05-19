use serde::{Deserialize, Serialize};

use crate::PodStatus;

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PodRunnerInfo {
    pub id: Option<i64>,
    #[serde(default)]
    pub node_id: Option<String>,
    #[serde(default)]
    pub status: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PodAgentInfo {
    #[serde(default)]
    pub name: Option<String>,
    #[serde(default)]
    pub slug: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PodRepositoryInfo {
    pub id: Option<i64>,
    #[serde(default)]
    pub name: Option<String>,
    #[serde(default)]
    pub slug: Option<String>,
    #[serde(default)]
    pub provider_type: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PodTicketInfo {
    pub id: Option<i64>,
    #[serde(default)]
    pub slug: Option<String>,
    #[serde(default)]
    pub title: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PodLoopInfo {
    pub id: Option<i64>,
    #[serde(default)]
    pub name: Option<String>,
    #[serde(default)]
    pub slug: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PodCreatedByInfo {
    pub id: Option<i64>,
    #[serde(default)]
    pub username: Option<String>,
    #[serde(default)]
    pub name: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Pod {
    #[serde(rename = "pod_key", alias = "key")]
    pub key: String,
    #[serde(default)]
    pub id: Option<i64>,
    #[serde(default)]
    pub status: PodStatus,
    #[serde(default)]
    pub agent_status: Option<String>,

    #[serde(default)]
    pub alias: Option<String>,
    #[serde(default)]
    pub title: Option<String>,

    #[serde(default)]
    pub agent_slug: String,
    #[serde(default)]
    pub runner_id: Option<i64>,
    #[serde(default)]
    pub runner_name: Option<String>,
    #[serde(default)]
    pub user_id: Option<i64>,
    #[serde(default)]
    pub ticket_slug: Option<String>,
    #[serde(default)]
    pub channel_id: Option<i64>,

    #[serde(default)]
    pub runner: Option<PodRunnerInfo>,
    #[serde(default)]
    pub agent: Option<PodAgentInfo>,
    #[serde(default)]
    pub repository: Option<PodRepositoryInfo>,
    #[serde(default)]
    pub ticket: Option<PodTicketInfo>,
    #[serde(default, rename = "loop")]
    pub loop_info: Option<PodLoopInfo>,
    #[serde(default)]
    pub created_by: Option<PodCreatedByInfo>,

    #[serde(default)]
    pub prompt: Option<String>,
    #[serde(default)]
    pub branch_name: Option<String>,
    #[serde(default)]
    pub sandbox_path: Option<String>,
    #[serde(default)]
    pub started_at: Option<String>,
    #[serde(default)]
    pub finished_at: Option<String>,
    #[serde(default)]
    pub last_activity: Option<String>,
    #[serde(default)]
    pub created_at: Option<String>,
    #[serde(default)]
    pub updated_at: Option<String>,
    #[serde(default)]
    pub interaction_mode: Option<String>,
    #[serde(default)]
    pub perpetual: Option<bool>,
    #[serde(default)]
    pub restart_count: Option<i32>,
    #[serde(default)]
    pub last_restart_at: Option<String>,

    #[serde(default)]
    pub error_code: Option<String>,
    #[serde(default)]
    pub error_message: Option<String>,
}

impl Default for Pod {
    fn default() -> Self {
        Self {
            key: String::new(),
            id: None,
            status: PodStatus::default(),
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
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PodConnectionInfo {
    pub relay_url: String,
    pub token: String,
    pub pod_key: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub local_relay_url: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub local_token: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub local_relay_node_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreatePodRequest {
    #[serde(default)]
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreatePodResponse {
    pub pod: Pod,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub warning: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdatePodAliasRequest {
    pub alias: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PodListResponse {
    pub pods: Vec<Pod>,
    pub total: Option<i64>,
    #[serde(default)]
    pub limit: Option<i64>,
    #[serde(default)]
    pub offset: Option<i64>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json;

    #[test]
    fn pod_roundtrip() {
        let pod = Pod {
            key: "pod-abc".into(),
            alias: Some("my-pod".into()),
            status: PodStatus::Running,
            agent_status: Some("idle".into()),
            agent_slug: "claude-code".into(),
            runner_id: Some(1),
            runner_name: Some("runner-1".into()),
            user_id: Some(10),
            created_at: Some("2026-01-01T00:00:00Z".into()),
            ..Default::default()
        };
        let json = serde_json::to_string(&pod).unwrap();
        let decoded: Pod = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.key, "pod-abc");
        assert_eq!(decoded.agent_slug, "claude-code");
        assert_eq!(decoded.runner_id, Some(1));
        assert_eq!(decoded.status, PodStatus::Running);
    }

    #[test]
    fn pod_minimal_json() {
        let json = r#"{
            "key":"k","status":"pending","agent_slug":"aider"
        }"#;
        let pod: Pod = serde_json::from_str(json).unwrap();
        assert_eq!(pod.key, "k");
        assert_eq!(pod.status, PodStatus::Pending);
        assert!(pod.alias.is_none());
        assert!(pod.runner_id.is_none());
        assert!(pod.error_code.is_none());
    }

    #[test]
    fn pod_connection_info_roundtrip() {
        let info = PodConnectionInfo {
            relay_url: "wss://relay.example.com".into(),
            token: "tok-123".into(),
            pod_key: "pod-1".into(),
            local_relay_url: String::new(),
            local_token: String::new(),
            local_relay_node_id: String::new(),
        };
        let json = serde_json::to_string(&info).unwrap();
        let decoded: PodConnectionInfo = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.relay_url, "wss://relay.example.com");
        assert!(decoded.local_relay_url.is_empty());
    }

    #[test]
    fn create_pod_request_serialization() {
        let req = CreatePodRequest {
            agent_slug: "claude-code".into(),
            agentfile_layer: Some("AGENT claude-code".into()),
            runner_id: None,
            alias: Some("test".into()),
            ticket_slug: None,
            cols: Some(80),
            rows: Some(24),
            source_pod_key: None,
            resume_agent_session: None,
            perpetual: None,
        };
        let json = serde_json::to_string(&req).unwrap();
        assert!(json.contains("\"agent_slug\":\"claude-code\""));
        assert!(json.contains("\"cols\":80"));
    }

    #[test]
    fn create_pod_request_minimal() {
        let json = r#"{"agent_slug":"aider"}"#;
        let req: CreatePodRequest = serde_json::from_str(json).unwrap();
        assert_eq!(req.agent_slug, "aider");
        assert!(req.agentfile_layer.is_none());
        assert!(req.cols.is_none());
    }

    #[test]
    fn create_pod_request_resume_without_agent_slug() {
        let json = r#"{
            "runner_id":1,
            "source_pod_key":"pod-source",
            "resume_agent_session":true,
            "cols":120,
            "rows":30
        }"#;
        let req: CreatePodRequest = serde_json::from_str(json).unwrap();
        assert!(req.agent_slug.is_empty());
        assert_eq!(req.runner_id, Some(1));
        assert_eq!(req.source_pod_key.as_deref(), Some("pod-source"));
        assert_eq!(req.resume_agent_session, Some(true));
    }

    #[test]
    fn pod_list_response_roundtrip() {
        let resp = PodListResponse {
            pods: vec![Pod {
                key: "p1".into(),
                status: PodStatus::Running,
                agent_slug: "claude".into(),
                ..Default::default()
            }],
            total: Some(1),
            limit: None,
            offset: None,
        };
        let json = serde_json::to_string(&resp).unwrap();
        let decoded: PodListResponse = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.pods.len(), 1);
        assert_eq!(decoded.total, Some(1));
    }

    #[test]
    fn pod_list_response_empty() {
        let json = r#"{"pods":[]}"#;
        let resp: PodListResponse = serde_json::from_str(json).unwrap();
        assert!(resp.pods.is_empty());
        assert!(resp.total.is_none());
    }

    #[test]
    fn pod_from_nested_api_json() {
        let json = r#"{
            "id": 42,
            "pod_key": "pod-xyz",
            "status": "initializing",
            "agent_status": "executing",
            "title": "My Pod",
            "alias": "dev-pod",
            "runner": {"id": 5, "node_id": "runner-5", "status": "online"},
            "agent": {"name": "Claude Code", "slug": "claude-code"},
            "repository": {"id": 10, "name": "my-repo", "slug": "my-repo", "provider_type": "github"},
            "ticket": {"id": 3, "slug": "TK-3", "title": "Fix bug"},
            "loop": {"id": 1, "name": "CI Loop", "slug": "ci-loop"},
            "created_by": {"id": 7, "username": "alice", "name": "Alice"},
            "interaction_mode": "acp",
            "perpetual": true,
            "created_at": "2026-01-01T00:00:00Z"
        }"#;
        let pod: Pod = serde_json::from_str(json).unwrap();
        assert_eq!(pod.key, "pod-xyz");
        assert_eq!(pod.id, Some(42));
        assert_eq!(pod.status, PodStatus::Initializing);
        assert_eq!(pod.runner.as_ref().unwrap().id, Some(5));
        assert_eq!(pod.agent.as_ref().unwrap().slug.as_deref(), Some("claude-code"));
        assert_eq!(pod.ticket.as_ref().unwrap().title.as_deref(), Some("Fix bug"));
        assert_eq!(pod.loop_info.as_ref().unwrap().name.as_deref(), Some("CI Loop"));
        assert_eq!(pod.created_by.as_ref().unwrap().username.as_deref(), Some("alice"));
        assert_eq!(pod.interaction_mode.as_deref(), Some("acp"));
        assert_eq!(pod.perpetual, Some(true));
    }
}
