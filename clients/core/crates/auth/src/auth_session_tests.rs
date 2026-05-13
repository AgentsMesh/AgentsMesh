#[cfg(test)]
mod auth_session_tests {
    use agentsmesh_types::proto_auth_v1 as auth_proto;
    use agentsmesh_types::RegisterRequest;
    use prost::Message;
    use wiremock::matchers::{header, method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    use crate::manager::AuthManager;
    use crate::state::session_storage_key;
    use crate::storage::PersistentStorage;
    use crate::test_support::InMemoryStorage;

    fn login_resp_proto() -> Vec<u8> {
        auth_proto::LoginResponse {
            token: "access-tok".into(),
            refresh_token: "refresh-tok".into(),
            expires_in: 3600,
            user: Some(auth_proto::User {
                id: 1,
                email: "dev@test.com".into(),
                username: "dev".into(),
                name: Some("Dev User".into()),
                avatar_url: None,
                is_email_verified: Some(true),
            }),
        }
        .encode_to_vec()
    }

    fn register_resp_proto() -> Vec<u8> {
        auth_proto::RegisterResponse {
            token: "access-tok".into(),
            refresh_token: "refresh-tok".into(),
            expires_in: 3600,
            user: Some(auth_proto::User {
                id: 1,
                email: "dev@test.com".into(),
                username: "dev".into(),
                name: Some("Dev User".into()),
                avatar_url: None,
                is_email_verified: Some(false),
            }),
            message: None,
        }
        .encode_to_vec()
    }

    fn connect_error(status: u16, msg: &str) -> ResponseTemplate {
        ResponseTemplate::new(status).set_body_json(serde_json::json!({"message": msg}))
    }

    #[tokio::test]
    async fn login_success() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/Login"))
            .and(header("content-type", "application/proto"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_bytes(login_resp_proto())
                    .insert_header("content-type", "application/proto"),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage.clone());

        let session = manager.login("dev@test.com", "pass").await.unwrap();
        assert_eq!(session.token, "access-tok");
        assert_eq!(session.user.email, "dev@test.com");
        assert!(manager.is_authenticated());
        let key = session_storage_key(&server.uri());
        assert!(storage.get(&key).is_some());
        assert!(storage.get("agentsmesh-auth").is_none());
    }

    #[tokio::test]
    async fn login_failure_401() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/Login"))
            .respond_with(connect_error(401, "invalid credentials"))
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        let err = manager.login("bad@test.com", "wrong").await.unwrap_err();
        match err {
            crate::AuthError::Server { status, message, .. } => {
                assert_eq!(status, 401);
                assert!(message.contains("invalid credentials"));
            }
            _ => panic!("expected Server error, got: {err:?}"),
        }
    }

    #[tokio::test]
    async fn register_success() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/Register"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_bytes(register_resp_proto())
                    .insert_header("content-type", "application/proto"),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        let req = RegisterRequest {
            name: "Dev".into(),
            email: "dev@test.com".into(),
            username: "dev".into(),
            password: "pass123".into(),
        };
        let session = manager.register(&req).await.unwrap();
        assert_eq!(session.user.username, "dev");
        assert!(manager.is_authenticated());
    }

    #[tokio::test]
    async fn register_failure() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/Register"))
            .respond_with(connect_error(409, "email taken"))
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        let req = RegisterRequest {
            name: "Dev".into(),
            email: "dup@test.com".into(),
            username: "dev".into(),
            password: "pass123".into(),
        };
        let err = manager.register(&req).await.unwrap_err();
        assert!(matches!(err, crate::AuthError::Server { status: 409, .. }));
    }

    #[tokio::test]
    async fn logout_success() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/Login"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_bytes(login_resp_proto())
                    .insert_header("content-type", "application/proto"),
            )
            .mount(&server)
            .await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthSessionService/Logout"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_bytes(
                        auth_proto::LogoutResponse {
                            message: "ok".into(),
                        }
                        .encode_to_vec(),
                    )
                    .insert_header("content-type", "application/proto"),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage.clone());

        manager.login("dev@test.com", "pass").await.unwrap();
        assert!(manager.is_authenticated());

        manager.logout().await.unwrap();
        assert!(!manager.is_authenticated());
        assert!(manager.current_user().is_none());
        let key = session_storage_key(&server.uri());
        assert!(storage.get(&key).is_none());
    }

    #[tokio::test]
    async fn logout_without_auth() {
        let storage = InMemoryStorage::new();
        let manager = AuthManager::new("http://localhost".into(), storage);
        let err = manager.logout().await.unwrap_err();
        assert!(matches!(err, crate::AuthError::NotAuthenticated));
    }

    #[tokio::test]
    async fn logout_server_5xx_still_clears_local() {
        // Plan I3 invariant: server-side logout failure must NOT leave
        // the renderer with `is_authenticated() == true`. We login, then
        // mount a 500 on Logout, and assert local state is dropped regardless.
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/Login"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_bytes(login_resp_proto())
                    .insert_header("content-type", "application/proto"),
            )
            .mount(&server)
            .await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthSessionService/Logout"))
            .respond_with(ResponseTemplate::new(500))
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage.clone());

        manager.login("dev@test.com", "pass").await.unwrap();
        assert!(manager.is_authenticated());

        manager.logout().await.unwrap();
        assert!(!manager.is_authenticated());
        assert!(manager.current_user().is_none());
        let key = session_storage_key(&server.uri());
        assert!(storage.get(&key).is_none());
    }
}
