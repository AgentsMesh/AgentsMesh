#[cfg(test)]
mod bootstrap_tests {
    use agentsmesh_types::proto_auth_v1 as auth_proto;
    use agentsmesh_types::proto_org_v1 as org_proto;
    use agentsmesh_types::proto_user_v1 as user_proto;
    use prost::Message;
    use wiremock::matchers::{method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    use crate::bootstrap::{BootstrapCleanupReason, BootstrapResult};
    use crate::manager::{now_unix_secs, AuthManager};
    use crate::state::{session_storage_key, PersistedSession, SCHEMA_VERSION};
    use crate::storage::PersistentStorage;
    use crate::test_support::InMemoryStorage;

    fn proto_response(bytes: Vec<u8>) -> ResponseTemplate {
        ResponseTemplate::new(200)
            .set_body_bytes(bytes)
            .insert_header("content-type", "application/proto")
    }

    fn me_proto() -> Vec<u8> {
        user_proto::User {
            id: 1,
            email: "dev@test.com".into(),
            username: "dev".into(),
            name: Some("Dev User".into()),
            avatar_url: None,
            is_active: true,
            is_system_admin: false,
            is_email_verified: true,
            last_login_at: None,
            default_git_credential_id: None,
            created_at: "2026-01-01T00:00:00Z".into(),
            updated_at: "2026-01-01T00:00:00Z".into(),
        }
        .encode_to_vec()
    }

    fn mock_org(id: i64, name: &str, slug: &str) -> org_proto::Organization {
        org_proto::Organization {
            id,
            name: name.into(),
            slug: slug.into(),
            logo_url: None,
            subscription_plan: String::new(),
            subscription_status: String::new(),
            role: None,
            created_at: "2026-01-01T00:00:00Z".into(),
            updated_at: "2026-01-01T00:00:00Z".into(),
        }
    }

    fn orgs_proto() -> Vec<u8> {
        org_proto::ListMyOrgsResponse {
            items: vec![mock_org(10, "Org A", "org-a"), mock_org(11, "Org B", "org-b")],
            total: 2,
            limit: 50,
            offset: 0,
        }
        .encode_to_vec()
    }

    fn install_session(storage: &InMemoryStorage, base_url: &str, session: PersistedSession) {
        let key = session_storage_key(base_url);
        let json = serde_json::to_string(&session).unwrap();
        storage.set(&key, &json);
    }

    fn fresh_session(base_url: &str) -> PersistedSession {
        PersistedSession {
            access_token: "live-tok".into(),
            refresh_token: "live-ref".into(),
            expires_at: now_unix_secs() + 3600,
            base_url: base_url.into(),
            current_org_slug: None,
            schema_version: SCHEMA_VERSION,
        }
    }

    async fn mount_me(server: &MockServer) {
        Mock::given(method("POST"))
            .and(path("/proto.user.v1.UserService/GetMe"))
            .respond_with(proto_response(me_proto()))
            .mount(server)
            .await;
    }

    async fn mount_orgs(server: &MockServer) {
        Mock::given(method("POST"))
            .and(path("/proto.org.v1.OrgService/ListMyOrgs"))
            .respond_with(proto_response(orgs_proto()))
            .mount(server)
            .await;
    }

    #[tokio::test]
    async fn happy_path_authenticates_and_picks_first_org() {
        let server = MockServer::start().await;
        mount_me(&server).await;
        mount_orgs(&server).await;

        let storage = InMemoryStorage::new();
        install_session(&storage, &server.uri(), fresh_session(&server.uri()));
        let manager = AuthManager::new(server.uri(), storage);

        match manager.bootstrap().await {
            BootstrapResult::Authenticated { user, current_org } => {
                assert_eq!(user.email, "dev@test.com");
                assert_eq!(current_org.unwrap().slug, "org-a");
            }
            other => panic!("expected Authenticated, got {other:?}"),
        }
    }

    #[tokio::test]
    async fn happy_path_honors_persisted_org_slug() {
        let server = MockServer::start().await;
        mount_me(&server).await;
        mount_orgs(&server).await;

        let storage = InMemoryStorage::new();
        let mut session = fresh_session(&server.uri());
        session.current_org_slug = Some("org-b".into());
        install_session(&storage, &server.uri(), session);
        let manager = AuthManager::new(server.uri(), storage);

        match manager.bootstrap().await {
            BootstrapResult::Authenticated { current_org, .. } => {
                assert_eq!(current_org.unwrap().slug, "org-b");
            }
            other => panic!("expected Authenticated, got {other:?}"),
        }
    }

    #[tokio::test]
    async fn empty_storage_is_anonymous() {
        let storage = InMemoryStorage::new();
        let manager = AuthManager::new("http://localhost".into(), storage);
        assert!(matches!(manager.bootstrap().await, BootstrapResult::Anonymous));
    }

    #[tokio::test]
    async fn base_url_mismatch_cleans() {
        let storage = InMemoryStorage::new();
        let mut session = fresh_session("http://manager-base");
        session.base_url = "http://other-server".into();
        install_session(&storage, "http://manager-base", session);
        let storage_arc = storage.clone();
        let manager = AuthManager::new("http://manager-base".into(), storage_arc);

        match manager.bootstrap().await {
            BootstrapResult::AnonymousAfterCleanup { reason } => {
                assert_eq!(reason, BootstrapCleanupReason::BaseUrlMismatch);
            }
            other => panic!("expected cleanup, got {other:?}"),
        }
        assert!(!storage
            .snapshot()
            .keys()
            .any(|k| k == &session_storage_key("http://manager-base")));
    }

    #[tokio::test]
    async fn storage_corrupt_cleans() {
        let storage = InMemoryStorage::new();
        let key = session_storage_key("http://localhost");
        storage.set(&key, "not-valid-json{{{");
        let manager = AuthManager::new("http://localhost".into(), storage.clone());

        match manager.bootstrap().await {
            BootstrapResult::AnonymousAfterCleanup { reason } => {
                assert_eq!(reason, BootstrapCleanupReason::StorageCorrupt);
            }
            other => panic!("expected cleanup, got {other:?}"),
        }
        assert!(storage.get(&key).is_none());
    }

    #[tokio::test]
    async fn legacy_key_alone_is_purged() {
        let storage = InMemoryStorage::new();
        storage.set("agentsmesh-auth", r#"{"token":"old"}"#);
        let manager = AuthManager::new("http://localhost".into(), storage.clone());

        match manager.bootstrap().await {
            BootstrapResult::AnonymousAfterCleanup { reason } => {
                assert_eq!(reason, BootstrapCleanupReason::LegacyDataPurged);
            }
            other => panic!("expected cleanup, got {other:?}"),
        }
        assert!(storage.get("agentsmesh-auth").is_none());
    }

    #[tokio::test]
    async fn expired_token_refreshes_and_continues() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/RefreshToken"))
            .respond_with(proto_response(
                auth_proto::RefreshTokenResponse {
                    token: "new-access".into(),
                    refresh_token: "new-refresh".into(),
                    expires_in: 7200,
                }
                .encode_to_vec(),
            ))
            .mount(&server)
            .await;
        mount_me(&server).await;
        mount_orgs(&server).await;

        let storage = InMemoryStorage::new();
        let mut session = fresh_session(&server.uri());
        session.expires_at = now_unix_secs() - 100;
        install_session(&storage, &server.uri(), session);
        let manager = AuthManager::new(server.uri(), storage);

        match manager.bootstrap().await {
            BootstrapResult::Authenticated { .. } => {}
            other => panic!("expected Authenticated, got {other:?}"),
        }
    }

    #[tokio::test]
    async fn expired_token_refresh_failure_cleans() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/RefreshToken"))
            .respond_with(
                ResponseTemplate::new(401)
                    .set_body_json(serde_json::json!({"message": "refresh expired"})),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        let mut session = fresh_session(&server.uri());
        session.expires_at = now_unix_secs() - 100;
        install_session(&storage, &server.uri(), session);
        let manager = AuthManager::new(server.uri(), storage.clone());

        match manager.bootstrap().await {
            BootstrapResult::AnonymousAfterCleanup { reason } => {
                assert_eq!(reason, BootstrapCleanupReason::TokenExpiredAndRefreshFailed);
            }
            other => panic!("expected cleanup, got {other:?}"),
        }
        assert!(storage.get(&session_storage_key(&server.uri())).is_none());
    }

    #[tokio::test]
    async fn identity_401_cleans() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.user.v1.UserService/GetMe"))
            .respond_with(
                ResponseTemplate::new(401)
                    .set_body_json(serde_json::json!({"message": "invalid token"})),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        install_session(&storage, &server.uri(), fresh_session(&server.uri()));
        let manager = AuthManager::new(server.uri(), storage.clone());

        match manager.bootstrap().await {
            BootstrapResult::AnonymousAfterCleanup { reason } => {
                assert_eq!(reason, BootstrapCleanupReason::UnauthorizedFromIdentityCall);
            }
            other => panic!("expected cleanup, got {other:?}"),
        }
        assert!(storage.get(&session_storage_key(&server.uri())).is_none());
    }

    #[tokio::test]
    async fn identity_5xx_keeps_session_returns_anonymous() {
        let server = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/proto.user.v1.UserService/GetMe"))
            .respond_with(
                ResponseTemplate::new(503)
                    .set_body_json(serde_json::json!({"message": "service unavailable"})),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        install_session(&storage, &server.uri(), fresh_session(&server.uri()));
        let manager = AuthManager::new(server.uri(), storage.clone());

        match manager.bootstrap().await {
            BootstrapResult::Anonymous => {}
            other => panic!("expected transient anonymous, got {other:?}"),
        }
        assert!(storage.get(&session_storage_key(&server.uri())).is_some());
    }

    #[tokio::test]
    async fn organizations_failure_falls_back_to_no_current_org() {
        let server = MockServer::start().await;
        mount_me(&server).await;
        Mock::given(method("POST"))
            .and(path("/proto.org.v1.OrgService/ListMyOrgs"))
            .respond_with(
                ResponseTemplate::new(500)
                    .set_body_json(serde_json::json!({"message": "internal"})),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        install_session(&storage, &server.uri(), fresh_session(&server.uri()));
        let manager = AuthManager::new(server.uri(), storage);

        match manager.bootstrap().await {
            BootstrapResult::Authenticated { current_org, .. } => {
                assert!(current_org.is_none());
            }
            other => panic!("expected Authenticated with no org, got {other:?}"),
        }
    }

    #[tokio::test]
    async fn organizations_401_cleans_session() {
        // R3 regression guard: a 401 from ListMyOrgs indicates the token
        // was just revoked (after GetMe succeeded). It MUST run cleanup —
        // silently falling back to empty orgs would leave a stale session
        // on disk + dashboard with empty data, so next business request
        // 401's into a refresh loop.
        let server = MockServer::start().await;
        mount_me(&server).await;
        Mock::given(method("POST"))
            .and(path("/proto.org.v1.OrgService/ListMyOrgs"))
            .respond_with(
                ResponseTemplate::new(401)
                    .set_body_json(serde_json::json!({"message": "token revoked"})),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        install_session(&storage, &server.uri(), fresh_session(&server.uri()));
        let manager = AuthManager::new(server.uri(), storage.clone());

        match manager.bootstrap().await {
            BootstrapResult::AnonymousAfterCleanup { reason } => {
                assert_eq!(reason, BootstrapCleanupReason::UnauthorizedFromIdentityCall);
            }
            other => panic!("expected cleanup, got {other:?}"),
        }
        assert!(storage.get(&session_storage_key(&server.uri())).is_none());
    }

    #[tokio::test]
    async fn isolated_namespaces_for_different_base_urls() {
        let storage = InMemoryStorage::new();
        install_session(&storage, "http://server-a", {
            let mut s = fresh_session("http://server-a");
            s.access_token = "tok-a".into();
            s
        });
        install_session(&storage, "http://server-b", {
            let mut s = fresh_session("http://server-b");
            s.access_token = "tok-b".into();
            s
        });

        let dump = storage.snapshot();
        assert_eq!(dump.len(), 2);
        let key_a = session_storage_key("http://server-a");
        let key_b = session_storage_key("http://server-b");
        assert!(dump.contains_key(&key_a));
        assert!(dump.contains_key(&key_b));
        assert_ne!(key_a, key_b);
    }

    #[tokio::test]
    async fn persisted_session_does_not_contain_user_object() {
        let storage = InMemoryStorage::new();
        install_session(&storage, "http://localhost", fresh_session("http://localhost"));

        let key = session_storage_key("http://localhost");
        let raw = storage.get(&key).unwrap();
        assert!(!raw.contains("\"email\""), "session blob leaked email field");
        assert!(!raw.contains("\"username\""), "session blob leaked username field");
        assert!(!raw.contains("\"name\""), "session blob leaked name field");
        assert!(!raw.contains("\"avatar_url\""), "session blob leaked avatar_url field");
    }
}
