#[cfg(test)]
mod api_core_tests {
    use std::sync::{Arc, Mutex};

    use serde_json::json;
    use wiremock::matchers::{method, path, query_param};
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
        fn set_tokens(&self, _t: String, _r: String, _e: Option<i64>) {}
        fn clear_tokens(&self) {}
        fn get_current_org_slug(&self) -> Option<String> {
            self.org_slug.lock().unwrap().clone()
        }
    }

    fn ok(body: serde_json::Value) -> ResponseTemplate {
        ResponseTemplate::new(200).set_body_json(body)
    }

    // ── pod ─────────────────────────────────────────────────────────────
    // Pod tests removed: REST surface eliminated; Connect handler tests in
    // backend/internal/api/connect/pod cover the same surface.

    // ── ticket ──────────────────────────────────────────────────────────
    // ticket REST mocks removed: REST surface eliminated; Connect handler
    // tests in backend/internal/api/connect/ticket cover the same surface.

    // ── runner ──────────────────────────────────────────────────────────
    // list_runners + sibling REST mocks removed: REST surface eliminated;
    // Connect handler tests in backend/internal/api/connect/runner cover
    // the same surface.

    // ── billing ─────────────────────────────────────────────────────────
    // get_billing_overview removed — Connect handler tests cover it
    // (backend/internal/api/connect/billing).

    // ── mesh ────────────────────────────────────────────────────────────
    // get_mesh_topology REST mock removed: REST surface eliminated;
    // Connect handler tests in backend/internal/api/connect/mesh cover
    // the same surface.

    // ── loop ────────────────────────────────────────────────────────────
    // REST surface dropped; covered by loop_connect.rs.

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

    // ── autopilot ───────────────────────────────────────────────────────
    // REST surface dropped; covered by autopilot_connect.rs.

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

    // ── file ────────────────────────────────────────────────────────────
    // REST `files/presign` removed; covered by file_connect.rs.


    // ── invitation ──────────────────────────────────────────────────────
    // REST surface dropped; covered by invitation_connect.rs.

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
    // REST surface dropped; covered by notification_connect.rs.

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
    // REST surface dropped; validate / redeem / history all live on
    // proto.promocode.v1.PromoCodeService — covered by
    // promocode_connect.rs and the wasm service tests.

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

    // ── ticket_relations ────────────────────────────────────────────────
    // REST mocks removed: REST surface eliminated; Connect handler tests in
    // backend/internal/api/connect/ticket_relations cover the same surface.

    // ── token_usage ─────────────────────────────────────────────────────
    // REST surface dropped; covered by token_usage_connect.rs.

    // ── user_agent_credential ───────────────────────────────────────────
    // REST surface dropped; covered by user_agent_credential_connect.rs.

    // ── user_git_credential ─────────────────────────────────────────────
    // REST surface dropped; covered by user_git_credential_connect.rs.

    // ── user_repository_provider ────────────────────────────────────────
    // REST surface dropped; covered by user_repository_provider_connect.rs.
}
