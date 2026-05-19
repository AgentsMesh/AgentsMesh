#[cfg(test)]
mod api_pod_runner_tests {
    use std::sync::{Arc, Mutex};

    use serde_json::json;
    use wiremock::matchers::{method, path, query_param};
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

    // Pod tests removed: REST surface eliminated; Connect handler tests in
    // backend/internal/api/connect/pod cover the same surface.
    //
    // Most runner REST mocks removed for the same reason; Connect handler
    // tests in backend/internal/api/connect/runner own the surface. The
    // mocks that remain cover REST carve-outs without proto coverage:
    // list_runner_pods (no runner_id filter in proto.pod.v1.ListPods),
    // get_runner_auth_status + authorize_runner (registration bootstrap).

    // redeem_promo_code dropped — REST surface gone, see promocode_connect.

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
