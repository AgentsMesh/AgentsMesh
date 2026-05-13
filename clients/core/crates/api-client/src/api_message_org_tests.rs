#[cfg(test)]
mod api_message_org_tests {
    use std::sync::{Arc, Mutex};

    use serde_json::json;
    use wiremock::matchers::{body_json, method, path, query_param};
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

    fn user_json() -> serde_json::Value {
        json!({"id":1,"email":"u@t.com","username":"u"})
    }

    #[tokio::test]
    async fn send_mesh_message() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/messages"))
            .respond_with(ok(json!({"id":1})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::SendDirectMessageRequest {
            receiver_pod: "pod-1".into(), message_type: None,
            content: "hi".into(), correlation_id: None, reply_to_id: None,
        };
        let _ = c.send_mesh_message(&data, None).await.unwrap();
    }

    #[tokio::test]
    async fn send_mesh_message_with_pod_key_header() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/messages"))
            .respond_with(ok(json!({"id":1})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::SendDirectMessageRequest {
            receiver_pod: "pod-1".into(), message_type: Some("text".into()),
            content: "hi".into(), correlation_id: Some("c".into()), reply_to_id: Some(5),
        };
        let _ = c.send_mesh_message(&data, Some("my-key")).await.unwrap();
        let reqs = s.received_requests().await.unwrap();
        assert_eq!(reqs[0].headers.get("X-Pod-Key").unwrap().to_str().unwrap(), "my-key");
    }

    #[tokio::test]
    async fn get_mesh_message_by_id() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/messages/42"))
            .respond_with(ok(json!({"id":42}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let r = c.get_mesh_message(42).await.unwrap();
        assert_eq!(r.id, 42);
    }

    #[tokio::test]
    async fn mark_mesh_messages_read() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/messages/mark-read"))
            .and(body_json(json!({"message_ids":[1,2]})))
            .respond_with(ok(json!({"marked_count":2}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::MarkMessagesReadRequest { message_ids: vec![1, 2] };
        let _ = c.mark_mesh_messages_read(&data).await.unwrap();
    }

    #[tokio::test]
    async fn mark_all_mesh_messages_read() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/messages/mark-all-read"))
            .respond_with(ok(json!({"marked_count":0}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.mark_all_mesh_messages_read().await.unwrap();
    }

    #[tokio::test]
    async fn get_mesh_conversation() {
        let s = MockServer::start().await;
        Mock::given(method("GET"))
            .and(path("/api/v1/orgs/acme/messages/conversation/c1"))
            .and(query_param("limit", "5"))
            .respond_with(ok(json!({"messages":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.get_mesh_conversation("c1", Some(5)).await.unwrap();
    }

    #[tokio::test]
    async fn get_mesh_sent_messages() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/messages/sent"))
            .and(query_param("limit", "10")).and(query_param("offset", "5"))
            .respond_with(ok(json!({"messages":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.get_mesh_sent_messages(Some(10), Some(5)).await.unwrap();
    }

    #[tokio::test]
    async fn get_mesh_dead_letters() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/messages/dlq"))
            .respond_with(ok(json!({"entries":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.get_mesh_dead_letters(None, None).await.unwrap();
    }

    #[tokio::test]
    async fn replay_mesh_dead_letter() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/messages/dlq/7/replay"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let _ = c.replay_mesh_dead_letter(7).await.unwrap();
    }

    #[tokio::test]
    async fn set_notification_preference() {
        let s = MockServer::start().await;
        Mock::given(method("PUT")).and(path("/api/v1/orgs/acme/notifications/preferences"))
            .respond_with(ok(json!({"source":"pod","is_muted":true})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::org("acme"));
        let data = agentsmesh_types::SetNotificationPreferenceRequest {
            source: "pod".into(), entity_id: None, is_muted: Some(true), channels: None,
        };
        let _ = c.set_notification_preference(&data).await.unwrap();
    }

    #[tokio::test]
    async fn create_organization() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs"))
            .and(body_json(json!({"name":"Acme","slug":"acme"})))
            .respond_with(ok(json!({"id":1,"slug":"acme","name":"Acme"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::CreateOrganizationRequest {
            name: "Acme".into(), slug: "acme".into(),
        };
        let r = c.create_organization(&data).await.unwrap();
        assert_eq!(r.slug, "acme");
    }

    #[tokio::test]
    async fn update_organization() {
        let s = MockServer::start().await;
        Mock::given(method("PUT")).and(path("/api/v1/orgs/acme"))
            .respond_with(ok(json!({"id":1,"slug":"acme","name":"Acme Inc"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::UpdateOrganizationRequest {
            name: Some("Acme Inc".into()), logo_url: None,
        };
        let r = c.update_organization("acme", &data).await.unwrap();
        assert_eq!(r.name, "Acme Inc");
    }

    #[tokio::test]
    async fn delete_organization() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/orgs/acme"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.delete_organization("acme").await.unwrap();
    }

    #[tokio::test]
    async fn list_org_members() {
        let s = MockServer::start().await;
        Mock::given(method("GET")).and(path("/api/v1/orgs/acme/members"))
            .respond_with(ok(json!({"members":[]}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.list_org_members("acme").await.unwrap();
    }

    #[tokio::test]
    async fn invite_org_member() {
        let s = MockServer::start().await;
        Mock::given(method("POST")).and(path("/api/v1/orgs/acme/members"))
            .respond_with(ok(json!({"user": user_json(), "role":"member"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::InviteMemberRequest {
            email: "n@t.com".into(), role: "member".into(),
        };
        let _ = c.invite_org_member("acme", &data).await.unwrap();
    }

    #[tokio::test]
    async fn remove_org_member() {
        let s = MockServer::start().await;
        Mock::given(method("DELETE")).and(path("/api/v1/orgs/acme/members/5"))
            .respond_with(ok(json!({}))).expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let _ = c.remove_org_member("acme", 5).await.unwrap();
    }

    #[tokio::test]
    async fn update_org_member_role() {
        let s = MockServer::start().await;
        Mock::given(method("PUT")).and(path("/api/v1/orgs/acme/members/5"))
            .respond_with(ok(json!({"user": user_json(), "role":"admin"})))
            .expect(1).mount(&s).await;
        let c = ApiClient::new(s.uri(), Tok::none());
        let data = agentsmesh_types::UpdateMemberRoleRequest { role: "admin".into() };
        let _ = c.update_org_member_role("acme", 5, &data).await.unwrap();
    }
}
