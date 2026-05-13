use serde::{Deserialize, Serialize};

// Connect-RPC binary-wire DTOs for proto.extension.v1 live in extension_proto.rs,
// re-exported through lib.rs as `pub mod proto_extension_v1`. The legacy serde
// types below remain for the REST handlers that haven't been migrated yet (repo
// skills/MCP install paths — those are flipping to Connect too).

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
pub struct RepoSkillInstallListResponse {
    pub skills: Vec<RepoSkillInstall>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepoMcpServerInstallListResponse {
    pub mcp_servers: Vec<RepoMcpServerInstall>,
}
