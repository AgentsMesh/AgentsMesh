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
    pub is_disabled: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpHeaderSchemaEntry {
    pub name: String,
    pub description: Option<String>,
    pub value: Option<String>,
    pub required: bool,
    pub sensitive: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnvVarSchemaEntry {
    pub name: String,
    pub label: String,
    pub required: bool,
    pub sensitive: bool,
    pub placeholder: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketSkill {
    pub id: i64,
    pub registry_id: Option<i64>,
    pub slug: Option<String>,
    #[serde(default, alias = "name")]
    pub display_name: Option<String>,
    pub description: Option<String>,
    pub license: Option<String>,
    pub category: Option<String>,
    pub content_sha: Option<String>,
    pub version: Option<i64>,
    pub is_active: Option<bool>,
    pub registry: Option<SkillRegistry>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MarketMcpServer {
    pub id: i64,
    pub name: String,
    pub slug: Option<String>,
    pub description: Option<String>,
    pub icon: Option<String>,
    pub transport_type: Option<String>,
    pub command: Option<String>,
    pub default_args: Option<Vec<String>>,
    pub default_http_url: Option<String>,
    pub default_http_headers: Option<Vec<McpHeaderSchemaEntry>>,
    pub env_var_schema: Option<Vec<EnvVarSchemaEntry>>,
    pub category: Option<String>,
    pub source: Option<String>,
    pub registry_name: Option<String>,
    pub version: Option<String>,
    pub repository_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepoSkillInstall {
    pub id: i64,
    pub organization_id: Option<i64>,
    pub market_item_id: Option<i64>,
    pub installed_by: Option<i64>,
    pub slug: Option<String>,
    pub scope: Option<String>,
    pub install_source: Option<String>,
    pub source_url: Option<String>,
    pub content_sha: Option<String>,
    pub package_size: Option<i64>,
    pub pinned_version: Option<String>,
    pub is_enabled: Option<bool>,
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
    pub organization_id: Option<i64>,
    pub market_item_id: Option<i64>,
    pub installed_by: Option<i64>,
    pub name: Option<String>,
    pub slug: Option<String>,
    pub transport_type: Option<String>,
    pub command: Option<String>,
    pub args: Option<Vec<String>>,
    pub http_url: Option<String>,
    pub http_headers: Option<serde_json::Value>,
    pub env_vars: Option<serde_json::Value>,
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
#[path = "extension_relay_tests.rs"]
mod extension_relay_tests;
