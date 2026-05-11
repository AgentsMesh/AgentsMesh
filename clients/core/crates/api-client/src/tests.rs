#[cfg(test)]
mod tests {
    use std::collections::HashMap;
    use std::sync::{Arc, Mutex};

    use serde::{Deserialize, Serialize};
    use wiremock::matchers::{header, method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    use crate::{ApiClient, ApiError, AuthTokenStore, RequestOptions};

    struct MockTokenStore {
        token: Mutex<Option<String>>,
        refresh_token: Mutex<Option<String>>,
        org_slug: Mutex<Option<String>>,
    }

    impl MockTokenStore {
        fn new(token: Option<&str>, refresh: Option<&str>, org: Option<&str>) -> Arc<Self> {
            Arc::new(Self {
                token: Mutex::new(token.map(String::from)),
                refresh_token: Mutex::new(refresh.map(String::from)),
                org_slug: Mutex::new(org.map(String::from)),
            })
        }
    }

    impl AuthTokenStore for MockTokenStore {
        fn get_token(&self) -> Option<String> {
            self.token.lock().unwrap().clone()
        }

        fn get_refresh_token(&self) -> Option<String> {
            self.refresh_token.lock().unwrap().clone()
        }

        fn set_tokens(&self, token: String, refresh_token: String, _expires_in_secs: Option<i64>) {
            *self.token.lock().unwrap() = Some(token);
            *self.refresh_token.lock().unwrap() = Some(refresh_token);
        }

        fn clear_tokens(&self) {
            *self.token.lock().unwrap() = None;
            *self.refresh_token.lock().unwrap() = None;
        }

        fn get_current_org_slug(&self) -> Option<String> {
            self.org_slug.lock().unwrap().clone()
        }
    }

    #[derive(Debug, Serialize, Deserialize, PartialEq)]
    struct TestPayload {
        message: String,
        count: i32,
    }

    // a. GET normal response → deserialize
    #[tokio::test]
    async fn get_normal_response() {
        let server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/test"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "ok", "count": 42})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let result: TestPayload = client.get("/api/v1/test").await.unwrap();
        assert_eq!(result.message, "ok");
        assert_eq!(result.count, 42);
    }

    // b. Empty response body → returns {}
    #[tokio::test]
    async fn empty_response_body() {
        let server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/empty"))
            .respond_with(ResponseTemplate::new(200))
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let result: serde_json::Value = client.get("/api/v1/empty").await.unwrap();
        assert_eq!(result, serde_json::json!({}));
    }

    // c. 401 auto-refresh → retry succeeds
    #[tokio::test]
    async fn auto_refresh_on_401() {
        let server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(path("/api/v1/data"))
            .and(header("Authorization", "Bearer old-tok"))
            .respond_with(ResponseTemplate::new(401).set_body_json(serde_json::json!({})))
            .up_to_n_times(1)
            .mount(&server)
            .await;

        Mock::given(method("POST"))
            .and(path("/api/v1/auth/refresh"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
                "token": "new-tok",
                "refresh_token": "new-ref",
                "expires_in": 3600
            })))
            .mount(&server)
            .await;

        Mock::given(method("GET"))
            .and(path("/api/v1/data"))
            .and(header("Authorization", "Bearer new-tok"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "refreshed", "count": 1})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("old-tok"), Some("ref-tok"), None);
        let client = ApiClient::new(server.uri(), store.clone());

        let result: TestPayload = client.get("/api/v1/data").await.unwrap();
        assert_eq!(result.message, "refreshed");
        assert_eq!(store.get_token(), Some("new-tok".into()));
    }

    // e. Refresh fails → AuthExpired
    #[tokio::test]
    async fn refresh_failure_returns_auth_expired() {
        let server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(path("/api/v1/data"))
            .respond_with(ResponseTemplate::new(401).set_body_json(serde_json::json!({})))
            .mount(&server)
            .await;

        Mock::given(method("POST"))
            .and(path("/api/v1/auth/refresh"))
            .respond_with(ResponseTemplate::new(401).set_body_json(serde_json::json!({})))
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("bad-tok"), Some("bad-ref"), None);
        let client = ApiClient::new(server.uri(), store.clone());

        let err = client.get::<serde_json::Value>("/api/v1/data").await.unwrap_err();
        assert!(matches!(err, ApiError::AuthExpired));
        assert!(store.get_token().is_none());
    }

    // f. ApiError parsing with code and error
    #[tokio::test]
    async fn api_error_parsing_422() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/validate"))
            .respond_with(ResponseTemplate::new(422).set_body_json(serde_json::json!({
                "code": "INVALID",
                "error": "bad input"
            })))
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let body = serde_json::json!({"field": "value"});
        let err = client.post::<serde_json::Value>("/api/v1/validate", &body).await.unwrap_err();
        match &err {
            ApiError::Http {
                status,
                code,
                server_message,
                url,
                ..
            } => {
                assert_eq!(*status, 422);
                assert_eq!(code.as_deref(), Some("INVALID"));
                assert_eq!(server_message.as_deref(), Some("bad input"));
                // Regression guard: parse_response must tag every Http
                // error with the request URL so wire errors are
                // self-describing (2026-05-10 OrbStack incident).
                let url = url.as_deref().expect("Http error must carry request url");
                assert!(url.ends_with("/api/v1/validate"), "got url: {url}");
            }
            _ => panic!("expected Http error, got: {err:?}"),
        }
        assert!(err.has_code("INVALID"));
        assert!(!err.has_code("OTHER"));
        assert_eq!(err.status(), Some(422));
    }

    // g. org_path splicing
    #[tokio::test]
    async fn org_path_with_slug() {
        let store = MockTokenStore::new(Some("tok"), None, Some("my-org"));
        let client = ApiClient::new("http://localhost".into(), store);
        assert_eq!(client.org_path("/pods"), "/api/v1/orgs/my-org/pods");
    }

    #[tokio::test]
    async fn org_path_without_slug() {
        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new("http://localhost".into(), store);
        assert_eq!(client.org_path("/pods"), "/api/v1/pods");
    }

    // h. public_get does not send Authorization header
    #[tokio::test]
    async fn public_get_no_authorization() {
        let server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/public"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "public", "count": 0})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("should-not-appear"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let result: TestPayload = client.public_get("/api/v1/public").await.unwrap();
        assert_eq!(result.message, "public");

        let requests = server.received_requests().await.unwrap();
        let auth_header = requests[0].headers.get("Authorization");
        assert!(auth_header.is_none(), "public_get should not send Authorization");
    }

    // i. Custom headers
    #[tokio::test]
    async fn custom_headers_sent() {
        let server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/pods"))
            .and(header("X-Pod-Key", "pod-123"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "with-header", "count": 1})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let mut headers = HashMap::new();
        headers.insert("X-Pod-Key".into(), "pod-123".into());
        let opts = RequestOptions {
            headers: Some(headers),
            ..Default::default()
        };

        let result: TestPayload = client
            .request(reqwest::Method::GET, "/api/v1/pods", opts)
            .await
            .unwrap();
        assert_eq!(result.message, "with-header");
    }

    // j. Network error — connect to non-existent port
    #[tokio::test]
    async fn network_error_unreachable() {
        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new("http://127.0.0.1:1".into(), store);

        let err = client.get::<serde_json::Value>("/api/v1/test").await.unwrap_err();
        assert!(matches!(err, ApiError::Network(_)));
    }

    // POST request sends body
    #[tokio::test]
    async fn post_sends_body() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/items"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "created", "count": 1})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let body = TestPayload {
            message: "new item".into(),
            count: 0,
        };
        let result: TestPayload = client.post("/api/v1/items", &body).await.unwrap();
        assert_eq!(result.message, "created");
    }

    // PUT request
    #[tokio::test]
    async fn put_request() {
        let server = MockServer::start().await;
        Mock::given(method("PUT"))
            .and(path("/api/v1/items/1"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "updated", "count": 1})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let body = serde_json::json!({"name": "updated"});
        let result: TestPayload = client.put("/api/v1/items/1", &body).await.unwrap();
        assert_eq!(result.message, "updated");
    }

    // DELETE request
    #[tokio::test]
    async fn delete_request() {
        let server = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/items/1"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({})))
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let result: serde_json::Value = client.delete("/api/v1/items/1").await.unwrap();
        assert_eq!(result, serde_json::json!({}));
    }

    // PATCH request
    #[tokio::test]
    async fn patch_request() {
        let server = MockServer::start().await;
        Mock::given(method("PATCH"))
            .and(path("/api/v1/items/1"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "patched", "count": 1})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let body = serde_json::json!({"field": "value"});
        let result: TestPayload = client.patch("/api/v1/items/1", &body).await.unwrap();
        assert_eq!(result.message, "patched");
    }

    // public_post
    #[tokio::test]
    async fn public_post_no_auth() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/auth/login"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "logged-in", "count": 0})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let body = serde_json::json!({"email": "a@b.com"});
        let result: TestPayload = client.public_post("/api/v1/auth/login", &body).await.unwrap();
        assert_eq!(result.message, "logged-in");

        let requests = server.received_requests().await.unwrap();
        assert!(requests[0].headers.get("Authorization").is_none());
    }

    // ApiError status() for non-Http variants
    #[tokio::test]
    async fn api_error_status_none_for_non_http() {
        let err = ApiError::AuthExpired;
        assert!(err.status().is_none());
        assert!(!err.has_code("any"));
    }

    // Refresh with no refresh token clears auth
    #[tokio::test]
    async fn refresh_without_refresh_token() {
        let server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(path("/api/v1/data"))
            .respond_with(ResponseTemplate::new(401).set_body_json(serde_json::json!({})))
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store.clone());

        let err = client.get::<serde_json::Value>("/api/v1/data").await.unwrap_err();
        assert!(matches!(err, ApiError::AuthExpired));
        assert!(store.get_token().is_none());
    }

    // Authenticated GET sends Bearer token
    #[tokio::test]
    async fn get_sends_bearer_token() {
        let server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/me"))
            .and(header("Authorization", "Bearer my-secret-token"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "authed", "count": 0})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("my-secret-token"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let result: TestPayload = client.get("/api/v1/me").await.unwrap();
        assert_eq!(result.message, "authed");
    }

    // MockTokenStore trait operations
    #[test]
    fn mock_token_store_operations() {
        let store = MockTokenStore::new(Some("t"), Some("r"), Some("org"));
        assert_eq!(store.get_token(), Some("t".into()));
        assert_eq!(store.get_refresh_token(), Some("r".into()));
        assert_eq!(store.get_current_org_slug(), Some("org".into()));

        store.set_tokens("new-t".into(), "new-r".into(), None);
        assert_eq!(store.get_token(), Some("new-t".into()));
        assert_eq!(store.get_refresh_token(), Some("new-r".into()));

        store.clear_tokens();
        assert!(store.get_token().is_none());
        assert!(store.get_refresh_token().is_none());
    }

    // d. Concurrent 401 refresh — token already refreshed by another request
    #[tokio::test]
    async fn concurrent_401_token_already_refreshed() {
        let server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(path("/api/v1/data"))
            .and(header("Authorization", "Bearer new-tok"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_json(serde_json::json!({"message": "ok", "count": 0})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("new-tok"), Some("ref"), None);
        let client = ApiClient::new(server.uri(), store.clone());

        let refreshed = client.handle_token_refresh(Some("old-tok")).await;
        assert!(refreshed, "should succeed since current token != failed token");
    }

    // 500 error without code/error fields
    #[tokio::test]
    async fn server_error_500_no_code() {
        let server = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/fail"))
            .respond_with(
                ResponseTemplate::new(500).set_body_json(serde_json::json!({"detail": "internal"})),
            )
            .mount(&server)
            .await;

        let store = MockTokenStore::new(Some("tok"), None, None);
        let client = ApiClient::new(server.uri(), store);

        let err = client.get::<serde_json::Value>("/api/v1/fail").await.unwrap_err();
        match err {
            ApiError::Http {
                status,
                code,
                server_message,
                url,
                ..
            } => {
                assert_eq!(status, 500);
                assert!(code.is_none());
                assert!(server_message.is_none());
                let url = url.expect("Http error must carry request url");
                assert!(url.ends_with("/api/v1/fail"), "got url: {url}");
            }
            _ => panic!("expected Http error"),
        }
    }
}
