#[cfg(test)]
mod api_repo_tests {
    use std::sync::{Arc, Mutex};

    use serde_json::json;
    use wiremock::matchers::{method, path, query_param};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    use crate::{ApiClient, AuthTokenStore};

    struct Tok(Mutex<Option<String>>);
    impl Tok {
        fn org(s: &str) -> Arc<Self> { Arc::new(Self(Mutex::new(Some(s.into())))) }
        fn none() -> Arc<Self> { Arc::new(Self(Mutex::new(None))) }
    }
    impl AuthTokenStore for Tok {
        fn get_token(&self) -> Option<String> { Some("tok".into()) }
        fn get_refresh_token(&self) -> Option<String> { None }
        fn set_tokens(&self, _t: String, _r: String, _e: Option<i64>) {}
        fn clear_tokens(&self) {}
        fn get_current_org_slug(&self) -> Option<String> { self.0.lock().unwrap().clone() }
    }

    fn ok(b: serde_json::Value) -> ResponseTemplate {
        ResponseTemplate::new(200).set_body_json(b)
    }

    #[tokio::test]
    async fn get_repository() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/repositories/5"))
            .respond_with(ok(json!({"id":5,"name":"demo"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let r = c.get_repository(5).await.unwrap();
        assert_eq!(r.name, "demo");
    }

    #[tokio::test]
    async fn create_repository() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/repositories"))
            .respond_with(ok(json!({"id":6,"name":"new-repo"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::CreateRepositoryRequest {
            provider_type: None, provider_base_url: None, http_clone_url: None,
            ssh_clone_url: None, external_id: None, name: "new-repo".into(),
            slug: None, default_branch: None, ticket_prefix: None, visibility: None,
        };
        let r = c.create_repository(&data).await.unwrap();
        assert_eq!(r.name, "new-repo");
    }

    #[tokio::test]
    async fn update_repository() {
        let s = MockServer::start().await;
        Mock::given(method("PUT")).and(path("/api/v1/orgs/acme/repositories/5"))
            .respond_with(ok(json!({"id":5,"name":"upd"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::UpdateRepositoryRequest {
            name: Some("upd".into()), default_branch: None, ticket_prefix: None,
            is_active: None, http_clone_url: None, ssh_clone_url: None,
        };
        let _ = c.update_repository(5, &data).await.unwrap();
    }

    #[tokio::test]
    async fn delete_repository() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/orgs/acme/repositories/5"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.delete_repository(5).await.unwrap();
    }

    #[tokio::test]
    async fn sync_repository_branches() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/repositories/5/sync-branches"))
            .respond_with(ok(json!({"branches":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::SyncBranchesRequest { access_token: None };
        let _ = c.sync_repository_branches(5, &data).await.unwrap();
    }

    #[tokio::test]
    async fn register_repository_webhook() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/repositories/5/webhook"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.register_repository_webhook(5).await.unwrap();
    }

    #[tokio::test]
    async fn delete_repository_webhook() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/orgs/acme/repositories/5/webhook"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.delete_repository_webhook(5).await.unwrap();
    }

    #[tokio::test]
    async fn get_repository_webhook_status() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/repositories/5/webhook/status"))
            .respond_with(ok(json!({"configured":true}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.get_repository_webhook_status(5).await.unwrap();
    }

    #[tokio::test]
    async fn get_repository_webhook_secret() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/repositories/5/webhook/secret"))
            .respond_with(ok(json!({"secret":"s3cr3t"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let r = c.get_repository_webhook_secret(5).await.unwrap();
        assert_eq!(r.secret, "s3cr3t");
    }

    #[tokio::test]
    async fn list_repository_merge_requests() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/repositories/5/merge-requests"))
            .and(query_param("branch", "main")).and(query_param("state", "open"))
            .respond_with(ok(json!({"merge_requests":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c
            .list_repository_merge_requests(5, Some("main"), Some("open"))
            .await
            .unwrap();
    }

    #[tokio::test]
    async fn get_token_usage_dashboard_all_params() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/token-usage/dashboard"))
            .and(query_param("start_time", "2026-01-01"))
            .and(query_param("end_time", "2026-01-31"))
            .and(query_param("agent_slug", "claude"))
            .and(query_param("user_id", "1"))
            .and(query_param("model", "opus"))
            .and(query_param("granularity", "hourly"))
            .respond_with(ok(json!({"total_input_tokens":100,"total_output_tokens":200})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.get_token_usage_dashboard(
            Some("2026-01-01"), Some("2026-01-31"),
            Some("claude"), Some(1), Some("opus"), Some("hourly"),
        ).await.unwrap();
    }
}
