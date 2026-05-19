use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillRegistry {
    pub id: i64,
    pub organization_id: Option<i64>,
    pub repository_url: Option<String>,
    pub branch: Option<String>,
    pub source_type: Option<String>,
    pub detected_type: Option<String>,
    pub compatible_agents: Option<Vec<String>>,
    pub auth_type: Option<String>,
    pub last_synced_at: Option<String>,
    pub last_commit_sha: Option<String>,
    pub sync_status: Option<String>,
    pub sync_error: Option<String>,
    pub skill_count: Option<i64>,
    pub is_active: Option<bool>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
    /// Legacy alias for `is_active` — kept readable so older clients sending
    /// `is_disabled` continue to round-trip during a partial rollout.
    #[serde(default)]
    pub is_disabled: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateSkillRegistryRequest {
    pub repository_url: String,
    pub branch: Option<String>,
    pub source_type: Option<String>,
    pub compatible_agents: Option<Vec<String>>,
    pub auth_type: Option<String>,
    pub auth_credential: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToggleRegistryRequest {
    pub disabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillRegistryOverride {
    pub id: i64,
    pub registry_id: Option<i64>,
    pub skill_slug: Option<String>,
    pub is_disabled: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketSkill {
    pub id: i64,
    pub slug: Option<String>,
    #[serde(default, alias = "name")]
    pub display_name: Option<String>,
    pub description: Option<String>,
    pub category: Option<String>,
    pub author: Option<String>,
    pub icon_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketMcpServer {
    pub id: i64,
    pub name: String,
    pub slug: Option<String>,
    pub description: Option<String>,
    pub category: Option<String>,
    pub author: Option<String>,
    pub transport_type: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepoSkillInstall {
    pub id: i64,
    pub repository_id: Option<i64>,
    pub skill_slug: Option<String>,
    pub name: Option<String>,
    pub scope: Option<String>,
    pub is_enabled: Option<bool>,
    pub pinned_version: Option<String>,
    pub source: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstallMarketSkillRequest {
    pub market_item_id: i64,
    pub scope: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstallGithubSkillRequest {
    pub url: String,
    pub branch: Option<String>,
    pub path: Option<String>,
    pub scope: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateSkillInstallRequest {
    pub is_enabled: Option<bool>,
    pub pinned_version: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepoMcpServerInstall {
    pub id: i64,
    pub repository_id: Option<i64>,
    pub name: Option<String>,
    pub slug: Option<String>,
    pub transport_type: Option<String>,
    pub scope: Option<String>,
    pub is_enabled: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstallMarketMcpRequest {
    pub market_item_id: i64,
    pub env_vars: Option<serde_json::Value>,
    pub scope: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstallCustomMcpRequest {
    pub name: String,
    pub slug: Option<String>,
    pub transport_type: Option<String>,
    pub command: Option<String>,
    pub args: Option<Vec<String>>,
    pub http_url: Option<String>,
    pub http_headers: Option<serde_json::Value>,
    pub env_vars: Option<serde_json::Value>,
    pub scope: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateMcpInstallRequest {
    pub is_enabled: Option<bool>,
    pub env_vars: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillRegistryListResponse {
    pub skill_registries: Vec<SkillRegistry>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketSkillListResponse {
    pub skills: Vec<MarketSkill>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketMcpServerListResponse {
    pub mcp_servers: Vec<MarketMcpServer>,
    pub total: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SkillRegistryOverrideListResponse {
    pub overrides: Vec<SkillRegistryOverride>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepoSkillInstallListResponse {
    pub skills: Vec<RepoSkillInstall>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepoMcpServerInstallListResponse {
    pub mcp_servers: Vec<RepoMcpServerInstall>,
}

#[cfg(test)]
mod skill_registry_tests {
    use super::{SkillRegistry, SkillRegistryListResponse};

    const BACKEND_PAYLOAD: &str = r#"{
        "id": 7,
        "organization_id": 42,
        "repository_url": "https://github.com/agentsmesh/skills-bundle",
        "branch": "main",
        "source_type": "auto",
        "detected_type": "collection",
        "compatible_agents": ["claude-code", "codex"],
        "auth_type": "github_pat",
        "last_synced_at": "2026-05-09T00:00:00Z",
        "last_commit_sha": "abc123def456",
        "sync_status": "success",
        "skill_count": 12,
        "is_active": true,
        "created_at": "2026-05-08T00:00:00Z",
        "updated_at": "2026-05-09T00:00:00Z"
    }"#;

    #[test]
    fn skill_registry_decodes_backend_payload() {
        let r: SkillRegistry = serde_json::from_str(BACKEND_PAYLOAD).unwrap();
        assert_eq!(r.id, 7);
        assert_eq!(r.sync_status.as_deref(), Some("success"));
        assert_eq!(r.skill_count, Some(12));
        assert_eq!(r.auth_type.as_deref(), Some("github_pat"));
        assert_eq!(r.is_active, Some(true));
    }

    #[test]
    fn skill_registry_wasm_relay_preserves_ui_fields() {
        let typed: SkillRegistry = serde_json::from_str(BACKEND_PAYLOAD).unwrap();
        let relayed = serde_json::to_string(&typed).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
        for key in ["sync_status", "auth_type", "skill_count", "is_active",
                    "last_commit_sha", "detected_type"] {
            assert!(!parsed[key].is_null(), "field `{key}` dropped by relay");
        }
    }

    // Regression for #341: backend wire wrapper is `skill_registries`, but PR #349
    // used `#[serde(alias)]` on a Rust field named `registries`. Serde's `alias`
    // only affects deserialization; re-serializing for the wasm relay emitted
    // `{"registries": ...}` instead, so the TS layer (which reads
    // `.skill_registries`) always saw `undefined` and rendered an empty list.
    // This test pins the round-trip key.
    #[test]
    fn skill_registry_list_response_relay_key_matches_backend_wire() {
        let backend_wire = format!(r#"{{"skill_registries":[{BACKEND_PAYLOAD}]}}"#);
        let typed: SkillRegistryListResponse = serde_json::from_str(&backend_wire).unwrap();
        assert_eq!(typed.skill_registries.len(), 1);

        let relayed = serde_json::to_string(&typed).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
        assert!(
            parsed.get("skill_registries").is_some(),
            "wasm relay must emit `skill_registries`, got: {relayed}"
        );
        assert!(
            parsed.get("registries").is_none(),
            "legacy `registries` key must not leak through: {relayed}"
        );
    }
}
