// Hand-maintained `prost::Message` mirrors of
// `proto/extension/v1/market.proto`. Tag numbers match the .proto
// byte-for-byte; `tools/validate_prost_tags` runs at build time to catch
// drift (watch list §8). NO `Serialize`/`Deserialize` derives — binary
// wire only (conventions §2.5, §3).

#[derive(Clone, PartialEq, prost::Message)]
pub struct SkillMarketItem {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub registry_id: i64,
    #[prost(string, tag = "3")]
    pub slug: String,
    #[prost(string, tag = "4")]
    pub display_name: String,
    #[prost(string, tag = "5")]
    pub description: String,
    #[prost(string, tag = "6")]
    pub license: String,
    #[prost(string, tag = "7")]
    pub compatibility: String,
    #[prost(string, tag = "8")]
    pub allowed_tools: String,
    #[prost(string, tag = "9")]
    pub category: String,
    #[prost(string, tag = "10")]
    pub content_sha: String,
    #[prost(string, tag = "11")]
    pub storage_key: String,
    #[prost(int64, tag = "12")]
    pub package_size: i64,
    #[prost(int32, tag = "13")]
    pub version: i32,
    #[prost(string, repeated, tag = "14")]
    pub agent_filter: Vec<String>,
    #[prost(bool, tag = "15")]
    pub is_active: bool,
    #[prost(string, tag = "16")]
    pub created_at: String,
    #[prost(string, tag = "17")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct McpEnvVarSchemaEntry {
    #[prost(string, tag = "1")]
    pub name: String,
    #[prost(string, tag = "2")]
    pub label: String,
    #[prost(bool, tag = "3")]
    pub required: bool,
    #[prost(bool, tag = "4")]
    pub sensitive: bool,
    #[prost(string, tag = "5")]
    pub placeholder: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct McpMarketItem {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub slug: String,
    #[prost(string, tag = "3")]
    pub name: String,
    #[prost(string, tag = "4")]
    pub description: String,
    #[prost(string, tag = "5")]
    pub icon: String,
    #[prost(string, tag = "6")]
    pub transport_type: String,
    #[prost(string, tag = "7")]
    pub command: String,
    #[prost(string, tag = "8")]
    pub default_args: String,
    #[prost(string, tag = "9")]
    pub default_http_url: String,
    #[prost(string, tag = "10")]
    pub default_http_headers: String,
    #[prost(message, repeated, tag = "11")]
    pub env_var_schema: Vec<McpEnvVarSchemaEntry>,
    #[prost(string, repeated, tag = "12")]
    pub agent_filter: Vec<String>,
    #[prost(string, tag = "13")]
    pub category: String,
    #[prost(bool, tag = "14")]
    pub is_active: bool,
    #[prost(string, tag = "15")]
    pub source: String,
    #[prost(string, tag = "16")]
    pub registry_name: String,
    #[prost(string, tag = "17")]
    pub version: String,
    #[prost(string, tag = "18")]
    pub repository_url: String,
    #[prost(string, tag = "19")]
    pub registry_meta: String,
    #[prost(string, optional, tag = "20")]
    pub last_synced_at: Option<String>,
    #[prost(string, tag = "21")]
    pub created_at: String,
    #[prost(string, tag = "22")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListMarketSkillsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, optional, tag = "2")]
    pub query: Option<String>,
    #[prost(string, optional, tag = "3")]
    pub category: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListMarketSkillsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<SkillMarketItem>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListMarketMcpServersRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(string, optional, tag = "2")]
    pub query: Option<String>,
    #[prost(string, optional, tag = "3")]
    pub category: Option<String>,
    #[prost(int32, optional, tag = "4")]
    pub offset: Option<i32>,
    #[prost(int32, optional, tag = "5")]
    pub limit: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListMarketMcpServersResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<McpMarketItem>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    fn sample_skill() -> SkillMarketItem {
        SkillMarketItem {
            id: 11,
            registry_id: 7,
            slug: "format-go".into(),
            display_name: "Format Go".into(),
            description: "Skill description".into(),
            license: "MIT".into(),
            compatibility: "go".into(),
            allowed_tools: "Bash".into(),
            category: "code".into(),
            content_sha: "abc123".into(),
            storage_key: "skills/format-go.zip".into(),
            package_size: 4096,
            version: 2,
            agent_filter: vec!["claude-code".into()],
            is_active: true,
            created_at: "2026-05-01T00:00:00Z".into(),
            updated_at: "2026-05-12T00:00:00Z".into(),
        }
    }

    #[test]
    fn skill_market_item_round_trip_preserves_every_field() {
        let original = sample_skill();
        let bytes = original.encode_to_vec();
        let decoded = SkillMarketItem::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn mcp_market_item_round_trip_preserves_every_field() {
        let original = McpMarketItem {
            id: 21,
            slug: "github".into(),
            name: "GitHub MCP".into(),
            description: "MCP for GitHub".into(),
            icon: "github".into(),
            transport_type: "stdio".into(),
            command: "npx".into(),
            default_args: r#"["@modelcontextprotocol/server-github"]"#.into(),
            default_http_url: "".into(),
            default_http_headers: "[]".into(),
            env_var_schema: vec![McpEnvVarSchemaEntry {
                name: "GITHUB_TOKEN".into(),
                label: "GitHub Token".into(),
                required: true,
                sensitive: true,
                placeholder: "ghp_xxx".into(),
            }],
            agent_filter: vec!["claude-code".into()],
            category: "vcs".into(),
            is_active: true,
            source: "seed".into(),
            registry_name: "official".into(),
            version: "1.0.0".into(),
            repository_url: "https://github.com/modelcontextprotocol/servers".into(),
            registry_meta: "{}".into(),
            last_synced_at: Some("2026-05-12T13:16:10Z".into()),
            created_at: "2026-05-01T00:00:00Z".into(),
            updated_at: "2026-05-12T00:00:00Z".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = McpMarketItem::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn list_skills_request_optional_fields_distinguishable() {
        let with_q = ListMarketSkillsRequest {
            org_slug: "acme".into(),
            query: Some("".into()),
            category: None,
        };
        let absent = ListMarketSkillsRequest {
            org_slug: "acme".into(),
            query: None,
            category: None,
        };
        assert_ne!(with_q.encode_to_vec(), absent.encode_to_vec());
    }

    #[test]
    fn list_mcp_servers_response_round_trip() {
        let original = ListMarketMcpServersResponse {
            items: vec![],
            total: 0,
            limit: 50,
            offset: 0,
        };
        let bytes = original.encode_to_vec();
        let decoded = ListMarketMcpServersResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }
}
