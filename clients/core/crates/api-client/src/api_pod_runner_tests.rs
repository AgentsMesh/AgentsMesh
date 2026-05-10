#[cfg(test)]
mod api_pod_runner_tests {
    use std::sync::{Arc, Mutex};

    use serde_json::json;
    use wiremock::matchers::{body_json, method, path, query_param};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    use crate::{ApiClient, AuthTokenStore};

    struct Tok(Mutex<Option<String>>);
    impl Tok {
        fn org(s: &str) -> Arc<Self> { Arc::new(Self(Mutex::new(Some(s.into())))) }
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

    fn pod_json(key: &str, status: &str) -> serde_json::Value {
        json!({"key": key, "status": status, "agent_slug": "claude"})
    }

    fn runner_json(id: i64) -> serde_json::Value {
        json!({
            "id": id, "name": "r1", "status": "online",
            "max_concurrent_pods": 5, "active_pod_count": 0, "is_enabled": true
        })
    }

    fn conn_json() -> serde_json::Value {
        json!({"relay_url":"wss://r.example.com","token":"t","pod_key":"p"})
    }

    #[tokio::test]
    async fn get_pod() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/pods/pod-abc"))
            .respond_with(ok(pod_json("pod-abc", "running")))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let r = c.get_pod("pod-abc").await.unwrap();
        assert_eq!(r.key, "pod-abc");
    }

    #[tokio::test]
    async fn create_pod() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/pods"))
            .respond_with(ok(json!({"pod": pod_json("pod-new", "creating")})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::CreatePodRequest {
            agent_slug: "claude".into(), agentfile_layer: None,
            runner_id: Some(1), alias: None, ticket_slug: None,
            cols: Some(80), rows: Some(24), source_pod_key: None,
            resume_agent_session: None, perpetual: None,
        };
        let r = c.create_pod(&data).await.unwrap();
        assert_eq!(r.pod.key, "pod-new");
    }

    #[tokio::test]
    async fn get_pod_connection_info() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/pods/p1/connect"))
            .respond_with(ok(conn_json())).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.get_pod_connection_info("p1").await.unwrap();
    }

    #[tokio::test]
    async fn get_pod_relay_connection() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/pods/p1/relay/connect"))
            .respond_with(ok(conn_json())).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.get_pod_relay_connection("p1").await.unwrap();
    }

    #[tokio::test]
    async fn update_pod_alias() {
        let s = MockServer::start().await;
        Mock::given(method("PATCH")).and(path("/api/v1/orgs/acme/pods/p1/alias"))
            .respond_with(ok(pod_json("p1", "running")))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::UpdatePodAliasRequest { alias: Some("my".into()) };
        let _ = c.update_pod_alias("p1", &data).await.unwrap();
    }

    #[tokio::test]
    async fn redeem_promo_code() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/orgs/acme/billing/promo-codes/redeem"))
            .and(body_json(json!({"code":"SAVE20"})))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::RedeemPromoRequest { code: "SAVE20".into() };
        let _ = c.redeem_promo_code(&data).await.unwrap();
    }

    #[tokio::test]
    async fn list_available_runners() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/runners/available"))
            .respond_with(ok(json!({"runners":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.list_available_runners().await.unwrap();
    }

    #[tokio::test]
    async fn get_runner() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/runners/3"))
            .respond_with(ok(json!({"runner": runner_json(3)})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let r = c.get_runner(3).await.unwrap();
        assert_eq!(r.runner.id, 3);
    }

    #[tokio::test]
    async fn update_runner() {
        let s = MockServer::start().await;
        Mock::given(method("PUT")).and(path("/api/v1/orgs/acme/runners/3"))
            .respond_with(ok(runner_json(3))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::UpdateRunnerRequest {
            description: Some("upd".into()), max_concurrent_pods: None,
            is_enabled: Some(true), visibility: None,
        };
        let _ = c.update_runner(3, &data).await.unwrap();
    }

    #[tokio::test]
    async fn delete_runner() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/orgs/acme/runners/3"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.delete_runner(3).await.unwrap();
    }

    #[tokio::test]
    async fn create_runner_token() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/runners/grpc/tokens"))
            .respond_with(ok(json!({"id":1}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::CreateRunnerTokenRequest {
            name: Some("dev".into()), labels: None,
            max_uses: None, expires_in_days: Some(30),
        };
        let _ = c.create_runner_token(&data).await.unwrap();
    }

    #[tokio::test]
    async fn list_runner_tokens() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/runners/grpc/tokens"))
            .respond_with(ok(json!({"tokens":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.list_runner_tokens().await.unwrap();
    }

    #[tokio::test]
    async fn delete_runner_token() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/orgs/acme/runners/grpc/tokens/9"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.delete_runner_token(9).await.unwrap();
    }

    #[tokio::test]
    async fn list_runner_pods() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/runners/3/pods"))
            .and(query_param("status", "running")).and(query_param("limit", "10"))
            .respond_with(ok(json!({"pods":[],"total":0}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.list_runner_pods(3, Some("running"), Some(10), None).await.unwrap();
    }

    #[tokio::test]
    async fn query_runner_sandboxes() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/runners/3/sandboxes/query"))
            .respond_with(ok(json!({"sandboxes":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::SandboxQueryRequest { pod_keys: vec!["p1".into()] };
        let _ = c.query_runner_sandboxes(3, &data).await.unwrap();
    }

    #[tokio::test]
    async fn upgrade_runner() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/runners/3/upgrade"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::UpgradeRunnerRequest {
            target_version: Some("1.2.0".into()), force: None,
        };
        let _ = c.upgrade_runner(3, &data).await.unwrap();
    }

    #[tokio::test]
    async fn request_runner_log_upload() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/runners/3/logs/upload"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.request_runner_log_upload(3).await.unwrap();
    }

    #[tokio::test]
    async fn list_runner_logs() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/runners/3/logs"))
            .respond_with(ok(json!({"logs":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.list_runner_logs(3).await.unwrap();
    }

    #[tokio::test]
    async fn get_runner_auth_status() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/runners/auth/k123"))
            .respond_with(ok(json!({"status":"pending"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.get_runner_auth_status("k123").await.unwrap();
    }

    #[tokio::test]
    async fn authorize_runner() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/runners/grpc/authorize"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::AuthorizeRunnerRequest {
            auth_key: "k".into(), node_id: None,
        };
        let _ = c.authorize_runner(&data).await.unwrap();
    }
}
