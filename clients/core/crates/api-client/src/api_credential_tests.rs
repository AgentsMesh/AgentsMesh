#[cfg(test)]
mod api_credential_tests {
    use std::sync::{Arc, Mutex};

    use serde_json::json;
    use wiremock::matchers::{method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    use crate::{ApiClient, AuthTokenStore};

    struct Tok(Mutex<Option<String>>);
    impl Tok {
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
    async fn list_user_agent_credentials_for_agent() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/users/agent-credentials/agents/claude"))
            .respond_with(ok(json!({"profiles":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.list_user_agent_credentials_for_agent("claude").await.unwrap();
    }

    #[tokio::test]
    async fn create_user_agent_credential() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/users/agent-credentials/agents/claude"))
            .respond_with(ok(json!({"id":1,"name":"c"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::CreateAgentCredentialProfileRequest {
            name: "c".into(), description: None, is_runner_host: None,
            credentials: None, is_default: None,
        };
        let _ = c.create_user_agent_credential("claude", &data).await.unwrap();
    }

    #[tokio::test]
    async fn get_user_agent_credential() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/users/agent-credentials/profiles/3"))
            .respond_with(ok(json!({"id":3,"name":"c3"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let r = c.get_user_agent_credential(3).await.unwrap();
        assert_eq!(r.id, 3);
    }

    #[tokio::test]
    async fn update_user_agent_credential() {
        let s = MockServer::start().await;
        Mock::given(method("PUT"))
            .and(path("/api/v1/users/agent-credentials/profiles/3"))
            .respond_with(ok(json!({"id":3,"name":"upd"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::UpdateAgentCredentialProfileRequest {
            name: Some("upd".into()), description: None, is_runner_host: None,
            credentials: None, is_default: None,
        };
        let _ = c.update_user_agent_credential(3, &data).await.unwrap();
    }

    #[tokio::test]
    async fn delete_user_agent_credential() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE"))
            .and(path("/api/v1/users/agent-credentials/profiles/3"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.delete_user_agent_credential(3).await.unwrap();
    }

    #[tokio::test]
    async fn create_user_git_credential() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/users/git-credentials"))
            .respond_with(ok(json!({"id":1,"name":"g"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::CreateGitCredentialRequest {
            name: "g".into(), credential_type: "pat".into(),
            repository_provider_id: None, pat: Some("ghp".into()),
            private_key: None, host_pattern: None,
        };
        let _ = c.create_user_git_credential(&data).await.unwrap();
    }

    #[tokio::test]
    async fn get_user_git_credential() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/git-credentials/2"))
            .respond_with(ok(json!({"id":2,"name":"g2"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.get_user_git_credential(2).await.unwrap();
    }

    #[tokio::test]
    async fn update_user_git_credential() {
        let s = MockServer::start().await;
        Mock::given(method("PUT")).and(path("/api/v1/users/git-credentials/2"))
            .respond_with(ok(json!({"id":2,"name":"upd"}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::UpdateGitCredentialRequest {
            name: Some("upd".into()), pat: None, private_key: None, host_pattern: None,
        };
        let _ = c.update_user_git_credential(2, &data).await.unwrap();
    }

    #[tokio::test]
    async fn delete_user_git_credential() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/users/git-credentials/2"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.delete_user_git_credential(2).await.unwrap();
    }

    #[tokio::test]
    async fn set_default_git_credential() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/users/git-credentials/default"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::SetDefaultGitCredentialRequest { credential_id: Some(2) };
        let _ = c.set_default_git_credential(&data).await.unwrap();
    }

    #[tokio::test]
    async fn clear_default_git_credential() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/users/git-credentials/default"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.clear_default_git_credential().await.unwrap();
    }

    #[tokio::test]
    async fn create_user_repository_provider() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/users/repository-providers"))
            .respond_with(ok(json!({"id":1,"provider_type":"gitlab","name":"GL"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::CreateRepositoryProviderRequest {
            provider_type: "gitlab".into(), name: "GL".into(),
            base_url: None, client_id: None, client_secret: None, bot_token: None,
        };
        let _ = c.create_user_repository_provider(&data).await.unwrap();
    }

    #[tokio::test]
    async fn get_user_repository_provider() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/users/repository-providers/2"))
            .respond_with(ok(json!({"id":2,"provider_type":"github","name":"GH"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let r = c.get_user_repository_provider(2).await.unwrap();
        assert_eq!(r.name, "GH");
    }

    #[tokio::test]
    async fn update_user_repository_provider() {
        let s = MockServer::start().await;
        Mock::given(method("PUT")).and(path("/api/v1/users/repository-providers/2"))
            .respond_with(ok(json!({"id":2,"provider_type":"github","name":"Upd"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::UpdateRepositoryProviderRequest {
            name: Some("Upd".into()), base_url: None,
            client_id: None, client_secret: None, bot_token: None,
            is_active: None,
        };
        let _ = c.update_user_repository_provider(2, &data).await.unwrap();
    }

    #[tokio::test]
    async fn delete_user_repository_provider() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/users/repository-providers/2"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.delete_user_repository_provider(2).await.unwrap();
    }

    #[tokio::test]
    async fn set_default_repository_provider() {
        let s = MockServer::start().await;
        Mock::given(method("POST"))
            .and(path("/api/v1/users/repository-providers/2/default"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.set_default_repository_provider(2).await.unwrap();
    }
}
