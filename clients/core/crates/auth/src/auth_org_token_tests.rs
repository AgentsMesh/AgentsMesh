#[cfg(test)]
mod auth_org_token_tests {
    use agentsmesh_api_client::AuthTokenStore;
    use agentsmesh_types::proto_auth_v1 as auth_proto;
    use agentsmesh_types::proto_org_v1 as org_proto;
    use prost::Message;
    use wiremock::matchers::{method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    use crate::manager::AuthManager;
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

    fn proto_response(bytes: Vec<u8>) -> ResponseTemplate {
        ResponseTemplate::new(200)
            .set_body_bytes(bytes)
            .insert_header("content-type", "application/proto")
    }

    fn orgs_resp(items: Vec<org_proto::Organization>) -> Vec<u8> {
        let total = items.len() as i64;
        org_proto::ListMyOrgsResponse {
            items,
            total,
            limit: 50,
            offset: 0,
        }
        .encode_to_vec()
    }

    fn mock_org(id: i64, name: &str, slug: &str) -> org_proto::Organization {
        org_proto::Organization {
            id,
            name: name.into(),
            slug: slug.into(),
            logo_url: None,
            // ListMyOrgs serializes plan/status; empty strings get
            // promoted to None in org_from_proto. Tests assert on slug
            // and name only, so the values themselves don't matter.
            subscription_plan: String::new(),
            subscription_status: String::new(),
            role: None,
            created_at: "2026-01-01T00:00:00Z".into(),
            updated_at: "2026-01-01T00:00:00Z".into(),
        }
    }

    fn mount_login(server: &MockServer) -> impl std::future::Future<Output = ()> + '_ {
        Mock::given(method("POST"))
            .and(path("/proto.auth.v1.AuthService/Login"))
            .respond_with(proto_response(login_resp_proto()))
            .mount(server)
    }

    fn mount_orgs<'a>(
        server: &'a MockServer,
        items: Vec<org_proto::Organization>,
    ) -> impl std::future::Future<Output = ()> + 'a {
        Mock::given(method("POST"))
            .and(path("/proto.org.v1.OrgService/ListMyOrgs"))
            .respond_with(proto_response(orgs_resp(items)))
            .mount(server)
    }

    #[tokio::test]
    async fn refresh_token_success() {
        let server = MockServer::start().await;
        mount_login(&server).await;
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

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        manager.login("dev@test.com", "pass").await.unwrap();
        let tokens = manager.refresh_token().await.unwrap();
        assert_eq!(tokens.token, "new-access");
        assert_eq!(tokens.refresh_token, "new-refresh");
        assert_eq!(manager.get_token(), Some("new-access".into()));
    }

    #[tokio::test]
    async fn refresh_without_auth() {
        let storage = InMemoryStorage::new();
        let manager = AuthManager::new("http://localhost".into(), storage);
        let err = manager.refresh_token().await.unwrap_err();
        assert!(matches!(err, crate::AuthError::NotAuthenticated));
    }

    #[tokio::test]
    async fn fetch_organizations_selects_first() {
        let server = MockServer::start().await;
        mount_login(&server).await;
        mount_orgs(
            &server,
            vec![
                mock_org(1, "Org A", "org-a"),
                mock_org(2, "Org B", "org-b"),
            ],
        )
        .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        manager.login("dev@test.com", "pass").await.unwrap();
        let orgs = manager.fetch_organizations().await.unwrap();
        assert_eq!(orgs.len(), 2);

        let current = manager.get_current_org().unwrap();
        assert_eq!(current.slug, "org-a");
        assert_eq!(manager.get_organizations().len(), 2);
        assert_eq!(manager.get_current_org_slug(), Some("org-a".into()));
    }

    #[tokio::test]
    async fn switch_org_success() {
        let server = MockServer::start().await;
        mount_login(&server).await;
        mount_orgs(
            &server,
            vec![
                mock_org(1, "Org A", "org-a"),
                mock_org(2, "Org B", "org-b"),
            ],
        )
        .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        manager.login("dev@test.com", "pass").await.unwrap();
        manager.fetch_organizations().await.unwrap();
        manager.switch_org("org-b").unwrap();
        assert_eq!(manager.get_current_org().unwrap().slug, "org-b");
        assert_eq!(manager.get_current_org_slug(), Some("org-b".into()));
    }

    #[tokio::test]
    async fn switch_org_not_found() {
        let server = MockServer::start().await;
        mount_login(&server).await;
        mount_orgs(&server, vec![mock_org(1, "Org A", "org-a")]).await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        manager.login("dev@test.com", "pass").await.unwrap();
        manager.fetch_organizations().await.unwrap();
        let err = manager.switch_org("nonexistent");
        assert!(err.is_err());
    }

    #[tokio::test]
    async fn get_current_org_slug_via_trait() {
        let server = MockServer::start().await;
        mount_login(&server).await;
        mount_orgs(&server, vec![mock_org(1, "Org A", "org-a")]).await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        assert!(manager.get_current_org_slug().is_none());
        manager.login("dev@test.com", "pass").await.unwrap();
        manager.fetch_organizations().await.unwrap();
        assert_eq!(manager.get_current_org_slug(), Some("org-a".into()));
    }

    #[tokio::test]
    async fn set_current_org_directly() {
        let server = MockServer::start().await;
        mount_login(&server).await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        manager.login("dev@test.com", "pass").await.unwrap();
        let org = agentsmesh_types::Organization {
            id: 99,
            name: "Direct Org".into(),
            slug: "direct-org".into(),
            role: None,
            logo_url: None,
            subscription_plan: None,
            subscription_status: None,
        };
        manager.set_current_org(Some(org));
        let current = manager.get_current_org().unwrap();
        assert_eq!(current.slug, "direct-org");
        assert_eq!(manager.get_current_org_slug(), Some("direct-org".into()));
    }

    #[tokio::test]
    async fn fetch_organizations_failure() {
        let server = MockServer::start().await;
        mount_login(&server).await;
        Mock::given(method("POST"))
            .and(path("/proto.org.v1.OrgService/ListMyOrgs"))
            .respond_with(
                ResponseTemplate::new(500)
                    .set_body_json(serde_json::json!({"message": "internal"})),
            )
            .mount(&server)
            .await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        manager.login("dev@test.com", "pass").await.unwrap();
        let err = manager.fetch_organizations().await.unwrap_err();
        assert!(matches!(err, crate::AuthError::Server { status: 500, .. }));
    }

    #[tokio::test]
    async fn fetch_organizations_not_authenticated() {
        let storage = InMemoryStorage::new();
        let manager = AuthManager::new("http://localhost".into(), storage);
        let err = manager.fetch_organizations().await.unwrap_err();
        assert!(matches!(err, crate::AuthError::NotAuthenticated));
    }

    #[tokio::test]
    async fn auth_token_store_trait() {
        let server = MockServer::start().await;
        mount_login(&server).await;

        let storage = InMemoryStorage::new();
        let manager = AuthManager::new(server.uri(), storage);

        assert!(manager.get_token().is_none());
        manager.login("dev@test.com", "pass").await.unwrap();
        assert_eq!(manager.get_token(), Some("access-tok".into()));
        assert_eq!(manager.get_refresh_token(), Some("refresh-tok".into()));

        manager.set_tokens("new-t".into(), "new-r".into(), Some(7200));
        assert_eq!(manager.get_token(), Some("new-t".into()));
        assert_eq!(manager.get_refresh_token(), Some("new-r".into()));

        manager.clear_tokens();
        assert!(manager.get_token().is_none());
        assert!(manager.get_refresh_token().is_none());
    }

    #[test]
    fn bearer_header_not_authenticated() {
        let storage = InMemoryStorage::new();
        let manager = AuthManager::new("http://localhost".into(), storage);
        let err = manager.bearer_header();
        assert!(err.is_err());
    }

    #[test]
    fn is_authenticated_states() {
        let storage = InMemoryStorage::new();
        let manager = AuthManager::new("http://localhost".into(), storage);
        assert!(!manager.is_authenticated());
        assert!(manager.current_user().is_none());
    }
}
