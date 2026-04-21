use serde::{Deserialize, Serialize};

use crate::RunnerStatus;

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Runner {
    pub id: i64,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub node_id: Option<String>,
    #[serde(default)]
    pub description: Option<String>,
    #[serde(default)]
    pub status: RunnerStatus,
    #[serde(default, rename = "runner_version")]
    pub version: Option<String>,
    #[serde(default)]
    pub max_concurrent_pods: i32,
    #[serde(default, rename = "current_pods")]
    pub active_pod_count: i32,
    #[serde(default)]
    pub is_enabled: bool,
    #[serde(default)]
    pub host_info: Option<serde_json::Value>,
    #[serde(default)]
    pub last_heartbeat: Option<String>,
    #[serde(default)]
    pub available_agents: Option<Vec<String>>,
    #[serde(default)]
    pub created_at: Option<String>,
    #[serde(default)]
    pub updated_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GRPCRegistrationToken {
    pub id: i64,
    pub name: Option<String>,
    pub token: Option<String>,
    pub max_uses: Option<i32>,
    pub used_count: Option<i32>,
    pub expires_at: Option<String>,
    pub created_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateRunnerRequest {
    pub description: Option<String>,
    pub max_concurrent_pods: Option<i32>,
    pub is_enabled: Option<bool>,
    pub visibility: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateRunnerTokenRequest {
    pub name: Option<String>,
    pub labels: Option<Vec<String>>,
    pub max_uses: Option<i32>,
    pub expires_in_days: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SandboxQueryRequest {
    pub pod_keys: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpgradeRunnerRequest {
    pub target_version: Option<String>,
    pub force: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthorizeRunnerRequest {
    pub auth_key: String,
    pub node_id: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerListResponse {
    #[serde(default)]
    pub runners: Vec<Runner>,
    #[serde(default)]
    pub latest_runner_version: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerTokenListResponse {
    pub tokens: Vec<GRPCRegistrationToken>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerAuthStatus {
    pub status: String,
    pub runner_id: Option<i64>,
    pub organization_slug: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerLog {
    pub id: i64,
    pub runner_id: i64,
    pub filename: Option<String>,
    pub url: Option<String>,
    pub created_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerLogListResponse {
    pub logs: Vec<RunnerLog>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json;

    #[test]
    fn runner_roundtrip() {
        let runner = Runner {
            id: 1,
            name: "my-runner".into(),
            status: RunnerStatus::Online,
            version: Some("0.5.0".into()),
            max_concurrent_pods: 4,
            active_pod_count: 2,
            is_enabled: true,
            host_info: Some(serde_json::json!({"os": "linux"})),
            created_at: Some("2026-01-01T00:00:00Z".into()),
            updated_at: None,
            ..Default::default()
        };
        let json = serde_json::to_string(&runner).unwrap();
        let decoded: Runner = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.name, "my-runner");
        assert_eq!(decoded.max_concurrent_pods, 4);
        assert!(decoded.is_enabled);
    }

    #[test]
    fn runner_minimal_json() {
        let json = r#"{
            "id":1,"name":"r","status":"offline",
            "max_concurrent_pods":2,"active_pod_count":0,"is_enabled":false
        }"#;
        let runner: Runner = serde_json::from_str(json).unwrap();
        assert_eq!(runner.id, 1);
        assert!(!runner.is_enabled);
        assert_eq!(runner.status, RunnerStatus::Offline);
        assert!(runner.version.is_none());
        assert!(runner.host_info.is_none());
    }

    #[test]
    fn runner_host_info_json_value() {
        let runner = Runner {
            id: 1,
            name: "r".into(),
            status: RunnerStatus::Online,
            version: None,
            max_concurrent_pods: 1,
            active_pod_count: 0,
            is_enabled: true,
            host_info: Some(serde_json::json!({"arch": "arm64", "cores": 8})),
            created_at: None,
            updated_at: None,
            ..Default::default()
        };
        let json = serde_json::to_string(&runner).unwrap();
        let decoded: Runner = serde_json::from_str(&json).unwrap();
        let info = decoded.host_info.unwrap();
        assert_eq!(info["arch"], "arm64");
        assert_eq!(info["cores"], 8);
    }

    #[test]
    fn grpc_registration_token_roundtrip() {
        let tok = GRPCRegistrationToken {
            id: 5,
            name: Some("dev-token".into()),
            token: Some("grpc-tok-abc".into()),
            max_uses: Some(10),
            used_count: Some(3),
            expires_at: Some("2026-12-31T23:59:59Z".into()),
            created_at: Some("2026-01-01T00:00:00Z".into()),
        };
        let json = serde_json::to_string(&tok).unwrap();
        let decoded: GRPCRegistrationToken = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.id, 5);
        assert_eq!(decoded.max_uses, Some(10));
    }

    #[test]
    fn grpc_registration_token_minimal() {
        let json = r#"{"id":1}"#;
        let tok: GRPCRegistrationToken = serde_json::from_str(json).unwrap();
        assert_eq!(tok.id, 1);
        assert!(tok.name.is_none());
        assert!(tok.token.is_none());
    }
}
