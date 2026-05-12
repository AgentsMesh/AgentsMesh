// Hand-maintained `prost::Message` mirrors of
// `proto/extension/v1/repo_skill.proto`. Tag numbers match the .proto
// byte-for-byte; `tools/validate_prost_tags` runs at build time to catch
// drift (watch list §8). NO `Serialize`/`Deserialize` derives — binary
// wire only (conventions §2.5, §3).

#[derive(Clone, PartialEq, prost::Message)]
pub struct InstalledSkill {
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
    pub slug: String,
    #[prost(string, tag = "8")]
    pub install_source: String,
    #[prost(string, tag = "9")]
    pub source_url: String,
    #[prost(string, tag = "10")]
    pub content_sha: String,
    #[prost(string, tag = "11")]
    pub storage_key: String,
    #[prost(int64, tag = "12")]
    pub package_size: i64,
    #[prost(int32, optional, tag = "13")]
    pub pinned_version: Option<i32>,
    #[prost(bool, tag = "14")]
    pub is_enabled: bool,
    #[prost(string, tag = "15")]
    pub created_at: String,
    #[prost(string, tag = "16")]
    pub updated_at: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListRepoSkillsRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(string, optional, tag = "3")]
    pub scope: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ListRepoSkillsResponse {
    #[prost(message, repeated, tag = "1")]
    pub items: Vec<InstalledSkill>,
    #[prost(int64, tag = "2")]
    pub total: i64,
    #[prost(int32, tag = "3")]
    pub limit: i32,
    #[prost(int32, tag = "4")]
    pub offset: i32,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct InstallSkillFromMarketRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(int64, tag = "3")]
    pub market_item_id: i64,
    #[prost(string, tag = "4")]
    pub scope: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct InstallSkillFromGitHubRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(string, tag = "3")]
    pub url: String,
    #[prost(string, optional, tag = "4")]
    pub branch: Option<String>,
    #[prost(string, optional, tag = "5")]
    pub path: Option<String>,
    #[prost(string, tag = "6")]
    pub scope: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UpdateSkillRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(int64, tag = "3")]
    pub install_id: i64,
    #[prost(bool, optional, tag = "4")]
    pub is_enabled: Option<bool>,
    #[prost(int32, optional, tag = "5")]
    pub pinned_version: Option<i32>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UninstallSkillRequest {
    #[prost(string, tag = "1")]
    pub org_slug: String,
    #[prost(int64, tag = "2")]
    pub repository_id: i64,
    #[prost(int64, tag = "3")]
    pub install_id: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct UninstallSkillResponse {}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    #[test]
    fn installed_skill_round_trip_preserves_every_field() {
        let original = InstalledSkill {
            id: 1,
            organization_id: 42,
            repository_id: 7,
            market_item_id: Some(11),
            scope: "user".into(),
            installed_by: Some(99),
            slug: "format-go".into(),
            install_source: "market".into(),
            source_url: "".into(),
            content_sha: "abc123".into(),
            storage_key: "skills/format-go.zip".into(),
            package_size: 4096,
            pinned_version: Some(2),
            is_enabled: true,
            created_at: "2026-05-01T00:00:00Z".into(),
            updated_at: "2026-05-12T00:00:00Z".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = InstalledSkill::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn install_from_github_round_trip_with_optionals() {
        let original = InstallSkillFromGitHubRequest {
            org_slug: "acme".into(),
            repository_id: 7,
            url: "https://github.com/owner/repo".into(),
            branch: Some("main".into()),
            path: Some("skills".into()),
            scope: "org".into(),
        };
        let bytes = original.encode_to_vec();
        let decoded = InstallSkillFromGitHubRequest::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn update_skill_tri_state_distinguishable() {
        let explicit_false = UpdateSkillRequest {
            org_slug: "acme".into(),
            repository_id: 7,
            install_id: 99,
            is_enabled: Some(false),
            pinned_version: None,
        };
        let absent = UpdateSkillRequest {
            org_slug: "acme".into(),
            repository_id: 7,
            install_id: 99,
            is_enabled: None,
            pinned_version: None,
        };
        assert_ne!(explicit_false.encode_to_vec(), absent.encode_to_vec());
    }

    #[test]
    fn uninstall_response_empty_round_trip() {
        let original = UninstallSkillResponse {};
        let bytes = original.encode_to_vec();
        assert!(bytes.is_empty());
        let decoded = UninstallSkillResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
    }
}
