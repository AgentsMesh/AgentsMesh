use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GitCredential {
    pub id: i64,
    pub name: String,
    pub credential_type: Option<String>,
    pub repository_provider_id: Option<i64>,
    pub host_pattern: Option<String>,
    pub is_default: Option<bool>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateGitCredentialRequest {
    pub name: String,
    pub credential_type: String,
    pub repository_provider_id: Option<i64>,
    pub pat: Option<String>,
    pub private_key: Option<String>,
    pub host_pattern: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateGitCredentialRequest {
    pub name: Option<String>,
    pub pat: Option<String>,
    pub private_key: Option<String>,
    pub host_pattern: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetDefaultGitCredentialRequest {
    pub credential_id: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GitCredentialListResponse {
    #[serde(default)]
    pub credentials: Vec<GitCredential>,
    #[serde(default)]
    pub runner_local: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentCredentialProfile {
    pub id: i64,
    pub agent_slug: Option<String>,
    pub name: String,
    pub description: Option<String>,
    pub is_runner_host: Option<bool>,
    pub is_default: Option<bool>,
    pub credentials: Option<serde_json::Value>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAgentCredentialProfileRequest {
    pub name: String,
    pub description: Option<String>,
    pub is_runner_host: Option<bool>,
    pub credentials: Option<serde_json::Value>,
    pub is_default: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateAgentCredentialProfileRequest {
    pub name: Option<String>,
    pub description: Option<String>,
    pub is_runner_host: Option<bool>,
    pub credentials: Option<serde_json::Value>,
    pub is_default: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CredentialProfilesByAgent {
    pub agent_slug: String,
    #[serde(default)]
    pub agent_name: Option<String>,
    #[serde(default)]
    pub profiles: Vec<AgentCredentialProfile>,
}

// Backend wire format: `GET /api/v1/users/agent-credentials` returns
// `{ "items": [{ agent_slug, agent_name, profiles: [...] }] }` — a grouped
// view across all agents. Distinct from the per-agent endpoint below.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentCredentialProfileListResponse {
    #[serde(default)]
    pub items: Vec<CredentialProfilesByAgent>,
}

// Backend wire format: `GET /api/v1/users/agent-credentials/agents/:slug`
// returns `{ "profiles": [...], "runner_host": {...} }` — a flat list for a
// single agent plus the virtual RunnerHost descriptor.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentCredentialProfilesForAgentResponse {
    #[serde(default)]
    pub profiles: Vec<AgentCredentialProfile>,
    #[serde(default)]
    pub runner_host: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepositoryProvider {
    pub id: i64,
    pub provider_type: String,
    pub name: String,
    pub base_url: Option<String>,
    #[serde(default)]
    pub has_client_id: Option<bool>,
    #[serde(default)]
    pub has_bot_token: Option<bool>,
    #[serde(default)]
    pub has_identity: Option<bool>,
    pub is_default: Option<bool>,
    #[serde(default)]
    pub is_active: Option<bool>,
    pub created_at: Option<String>,
    pub updated_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateRepositoryProviderRequest {
    pub provider_type: String,
    pub name: String,
    pub base_url: Option<String>,
    pub client_id: Option<String>,
    pub client_secret: Option<String>,
    pub bot_token: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateRepositoryProviderRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub base_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub client_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub client_secret: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bot_token: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_active: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RepositoryProviderListResponse {
    pub providers: Vec<RepositoryProvider>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderRepository {
    pub id: Option<String>,
    pub name: String,
    pub slug: Option<String>,
    pub description: Option<String>,
    pub default_branch: Option<String>,
    pub visibility: Option<String>,
    pub http_clone_url: Option<String>,
    pub ssh_clone_url: Option<String>,
    pub web_url: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderRepositoryListResponse {
    pub repositories: Vec<ProviderRepository>,
    pub total: Option<i64>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn agent_credential_list_decodes_backend_grouped_shape() {
        // Wire shape produced by ListProfiles handler — grouped by agent.
        // Regression: prior type modelled this as a flat profiles list which
        // failed reverse-serialization (missing `id` on group items) and
        // collapsed the renderer's credential list to empty.
        let backend_json = r#"{
            "items": [
                {
                    "agent_slug": "claude",
                    "agent_name": "Claude",
                    "profiles": [
                        {
                            "id": 7,
                            "agent_slug": "claude",
                            "name": "personal",
                            "description": "my key",
                            "is_runner_host": false,
                            "is_default": true,
                            "credentials": null,
                            "created_at": "2026-05-01T00:00:00Z",
                            "updated_at": "2026-05-01T00:00:00Z"
                        }
                    ]
                },
                { "agent_slug": "codex", "agent_name": "Codex", "profiles": [] }
            ]
        }"#;

        let resp: AgentCredentialProfileListResponse =
            serde_json::from_str(backend_json).unwrap();
        assert_eq!(resp.items.len(), 2);
        assert_eq!(resp.items[0].agent_slug, "claude");
        assert_eq!(resp.items[0].profiles.len(), 1);
        assert_eq!(resp.items[0].profiles[0].id, 7);
        assert_eq!(resp.items[0].profiles[0].is_default, Some(true));
        assert_eq!(resp.items[1].agent_slug, "codex");
        assert!(resp.items[1].profiles.is_empty());

        // Re-serialize: front-end relays the JSON unchanged, so the renderer
        // must still see `items`, not `profiles`.
        let relayed = serde_json::to_value(&resp).unwrap();
        assert!(relayed.get("items").is_some());
        assert!(relayed.get("profiles").is_none());
    }

    #[test]
    fn agent_credential_list_tolerates_empty_items() {
        let resp: AgentCredentialProfileListResponse =
            serde_json::from_str(r#"{"items":[]}"#).unwrap();
        assert!(resp.items.is_empty());
    }

    #[test]
    fn agent_credential_for_agent_decodes_flat_shape() {
        // Wire shape produced by ListProfilesForAgent handler — flat profiles
        // list plus the virtual RunnerHost descriptor.
        let backend_json = r#"{
            "profiles": [
                {
                    "id": 11,
                    "agent_slug": "claude",
                    "name": "work",
                    "description": null,
                    "is_runner_host": false,
                    "is_default": false,
                    "credentials": null,
                    "created_at": "2026-05-01T00:00:00Z",
                    "updated_at": "2026-05-01T00:00:00Z"
                }
            ],
            "runner_host": { "available": true, "description": "Use Runner machine" }
        }"#;

        let resp: AgentCredentialProfilesForAgentResponse =
            serde_json::from_str(backend_json).unwrap();
        assert_eq!(resp.profiles.len(), 1);
        assert_eq!(resp.profiles[0].id, 11);
        assert!(resp.runner_host.is_some());
    }

    #[test]
    fn repository_provider_preserves_backend_fields() {
        let backend_json = r#"{
            "id": 42,
            "provider_type": "github",
            "name": "GitHub",
            "base_url": "https://github.com",
            "has_client_id": false,
            "has_bot_token": false,
            "has_identity": true,
            "is_default": true,
            "is_active": true,
            "created_at": "2026-05-06T00:00:00Z",
            "updated_at": "2026-05-06T00:00:00Z"
        }"#;

        let provider: RepositoryProvider = serde_json::from_str(backend_json).unwrap();
        assert_eq!(provider.is_active, Some(true));
        assert_eq!(provider.has_identity, Some(true));
        assert_eq!(provider.has_bot_token, Some(false));
        assert_eq!(provider.has_client_id, Some(false));

        let reserialized = serde_json::to_value(&provider).unwrap();
        assert_eq!(reserialized["is_active"], serde_json::json!(true));
        assert_eq!(reserialized["has_identity"], serde_json::json!(true));
        assert_eq!(reserialized["has_bot_token"], serde_json::json!(false));
        assert_eq!(reserialized["has_client_id"], serde_json::json!(false));
    }

    #[test]
    fn repository_provider_tolerates_missing_optional_fields() {
        let minimal_json = r#"{
            "id": 1,
            "provider_type": "gitlab",
            "name": "GitLab"
        }"#;

        let provider: RepositoryProvider = serde_json::from_str(minimal_json).unwrap();
        assert_eq!(provider.is_active, None);
        assert_eq!(provider.has_identity, None);
    }

    #[test]
    fn update_repository_provider_request_round_trip_keeps_is_active() {
        let frontend_json = r#"{"is_active":true}"#;
        let req: UpdateRepositoryProviderRequest = serde_json::from_str(frontend_json).unwrap();
        assert_eq!(req.is_active, Some(true));

        let body = serde_json::to_value(&req).unwrap();
        assert_eq!(body["is_active"], serde_json::json!(true));
        assert!(
            body.get("name").is_none(),
            "skip_serializing_if should drop None fields so backend treats them as untouched"
        );
        assert!(body.get("client_id").is_none());
    }

    #[test]
    fn update_repository_provider_request_round_trip_keeps_is_active_false() {
        let frontend_json = r#"{"is_active":false}"#;
        let req: UpdateRepositoryProviderRequest = serde_json::from_str(frontend_json).unwrap();
        assert_eq!(req.is_active, Some(false));

        let body = serde_json::to_value(&req).unwrap();
        assert_eq!(body["is_active"], serde_json::json!(false));
    }

    #[test]
    fn update_repository_provider_request_skips_omitted_fields() {
        let req = UpdateRepositoryProviderRequest {
            name: Some("Renamed".to_string()),
            base_url: None,
            client_id: None,
            client_secret: None,
            bot_token: None,
            is_active: None,
        };
        let body = serde_json::to_value(&req).unwrap();
        assert_eq!(body["name"], serde_json::json!("Renamed"));
        assert!(body.get("is_active").is_none());
        assert!(body.get("base_url").is_none());
        assert!(body.get("client_secret").is_none());
    }

    #[test]
    fn repository_provider_list_response_round_trip() {
        let backend_json = r#"{
            "providers": [
                {
                    "id": 1,
                    "provider_type": "github",
                    "name": "GitHub",
                    "is_active": true,
                    "has_identity": false
                },
                {
                    "id": 2,
                    "provider_type": "gitlab",
                    "name": "GitLab Self-Hosted",
                    "is_active": false,
                    "has_bot_token": true
                }
            ]
        }"#;

        let resp: RepositoryProviderListResponse = serde_json::from_str(backend_json).unwrap();
        assert_eq!(resp.providers.len(), 2);
        assert_eq!(resp.providers[0].is_active, Some(true));
        assert_eq!(resp.providers[1].is_active, Some(false));
        assert_eq!(resp.providers[1].has_bot_token, Some(true));

        let reserialized = serde_json::to_value(&resp).unwrap();
        assert_eq!(reserialized["providers"][0]["is_active"], serde_json::json!(true));
        assert_eq!(reserialized["providers"][1]["is_active"], serde_json::json!(false));
    }

    const PROVIDER_REPO_PAYLOAD: &str = r#"{
        "id": "987654",
        "name": "infra-tools",
        "slug": "infra-tools",
        "description": "Internal infrastructure helpers",
        "default_branch": "main",
        "visibility": "private",
        "http_clone_url": "https://gitlab.example.com/group/infra-tools.git",
        "ssh_clone_url": "git@gitlab.example.com:group/infra-tools.git",
        "web_url": "https://gitlab.example.com/group/infra-tools"
    }"#;

    #[test]
    fn provider_repository_decodes_backend_payload() {
        let r: ProviderRepository = serde_json::from_str(PROVIDER_REPO_PAYLOAD).unwrap();
        assert_eq!(r.name, "infra-tools");
        assert_eq!(r.description.as_deref(), Some("Internal infrastructure helpers"));
        assert_eq!(
            r.http_clone_url.as_deref(),
            Some("https://gitlab.example.com/group/infra-tools.git"),
        );
        assert_eq!(
            r.ssh_clone_url.as_deref(),
            Some("git@gitlab.example.com:group/infra-tools.git"),
        );
    }

    #[test]
    fn provider_repository_wasm_relay_preserves_import_fields() {
        let typed: ProviderRepository = serde_json::from_str(PROVIDER_REPO_PAYLOAD).unwrap();
        let relayed = serde_json::to_string(&typed).unwrap();
        let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
        for key in ["name", "description", "http_clone_url", "ssh_clone_url",
                    "default_branch", "visibility", "web_url", "slug"] {
            assert!(!parsed[key].is_null(), "field `{key}` dropped by relay");
        }
    }
}
