#[cfg(test)]
mod api_channel_extension_tests {
    use std::sync::{Arc, Mutex};

    use serde_json::json;
    use wiremock::matchers::{body_json, method, path, query_param};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    use crate::{ApiClient, AuthTokenStore};

    struct MockTokenStore {
        org_slug: Mutex<Option<String>>,
    }
    impl MockTokenStore {
        fn with_org(slug: &str) -> Arc<Self> {
            Arc::new(Self { org_slug: Mutex::new(Some(slug.into())) })
        }
        fn no_org() -> Arc<Self> {
            Arc::new(Self { org_slug: Mutex::new(None) })
        }
    }
    impl AuthTokenStore for MockTokenStore {
        fn get_token(&self) -> Option<String> { Some("tok".into()) }
        fn get_refresh_token(&self) -> Option<String> { None }
        fn set_tokens(&self, _t: String, _r: String) {}
        fn clear_tokens(&self) {}
        fn get_current_org_slug(&self) -> Option<String> {
            self.org_slug.lock().unwrap().clone()
        }
    }

    fn ok(body: serde_json::Value) -> ResponseTemplate {
        ResponseTemplate::new(200).set_body_json(body)
    }

    // ── binding ─────────────────────────────────────────────────────────

    #[tokio::test]
    async fn request_binding() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/bindings"))
            .and(body_json(json!({
                "target_pod":"pod-x","scopes":null,"policy":null
            })))
            .respond_with(ok(json!({"id":1,"target_pod":"pod-x"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::CreateBindingRequest {
            target_pod: "pod-x".into(),
            scopes: None,
            policy: None,
        };
        let r = c.request_binding(&data, None).await.unwrap();
        assert_eq!(r.id, 1);
    }

    #[tokio::test]
    async fn request_binding_with_pod_key() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/bindings"))
            .respond_with(ok(json!({"id":2,"target_pod":"pod-y"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::CreateBindingRequest {
            target_pod: "pod-y".into(),
            scopes: Some(vec!["read".into()]),
            policy: None,
        };
        let r = c.request_binding(&data, Some("key-1")).await.unwrap();
        assert_eq!(r.id, 2);
        let reqs = s.received_requests().await.unwrap();
        let hdr = reqs[0].headers.get("X-Pod-Key").unwrap();
        assert_eq!(hdr.as_bytes(), b"key-1");
    }

    #[tokio::test]
    async fn accept_binding() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/bindings/accept"))
            .and(body_json(json!({"binding_id":5})))
            .respond_with(ok(json!({"id":5,"status":"accepted"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::AcceptBindingRequest { binding_id: 5 };
        let r = c.accept_binding(&data).await.unwrap();
        assert_eq!(r.id, 5);
    }

    #[tokio::test]
    async fn reject_binding() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/bindings/reject"))
            .and(body_json(json!({"binding_id":6,"reason":null})))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::RejectBindingRequest {
            binding_id: 6,
            reason: None,
        };
        let _ = c.reject_binding(&data).await.unwrap();
    }

    #[tokio::test]
    async fn request_binding_scopes() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/bindings/10/scopes"))
            .and(body_json(json!({"scopes":["read","write"]})))
            .respond_with(ok(json!({"id":10,"scopes":["read","write"]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::RequestScopesBody {
            scopes: vec!["read".into(), "write".into()],
        };
        let r = c.request_binding_scopes(10, &data).await.unwrap();
        assert_eq!(r.id, 10);
    }

    #[tokio::test]
    async fn approve_binding_scopes() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/bindings/10/scopes/approve"))
            .and(body_json(json!({"scopes":["read"]})))
            .respond_with(ok(json!({"id":10,"scopes":["read"]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::ApproveScopesBody {
            scopes: vec!["read".into()],
        };
        let r = c.approve_binding_scopes(10, &data).await.unwrap();
        assert_eq!(r.id, 10);
    }

    #[tokio::test]
    async fn unbind() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/bindings/unbind"))
            .and(body_json(json!({"target_pod":"pod-z"})))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::UnbindRequest {
            target_pod: "pod-z".into(),
        };
        let _ = c.unbind(&data).await.unwrap();
    }

    #[tokio::test]
    async fn list_bindings_no_filter() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/bindings"))
            .respond_with(ok(json!({"bindings":[{"id":1}]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_bindings(None).await.unwrap();
        assert_eq!(r.bindings.len(), 1);
    }

    #[tokio::test]
    async fn get_bound_pods() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/bindings/pods"))
            .respond_with(ok(json!({"pods":["p1","p2"]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.get_bound_pods().await.unwrap();
        assert_eq!(r.pods.len(), 2);
    }

    #[tokio::test]
    async fn check_binding() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/bindings/check/pod-abc"))
            .respond_with(ok(json!({"id":3,"target_pod":"pod-abc","status":"active"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.check_binding("pod-abc").await.unwrap();
        assert_eq!(r.id, 3);
    }

    // ── channel ─────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_channels_with_archived() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/channels"))
            .and(query_param("include_archived", "true"))
            .respond_with(ok(json!({"channels":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_channels(Some(true)).await.unwrap();
        assert!(r.channels.is_empty());
    }

    #[tokio::test]
    async fn get_channel() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/channels/7"))
            .respond_with(ok(json!({
                "id":7,"name":"general","is_archived":false
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.get_channel(7).await.unwrap();
        assert_eq!(r.name, "general");
    }

    #[tokio::test]
    async fn create_channel() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/channels"))
            .and(body_json(json!({
                "name":"dev","description":null,"document":null,
                "repository_id":null,"ticket_slug":null
            })))
            .respond_with(ok(json!({"id":1,"name":"dev","is_archived":false})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::CreateChannelRequest {
            name: "dev".into(),
            description: None,
            document: None,
            repository_id: None,
            ticket_slug: None,
        };
        let r = c.create_channel(&data).await.unwrap();
        assert_eq!(r.name, "dev");
    }

    #[tokio::test]
    async fn update_channel() {
        let s = MockServer::start().await;
        Mock::given(method("PUT")).and(path("/api/v1/orgs/acme/channels/3"))
            .and(body_json(json!({"name":"renamed","description":null})))
            .respond_with(ok(json!({"id":3,"name":"renamed","is_archived":false})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::UpdateChannelRequest {
            name: Some("renamed".into()),
            description: None,
        };
        let r = c.update_channel(3, &data).await.unwrap();
        assert_eq!(r.name, "renamed");
    }

    #[tokio::test]
    async fn archive_channel() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/channels/5/archive"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.archive_channel(5).await.unwrap();
    }

    #[tokio::test]
    async fn unarchive_channel() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/channels/5/unarchive"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.unarchive_channel(5).await.unwrap();
    }

    #[tokio::test]
    async fn get_channel_messages() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/channels/2/messages"))
            .and(query_param("limit", "20"))
            .respond_with(ok(json!({"messages":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.get_channel_messages(2, Some(20), None).await.unwrap();
        assert!(r.messages.is_empty());
    }

    #[tokio::test]
    async fn get_channel_messages_with_before_id() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/channels/2/messages"))
            .and(query_param("limit", "10"))
            .and(query_param("before_id", "99"))
            .respond_with(ok(json!({"messages":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_channel_messages(2, Some(10), Some(99)).await.unwrap();
    }

    #[tokio::test]
    async fn send_channel_message() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/channels/2/messages"))
            .and(body_json(json!({
                "content":"hello"
            })))
            .respond_with(ok(json!({
                "id":1,"channel_id":2,"body":"hello","content":"hello"
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::SendChannelMessageRequest {
            content: Some(serde_json::json!("hello")),
            pod_key: None,
            reply_to: None,
            ..Default::default()
        };
        let r = c.send_channel_message(2, &data).await.unwrap();
        assert_eq!(r.body, "hello");
    }

    #[tokio::test]
    async fn get_channel_pods() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/channels/4/pods"))
            .respond_with(ok(json!({"pods":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.get_channel_pods(4).await.unwrap();
        assert!(r.pods.is_empty());
    }

    #[tokio::test]
    async fn join_channel_pod() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/channels/4/pods"))
            .and(body_json(json!({"pod_key":"pk-1"})))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::JoinChannelPodRequest {
            pod_key: "pk-1".into(),
        };
        let _ = c.join_channel_pod(4, &data).await.unwrap();
    }

    #[tokio::test]
    async fn leave_channel_pod() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/orgs/acme/channels/4/pods/pk-1"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.leave_channel_pod(4, "pk-1").await.unwrap();
    }

    #[tokio::test]
    async fn mark_channel_read() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/channels/4/read"))
            .and(body_json(json!({"message_id":50})))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.mark_channel_read(4, 50).await.unwrap();
    }

    #[tokio::test]
    async fn get_channel_unread_counts() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/channels/unread"))
            .respond_with(ok(json!({"unread":{}})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_channel_unread_counts().await.unwrap();
    }

    #[tokio::test]
    async fn edit_channel_message() {
        let s = MockServer::start().await;
        Mock::given(method("PUT"))
            .and(path("/api/v1/orgs/acme/channels/2/messages/10"))
            .and(body_json(json!({"content":"edited"})))
            .respond_with(ok(json!({
                "id":10,"channel_id":2,"body":"edited","content":"edited"
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::EditChannelMessageRequest {
            content: Some(serde_json::json!("edited")),
            ..Default::default()
        };
        let r = c.edit_channel_message(2, 10, &data).await.unwrap();
        assert_eq!(r.body, "edited");
    }

    #[tokio::test]
    async fn delete_channel_message() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/orgs/acme/channels/2/messages/10"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.delete_channel_message(2, 10).await.unwrap();
    }

    #[tokio::test]
    async fn mute_channel() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/channels/3/mute"))
            .and(body_json(json!({"muted":true})))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::MuteChannelRequest { muted: true };
        let _ = c.mute_channel(3, &data).await.unwrap();
    }

    // ── extension ───────────────────────────────────────────────────────

    #[tokio::test]
    async fn create_skill_registry() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/skill-registries"))
            .and(body_json(json!({
                "repository_url":"https://github.com/r",
                "branch":null,"source_type":null,
                "compatible_agents":null,"auth_type":null,
                "auth_credential":null
            })))
            .respond_with(ok(json!({
                "id":1,"repository_url":"https://github.com/r"
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::CreateSkillRegistryRequest {
            repository_url: "https://github.com/r".into(),
            branch: None,
            source_type: None,
            compatible_agents: None,
            auth_type: None,
            auth_credential: None,
        };
        let r = c.create_skill_registry(&data).await.unwrap();
        assert_eq!(r.id, 1);
    }

    #[tokio::test]
    async fn sync_skill_registry() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/skill-registries/3/sync"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.sync_skill_registry(3).await.unwrap();
    }

    #[tokio::test]
    async fn toggle_skill_registry() {
        let s = MockServer::start().await;
        Mock::given(method("PUT"))
            .and(path("/api/v1/orgs/acme/skill-registries/3/toggle"))
            .and(body_json(json!({"disabled":true})))
            .respond_with(ok(json!({"id":3,"is_disabled":true})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::ToggleRegistryRequest { disabled: true };
        let r = c.toggle_skill_registry(3, &data).await.unwrap();
        assert_eq!(r.id, 3);
    }

    #[tokio::test]
    async fn delete_skill_registry() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/orgs/acme/skill-registries/3"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.delete_skill_registry(3).await.unwrap();
    }

    #[tokio::test]
    async fn list_skill_registry_overrides() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/skill-registry-overrides"))
            .respond_with(ok(json!({"overrides":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_skill_registry_overrides().await.unwrap();
        assert!(r.overrides.is_empty());
    }

    #[tokio::test]
    async fn list_market_mcp_servers() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/market/mcp-servers"))
            .and(query_param("q", "docker"))
            .and(query_param("limit", "5"))
            .respond_with(ok(json!({"mcp_servers":[],"total":0})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_market_mcp_servers(Some("docker"), Some(5), None).await.unwrap();
        assert!(r.mcp_servers.is_empty());
    }

    #[tokio::test]
    async fn list_repo_skills() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/repositories/5/skills"))
            .and(query_param("scope", "org"))
            .respond_with(ok(json!({"skills":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_repo_skills(5, Some("org")).await.unwrap();
        assert!(r.skills.is_empty());
    }

    #[tokio::test]
    async fn install_skill_from_market() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/repositories/5/skills/install-from-market"))
            .and(body_json(json!({"market_item_id":42,"scope":null})))
            .respond_with(ok(json!({"id":1,"skill_slug":"git-commit"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::InstallMarketSkillRequest {
            market_item_id: 42,
            scope: None,
        };
        let r = c.install_skill_from_market(5, &data).await.unwrap();
        assert_eq!(r.id, 1);
    }

    #[tokio::test]
    async fn install_skill_from_github() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/repositories/5/skills/install-from-github"))
            .and(body_json(json!({
                "url":"https://github.com/org/skill",
                "branch":null,"path":null,"scope":null
            })))
            .respond_with(ok(json!({"id":2,"source":"github"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::InstallGithubSkillRequest {
            url: "https://github.com/org/skill".into(),
            branch: None,
            path: None,
            scope: None,
        };
        let r = c.install_skill_from_github(5, &data).await.unwrap();
        assert_eq!(r.id, 2);
    }

    #[tokio::test]
    async fn update_skill_install() {
        let s = MockServer::start().await;
        Mock::given(method("PUT"))
            .and(path("/api/v1/orgs/acme/repositories/5/skills/10"))
            .and(body_json(json!({
                "is_enabled":false,"pinned_version":null
            })))
            .respond_with(ok(json!({"id":10,"is_enabled":false})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::UpdateSkillInstallRequest {
            is_enabled: Some(false),
            pinned_version: None,
        };
        let r = c.update_skill_install(5, 10, &data).await.unwrap();
        assert_eq!(r.id, 10);
    }

    #[tokio::test]
    async fn uninstall_skill() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/orgs/acme/repositories/5/skills/10"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.uninstall_skill(5, 10).await.unwrap();
    }

    #[tokio::test]
    async fn list_repo_mcp_servers() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/repositories/5/mcp-servers"))
            .respond_with(ok(json!({"mcp_servers":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_repo_mcp_servers(5, None).await.unwrap();
        assert!(r.mcp_servers.is_empty());
    }

    #[tokio::test]
    async fn install_mcp_from_market() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path(
                "/api/v1/orgs/acme/repositories/5/mcp-servers/install-from-market",
            ))
            .and(body_json(json!({
                "market_item_id":99,"env_vars":null,"scope":null
            })))
            .respond_with(ok(json!({"id":1,"slug":"docker-mcp"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::InstallMarketMcpRequest {
            market_item_id: 99,
            env_vars: None,
            scope: None,
        };
        let r = c.install_mcp_from_market(5, &data).await.unwrap();
        assert_eq!(r.id, 1);
    }

    #[tokio::test]
    async fn install_custom_mcp_server() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path(
                "/api/v1/orgs/acme/repositories/5/mcp-servers/install-custom",
            ))
            .and(body_json(json!({
                "name":"my-mcp","slug":null,"transport_type":null,
                "command":null,"args":null,"http_url":null,
                "http_headers":null,"env_vars":null,"scope":null
            })))
            .respond_with(ok(json!({"id":2,"name":"my-mcp"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::InstallCustomMcpRequest {
            name: "my-mcp".into(),
            slug: None,
            transport_type: None,
            command: None,
            args: None,
            http_url: None,
            http_headers: None,
            env_vars: None,
            scope: None,
        };
        let r = c.install_custom_mcp_server(5, &data).await.unwrap();
        assert_eq!(r.id, 2);
    }

    #[tokio::test]
    async fn update_mcp_install() {
        let s = MockServer::start().await;
        Mock::given(method("PUT"))
            .and(path("/api/v1/orgs/acme/repositories/5/mcp-servers/8"))
            .and(body_json(json!({"is_enabled":true,"env_vars":null})))
            .respond_with(ok(json!({"id":8,"is_enabled":true})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::UpdateMcpInstallRequest {
            is_enabled: Some(true),
            env_vars: None,
        };
        let r = c.update_mcp_install(5, 8, &data).await.unwrap();
        assert_eq!(r.id, 8);
    }

    #[tokio::test]
    async fn uninstall_mcp_server() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/orgs/acme/repositories/5/mcp-servers/8"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.uninstall_mcp_server(5, 8).await.unwrap();
    }

    // ── invitation ──────────────────────────────────────────────────────

    #[tokio::test]
    async fn create_invitation() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/invitations"))
            .and(body_json(json!({"email":"a@b.com","role":"member"})))
            .respond_with(ok(json!({
                "id":1,"email":"a@b.com","role":"member"
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::CreateInvitationRequest {
            email: "a@b.com".into(),
            role: "member".into(),
        };
        let r = c.create_invitation(&data).await.unwrap();
        assert_eq!(r.email, "a@b.com");
    }

    #[tokio::test]
    async fn revoke_invitation() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/orgs/acme/invitations/9"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.revoke_invitation(9).await.unwrap();
    }

    #[tokio::test]
    async fn resend_invitation() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/invitations/9/resend"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.resend_invitation(9).await.unwrap();
    }

    #[tokio::test]
    async fn accept_invitation() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/invitations/tok-xyz/accept"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.accept_invitation("tok-xyz").await.unwrap();
        let reqs = s.received_requests().await.unwrap();
        assert!(reqs[0].headers.get("Authorization").is_none());
    }

    #[tokio::test]
    async fn list_pending_invitations() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/invitations/pending"))
            .respond_with(ok(json!({"invitations":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let r = c.list_pending_invitations().await.unwrap();
        assert!(r.invitations.is_empty());
    }

    // ── loop_api ────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_loops_no_filter() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/loops"))
            .respond_with(ok(json!({"loops":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_loops(None, None, None).await.unwrap();
        assert!(r.loops.is_empty());
    }

    #[tokio::test]
    async fn list_loops_with_pagination() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/loops"))
            .and(query_param("limit", "10"))
            .and(query_param("offset", "20"))
            .respond_with(ok(json!({"loops":[],"total":50})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_loops(None, Some(10), Some(20)).await.unwrap();
        assert_eq!(r.total, Some(50));
    }

    #[tokio::test]
    async fn get_loop() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/loops/daily-check"))
            .respond_with(ok(json!({
                "slug":"daily-check","name":"Daily Check","is_enabled":true
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.get_loop("daily-check").await.unwrap();
        assert_eq!(r.slug, "daily-check");
        assert!(r.is_enabled);
    }

    #[tokio::test]
    async fn create_loop() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/loops"))
            .and(body_json(json!({
                "name":"Nightly","slug":null,"description":null,
                "agent_slug":null,"custom_agent_slug":null,
                "permission_mode":null,"prompt_template":null,
                "prompt_variables":null,"repository_id":null,
                "runner_id":null,"branch_name":null,"ticket_id":null,
                "credential_profile_id":null,"config_overrides":null,
                "execution_mode":null,"cron_expression":null,
                "autopilot_config":null,"callback_url":null,
                "sandbox_strategy":null,"session_persistence":null,
                "concurrency_policy":null,"max_concurrent_runs":null,
                "max_retained_runs":null,"timeout_minutes":null
            })))
            .respond_with(ok(json!({
                "slug":"nightly","name":"Nightly","is_enabled":false
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::CreateLoopRequest {
            name: "Nightly".into(),
            slug: None,
            description: None,
            agent_slug: None,
            custom_agent_slug: None,
            permission_mode: None,
            prompt_template: None,
            prompt_variables: None,
            repository_id: None,
            runner_id: None,
            branch_name: None,
            ticket_id: None,
            credential_profile_id: None,
            config_overrides: None,
            execution_mode: None,
            cron_expression: None,
            autopilot_config: None,
            callback_url: None,
            sandbox_strategy: None,
            session_persistence: None,
            concurrency_policy: None,
            max_concurrent_runs: None,
            max_retained_runs: None,
            timeout_minutes: None,
        };
        let r = c.create_loop(&data).await.unwrap();
        assert_eq!(r.name, "Nightly");
    }

    #[tokio::test]
    async fn update_loop() {
        let s = MockServer::start().await;
        Mock::given(method("PUT"))
            .and(path("/api/v1/orgs/acme/loops/nightly"))
            .and(body_json(json!({
                "name":"Nightly v2","description":null,
                "agent_slug":null,"prompt_template":null,
                "prompt_variables":null,"repository_id":null,
                "runner_id":null,"branch_name":null,
                "cron_expression":null,"autopilot_config":null,
                "sandbox_strategy":null,"session_persistence":null,
                "concurrency_policy":null,"max_concurrent_runs":null,
                "max_retained_runs":null,"timeout_minutes":null
            })))
            .respond_with(ok(json!({
                "slug":"nightly","name":"Nightly v2","is_enabled":true
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::UpdateLoopRequest {
            name: Some("Nightly v2".into()),
            description: None,
            agent_slug: None,
            prompt_template: None,
            prompt_variables: None,
            repository_id: None,
            runner_id: None,
            branch_name: None,
            cron_expression: None,
            autopilot_config: None,
            sandbox_strategy: None,
            session_persistence: None,
            concurrency_policy: None,
            max_concurrent_runs: None,
            max_retained_runs: None,
            timeout_minutes: None,
        };
        let r = c.update_loop("nightly", &data).await.unwrap();
        assert_eq!(r.name, "Nightly v2");
    }

    #[tokio::test]
    async fn delete_loop() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/orgs/acme/loops/old-loop"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.delete_loop("old-loop").await.unwrap();
    }

    #[tokio::test]
    async fn enable_loop() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/loops/my-loop/enable"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.enable_loop("my-loop").await.unwrap();
    }

    #[tokio::test]
    async fn disable_loop() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/loops/my-loop/disable"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.disable_loop("my-loop").await.unwrap();
    }

    #[tokio::test]
    async fn trigger_loop() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/loops/my-loop/trigger"))
            .respond_with(ok(json!({
                "id":1,"loop_slug":"my-loop","status":"running"
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.trigger_loop("my-loop").await.unwrap();
        assert_eq!(r.status, agentsmesh_types::LoopRunStatus::Running);
    }

    #[tokio::test]
    async fn list_loop_runs() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/loops/my-loop/runs"))
            .and(query_param("status", "completed"))
            .respond_with(ok(json!({"runs":[],"total":0})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c
            .list_loop_runs("my-loop", Some("completed"), None, None)
            .await
            .unwrap();
        assert!(r.runs.is_empty());
    }

    #[tokio::test]
    async fn get_loop_run() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/loops/my-loop/runs/7"))
            .respond_with(ok(json!({
                "id":7,"loop_slug":"my-loop","status":"completed"
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.get_loop_run("my-loop", 7).await.unwrap();
        assert_eq!(r.id, 7);
        assert_eq!(r.status, agentsmesh_types::LoopRunStatus::Completed);
    }

    #[tokio::test]
    async fn cancel_loop_run() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/loops/my-loop/runs/7/cancel"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.cancel_loop_run("my-loop", 7).await.unwrap();
    }
}
