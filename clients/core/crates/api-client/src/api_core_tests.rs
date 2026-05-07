#[cfg(test)]
mod api_core_tests {
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

    // ── user ────────────────────────────────────────────────────────────

    #[tokio::test]
    async fn get_me() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/me"))
            .respond_with(ok(json!({"id":1,"email":"u@a.com","username":"u"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let r = c.get_me().await.unwrap();
        assert_eq!(r.email, "u@a.com");
    }

    // ── pod ─────────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_pods_with_filters() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/pods"))
            .and(query_param("status", "running")).and(query_param("limit", "10"))
            .respond_with(ok(json!({"pods":[],"total":0})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_pods(Some("running"), None, None, Some(10), None).await.unwrap();
        assert_eq!(r.total, Some(0));
    }

    #[tokio::test]
    async fn list_pods_no_filters() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/pods"))
            .respond_with(ok(json!({"pods":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_pods(None, None, None, None, None).await.unwrap();
        assert!(r.pods.is_empty());
    }

    #[tokio::test]
    async fn terminate_pod() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/pods/pod-abc/terminate"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.terminate_pod("pod-abc").await.unwrap();
    }

    // ── channel ─────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_channels() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/channels"))
            .respond_with(ok(json!({"channels":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_channels(None).await.unwrap();
    }

    // ── ticket ──────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_tickets() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/tickets"))
            .and(query_param("status", "open"))
            .respond_with(ok(json!({"tickets":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_tickets(Some("open"), None, None).await.unwrap();
    }

    #[tokio::test]
    async fn get_ticket_board() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/tickets/board"))
            .and(query_param("repository_id", "42"))
            .respond_with(ok(json!({"columns":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_ticket_board(Some(42)).await.unwrap();
    }

    // ── runner ──────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_runners() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/runners"))
            .and(query_param("status", "online"))
            .respond_with(ok(json!({"runners":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_runners(Some("online")).await.unwrap();
    }

    // ── billing ─────────────────────────────────────────────────────────

    #[tokio::test]
    async fn get_billing_overview() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/billing/overview"))
            .respond_with(ok(json!({"plan":"pro"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_billing_overview().await.unwrap();
    }

    // ── sso ─────────────────────────────────────────────────────────────

    #[tokio::test]
    async fn sso_discover() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/auth/sso/discover"))
            .and(query_param("email", "a@b.com"))
            .respond_with(ok(json!({"configs":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.sso_discover("a@b.com").await.unwrap();
        let reqs = s.received_requests().await.unwrap();
        assert!(reqs[0].headers.get("Authorization").is_none());
    }

    // ── mesh ────────────────────────────────────────────────────────────

    #[tokio::test]
    async fn get_mesh_topology() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/mesh/topology"))
            .respond_with(ok(json!({"nodes":[],"edges":[],"channels":[],"runners":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.get_mesh_topology().await.unwrap();
        assert!(r.nodes.is_empty());
    }

    // ── loop ────────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_loops() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/loops"))
            .and(query_param("status", "active"))
            .respond_with(ok(json!({"loops":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_loops(Some("active"), None, None).await.unwrap();
    }

    // ── agent ───────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_agents() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/agents"))
            .respond_with(ok(json!({"agents":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.list_agents().await.unwrap();
        assert!(r.agents.is_empty());
    }

    #[tokio::test]
    async fn get_agent_config_schema() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/agents/claude/config-schema"))
            .respond_with(ok(json!({"fields":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_agent_config_schema("claude").await.unwrap();
    }

    // ── agentpod ────────────────────────────────────────────────────────

    #[tokio::test]
    async fn get_agentpod_settings() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/me/agentpod/settings"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.get_agentpod_settings().await.unwrap();
    }

    #[tokio::test]
    async fn list_agentpod_providers() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/me/agentpod/providers"))
            .respond_with(ok(json!({"providers":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.list_agentpod_providers().await.unwrap();
    }

    // ── apikey ──────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_api_keys() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/api-keys"))
            .respond_with(ok(json!({"api_keys":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_api_keys().await.unwrap();
    }

    #[tokio::test]
    async fn revoke_api_key() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/api-keys/7/revoke"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.revoke_api_key(7).await.unwrap();
    }

    // ── autopilot ───────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_autopilots() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/autopilot-controllers"))
            .respond_with(ok(json!([])))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_autopilots().await.unwrap();
    }

    #[tokio::test]
    async fn pause_autopilot() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/autopilot-controllers/ctrl-1/pause"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.pause_autopilot("ctrl-1").await.unwrap();
    }

    // ── billing_public ──────────────────────────────────────────────────

    #[tokio::test]
    async fn get_public_pricing() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/config/pricing"))
            .respond_with(ok(json!({"deployment_type":"global","currency":"USD","plans":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.get_public_pricing().await.unwrap();
        let reqs = s.received_requests().await.unwrap();
        assert!(reqs[0].headers.get("Authorization").is_none());
    }

    #[tokio::test]
    async fn get_public_deployment_info() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/config/deployment"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.get_public_deployment_info().await.unwrap();
    }

    // ── binding ─────────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_bindings() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/bindings"))
            .and(query_param("status", "active"))
            .respond_with(ok(json!({"bindings":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_bindings(Some("active")).await.unwrap();
    }

    #[tokio::test]
    async fn get_pending_bindings() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/bindings/pending"))
            .respond_with(ok(json!({"bindings":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_pending_bindings().await.unwrap();
    }

    // ── extension ───────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_skill_registries() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/skill-registries"))
            .respond_with(ok(json!({"registries":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_skill_registries().await.unwrap();
    }

    #[tokio::test]
    async fn list_market_skills() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/market/skills"))
            .and(query_param("q", "git"))
            .respond_with(ok(json!({"skills":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_market_skills(Some("git"), None).await.unwrap();
    }

    // ── file ────────────────────────────────────────────────────────────

    #[tokio::test]
    async fn presign_file_upload() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/files/presign"))
            .respond_with(ok(json!({
                "put_url":"https://s3.example.com/upload",
                "get_url":"https://s3.example.com/download"
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::PresignRequest {
            filename: "test.txt".into(),
            content_type: "text/plain".into(),
            size: 1024,
        };
        let r = c.presign_file_upload(&data).await.unwrap();
        assert_eq!(r.put_url, "https://s3.example.com/upload");
    }

    // ── invitation ──────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_invitations() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/invitations"))
            .respond_with(ok(json!({"invitations":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_invitations().await.unwrap();
    }

    #[tokio::test]
    async fn get_invitation_by_token() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/invitations/abc123"))
            .respond_with(ok(json!({
                "id":1,"token":"abc123","email":"a@b.com","role":"member"
            })))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.get_invitation_by_token("abc123").await.unwrap();
        let reqs = s.received_requests().await.unwrap();
        assert!(reqs[0].headers.get("Authorization").is_none());
    }

    // ── message ─────────────────────────────────────────────────────────

    #[tokio::test]
    async fn get_mesh_messages() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/messages"))
            .and(query_param("unread_only", "true")).and(query_param("limit", "20"))
            .respond_with(ok(json!({"messages":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_mesh_messages(Some(true), Some(20), None).await.unwrap();
    }

    #[tokio::test]
    async fn get_mesh_unread_count() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/messages/unread-count"))
            .respond_with(ok(json!({"count":5})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let r = c.get_mesh_unread_count().await.unwrap();
        assert_eq!(r.count, 5);
    }

    // ── notification ────────────────────────────────────────────────────

    #[tokio::test]
    async fn get_notification_preferences() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/notifications/preferences"))
            .respond_with(ok(json!({"preferences":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_notification_preferences().await.unwrap();
    }

    // ── organization ────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_organizations() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs"))
            .respond_with(ok(json!({"organizations":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.list_organizations().await.unwrap();
    }

    #[tokio::test]
    async fn get_organization() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme"))
            .respond_with(ok(json!({"id":1,"slug":"acme","name":"Acme"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let r = c.get_organization("acme").await.unwrap();
        assert_eq!(r.slug, "acme");
    }

    // ── promocode ───────────────────────────────────────────────────────

    #[tokio::test]
    async fn validate_promo_code() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/billing/promo-codes/validate"))
            .and(body_json(json!({"code":"SAVE20"})))
            .respond_with(ok(json!({"valid":true})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let data = agentsmesh_types::ValidatePromoRequest {
            code: "SAVE20".into(),
        };
        let r = c.validate_promo_code(&data).await.unwrap();
        assert!(r.valid);
    }

    #[tokio::test]
    async fn get_promo_code_history() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/billing/promo-codes/history"))
            .respond_with(ok(json!({"history":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_promo_code_history().await.unwrap();
    }

    // ── repository ──────────────────────────────────────────────────────

    #[tokio::test]
    async fn list_repositories() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/repositories"))
            .respond_with(ok(json!({"repositories":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_repositories().await.unwrap();
    }

    #[tokio::test]
    async fn list_repository_branches() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/repositories/5/branches"))
            .respond_with(ok(json!({"branches":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_repository_branches(5).await.unwrap();
    }

    // ── support_ticket ──────────────────────────────────────────────────

    #[tokio::test]
    async fn list_support_tickets() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/support-tickets"))
            .and(query_param("status", "open")).and(query_param("page", "1"))
            .respond_with(ok(json!({"data":[],"total":0})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.list_support_tickets(Some("open"), Some(1), None).await.unwrap();
    }

    #[tokio::test]
    async fn get_support_ticket_detail() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/support-tickets/42"))
            .respond_with(ok(json!({"id":42,"title":"help"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.get_support_ticket_detail(42).await.unwrap();
    }

    // ── ticket_relations ────────────────────────────────────────────────

    #[tokio::test]
    async fn list_ticket_relations() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/tickets/TK-1/relations"))
            .respond_with(ok(json!({"relations":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_ticket_relations("TK-1").await.unwrap();
    }

    #[tokio::test]
    async fn list_ticket_comments() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/tickets/TK-1/comments"))
            .and(query_param("limit", "10"))
            .respond_with(ok(json!({"comments":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_ticket_comments("TK-1", Some(10), None).await.unwrap();
    }

    #[tokio::test]
    async fn list_ticket_commits() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/tickets/TK-1/commits"))
            .respond_with(ok(json!({"commits":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.list_ticket_commits("TK-1").await.unwrap();
    }

    // ── token_usage ─────────────────────────────────────────────────────

    #[tokio::test]
    async fn get_token_usage_dashboard() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/token-usage/dashboard"))
            .and(query_param("granularity", "daily"))
            .respond_with(ok(json!({"total_input_tokens":0,"total_output_tokens":0})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::with_org("acme"));
        let _ = c.get_token_usage_dashboard(
            None, None, None, None, None, Some("daily"),
        ).await.unwrap();
    }

    // ── user_agent_credential ───────────────────────────────────────────

    #[tokio::test]
    async fn list_user_agent_credentials() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/agent-credentials"))
            .respond_with(ok(json!({"profiles":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.list_user_agent_credentials().await.unwrap();
    }

    #[tokio::test]
    async fn set_default_agent_credential() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/users/agent-credentials/profiles/3/set-default"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.set_default_agent_credential(3).await.unwrap();
    }

    // ── user_git_credential ─────────────────────────────────────────────

    #[tokio::test]
    async fn list_user_git_credentials() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/git-credentials"))
            .respond_with(ok(json!({"credentials":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.list_user_git_credentials().await.unwrap();
    }

    #[tokio::test]
    async fn get_default_git_credential() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/git-credentials/default"))
            .respond_with(ok(json!({"id":1,"name":"default"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.get_default_git_credential().await.unwrap();
    }

    // ── user_repository_provider ────────────────────────────────────────

    #[tokio::test]
    async fn list_user_repository_providers() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/repository-providers"))
            .respond_with(ok(json!({"providers":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.list_user_repository_providers().await.unwrap();
    }

    #[tokio::test]
    async fn test_repository_provider_connection() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/users/repository-providers/2/test"))
            .respond_with(ok(json!({})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.test_repository_provider_connection(2).await.unwrap();
    }

    #[tokio::test]
    async fn list_provider_repositories() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/users/repository-providers/2/repositories"))
            .and(query_param("search", "demo"))
            .respond_with(ok(json!({"repositories":[]})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), MockTokenStore::no_org());
        let _ = c.list_provider_repositories(2, None, None, Some("demo")).await.unwrap();
    }
}
