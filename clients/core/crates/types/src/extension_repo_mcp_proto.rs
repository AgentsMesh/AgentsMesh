// Hand-maintained `prost::Message` mirrors of
// `proto/extension/v1/repo_mcp.proto`. Tag numbers match the .proto
// byte-for-byte; `tools/validate_prost_tags` runs at build time to catch
// drift (watch list §8). NO `Serialize`/`Deserialize` derives — binary
// wire only (conventions §2.5, §3).

#[derive(Clone, PartialEq, prost::Message)]
pub struct InstalledMcpServer {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(int64, tag = "2")]
    pub organization_id: i64,
    #[prost(int64, tag = "3")]
    pub repository_id: i64,
    #[prost(int64, optional, tag = "4")]
    pub market_item_id: Option<i64>,
    #[prost(string, tag = "5")]
    pub scope: String,
    #[prost(int64, optional, tag = "6")]
    pub installed_by: Option<i64>,
    #[prost(string, tag = "7")]
    pub name: String,
    #[prost(string, tag = "8")]
    pub slug: String,
    #[prost(string, tag = "9")]
    pub transport_type: String,
    #[prost(string, tag = "10")]
    pub command: String,
    #[prost(string, tag = "11")]
    pub args: String,
    #[prost(string, tag = "12")]
    pub http_url: String,
    #[prost(string, tag = "13")]
    pub http_headers: String,
    #[prost(string, tag = "14")]
    pub env_vars: String,
    #[prost(bool, tag = "15")]
    pub is_enabled: bool,
    #[prost(string, tag = "16")]
    pub created_at: String,
    #[prost(string, tag = "17")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListRepoMcpServersRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(string, optional, tag = "3")]
    pub scope: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListRepoMcpServersResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<InstalledMcpServer>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct InstallMcpFromMarketRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(int64, tag = "3")]
    pub market_item_id: i64,
    #[prost(string, tag = "4")]
    pub scope: String,
    #[prost(string, optional, tag = "5")]
    pub env_vars: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct InstallCustomMcpServerRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(string, tag = "3")]
    pub name: String,
    #[prost(string, tag = "4")]
    pub slug: String,
    #[prost(string, tag = "5")]
    pub transport_type: String,
    #[prost(string, optional, tag = "6")]
    pub command: Option<String>,
    #[prost(string, optional, tag = "7")]
    pub args: Option<String>,
    #[prost(string, optional, tag = "8")]
    pub http_url: Option<String>,
    #[prost(string, optional, tag = "9")]
    pub http_headers: Option<String>,
    #[prost(string, tag = "10")]
    pub scope: String,
    #[prost(string, optional, tag = "11")]
    pub env_vars: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateMcpServerRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(int64, tag = "3")]
    pub install_id: i64,
    #[prost(bool, optional, tag = "4")]
    pub is_enabled: Option<bool>,
    #[prost(string, optional, tag = "5")]
    pub env_vars: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UninstallMcpServerRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(int64, tag = "3")]
    pub install_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UninstallMcpServerResponse {}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn installed_mcp_server_round_trip_preserves_every_field() {
        let original = InstalledMcpServer {
            id: 1,
            organization_id: 42,
            repository_id: 7,
            market_item_id: Some(21),
            scope: "org".into(),
            installed_by: Some(99),
            name: "GitHub MCP".into(),
            slug: "github".into(),
            transport_type: "stdio".into(),
            command: "npx".into(),
            args: r#"["@modelcontextprotocol/server-github"]"#.into(),
            http_url: "".into(),
            http_headers: "{}".into(),
            env_vars: r#"{"GITHUB_TOKEN":"ghp_xxx"}"#.into(),
            is_enabled: true,
            created_at: "2026-05-01T00:00:00Z".into(),
            updated_at: "2026-05-12T00:00:00Z".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = InstalledMcpServer::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn install_custom_round_trip_with_all_optionals() {
        let original = InstallCustomMcpServerRequest {
            org_slug: "acme".into(),
            repository_id: 7,
            name: "custom".into(),
            slug: "custom-mcp".into(),
            transport_type: "http".into(),
            command: Some("".into()),
            args: Some("[]".into()),
            http_url: Some("https://example.com/mcp".into()),
            http_headers: Some(r#"{"X-Api-Key":"k"}"#.into()),
            scope: "user".into(),
            env_vars: Some("{}".into()),
        };
        let bytes = original.encode_to_vec();
        let decoded = InstallCustomMcpServerRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn update_mcp_server_tri_state_distinguishable() {
        let explicit_false = UpdateMcpServerRequest {
            org_slug: "acme".into(),
            repository_id: 7,
            install_id: 99,
            is_enabled: Some(false),
            env_vars: None,
        };
        let absent = UpdateMcpServerRequest {
            org_slug: "acme".into(),
            repository_id: 7,
            install_id: 99,
            is_enabled: None,
            env_vars: None,
        };
        assert_ne!(explicit_false.encode_to_vec(), absent.encode_to_vec());
    }

    #[test]
    fn uninstall_response_empty_round_trip() {
        let original = UninstallMcpServerResponse {};
        let bytes = original.encode_to_vec();
        assert!(bytes.is_empty());
        let decoded = UninstallMcpServerResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }
}
