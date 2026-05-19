use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: i64,
    pub email: String,
    pub username: String,
    pub name: Option<String>,
    pub avatar_url: Option<String>,
    #[serde(default)]
    pub is_email_verified: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserIdentity {
    pub id: i64,
    pub provider: String,
    pub provider_user_id: Option<String>,
    pub provider_username: Option<String>,
    pub created_at: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Organization {
    pub id: i64,
    pub name: String,
    pub slug: String,
    pub role: Option<String>,
    pub logo_url: Option<String>,
    pub subscription_plan: Option<String>,
    pub subscription_status: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthSession {
    pub token: String,
    pub refresh_token: String,
    pub user: User,
    pub expires_in: Option<i64>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthTokens {
    pub token: String,
    pub refresh_token: String,
    pub expires_in: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterRequest {
    pub name: String,
    pub email: String,
    pub username: String,
    pub password: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SSOConfig {
    pub domain: String,
    pub protocol: String,
    pub name: Option<String>,
    pub enforce_sso: Option<bool>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json;

    #[test]
    fn user_roundtrip() {
        let user = User {
            id: 1,
            email: "dev@test.com".into(),
            username: "dev".into(),
            name: Some("Dev User".into()),
            avatar_url: None,
            is_email_verified: Some(true),
        };
        let json = serde_json::to_string(&user).unwrap();
        let decoded: User = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.id, 1);
        assert_eq!(decoded.email, "dev@test.com");
        assert_eq!(decoded.name, Some("Dev User".into()));
        assert!(decoded.avatar_url.is_none());
    }

    #[test]
    fn user_snake_case_json() {
        let json = r#"{"id":1,"email":"a@b.com","username":"u","name":null,"avatar_url":null}"#;
        let user: User = serde_json::from_str(json).unwrap();
        assert_eq!(user.id, 1);
        assert_eq!(user.username, "u");
    }

    #[test]
    fn user_optional_fields_missing() {
        let json = r#"{"id":1,"email":"a@b.com","username":"u"}"#;
        let user: User = serde_json::from_str(json).unwrap();
        assert!(user.name.is_none());
        assert!(user.avatar_url.is_none());
    }

    #[test]
    fn organization_roundtrip() {
        let org = Organization {
            id: 10,
            name: "Acme".into(),
            slug: "acme".into(),
            role: Some("owner".into()),
            logo_url: None,
            subscription_plan: Some("pro".into()),
            subscription_status: Some("active".into()),
        };
        let json = serde_json::to_string(&org).unwrap();
        let decoded: Organization = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.slug, "acme");
        assert_eq!(decoded.role, Some("owner".into()));
    }

    #[test]
    fn organization_optional_fields_missing() {
        let json = r#"{"id":1,"name":"O","slug":"o"}"#;
        let org: Organization = serde_json::from_str(json).unwrap();
        assert!(org.role.is_none());
        assert!(org.logo_url.is_none());
        assert!(org.subscription_plan.is_none());
        assert!(org.subscription_status.is_none());
    }

    #[test]
    fn auth_session_roundtrip() {
        let session = AuthSession {
            token: "tok".into(),
            refresh_token: "ref".into(),
            user: User {
                id: 1,
                email: "a@b.com".into(),
                username: "u".into(),
                name: None,
                avatar_url: None,
                is_email_verified: None,
            },
            expires_in: Some(3600),
            message: None,
        };
        let json = serde_json::to_string(&session).unwrap();
        let decoded: AuthSession = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.token, "tok");
        assert_eq!(decoded.user.id, 1);
        assert_eq!(decoded.expires_in, Some(3600));
    }

    #[test]
    fn auth_session_optional_expires() {
        let json = r#"{
            "token":"t","refresh_token":"r",
            "user":{"id":1,"email":"e","username":"u"}
        }"#;
        let session: AuthSession = serde_json::from_str(json).unwrap();
        assert!(session.expires_in.is_none());
    }

    #[test]
    fn auth_tokens_roundtrip() {
        let tokens = AuthTokens {
            token: "new-tok".into(),
            refresh_token: "new-ref".into(),
            expires_in: None,
        };
        let json = serde_json::to_string(&tokens).unwrap();
        let decoded: AuthTokens = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.token, "new-tok");
        assert!(decoded.expires_in.is_none());
    }

    #[test]
    fn login_request_serialization() {
        let req = LoginRequest {
            email: "dev@test.com".into(),
            password: "pass".into(),
        };
        let json = serde_json::to_string(&req).unwrap();
        assert!(json.contains("\"email\":\"dev@test.com\""));
        assert!(json.contains("\"password\":\"pass\""));
    }

    #[test]
    fn register_request_serialization() {
        let req = RegisterRequest {
            name: "Dev".into(),
            email: "dev@test.com".into(),
            username: "dev".into(),
            password: "pass123".into(),
        };
        let json = serde_json::to_string(&req).unwrap();
        let decoded: RegisterRequest = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.username, "dev");
    }

    #[test]
    fn sso_config_roundtrip() {
        let cfg = SSOConfig {
            domain: "example.com".into(),
            protocol: "saml".into(),
            name: Some("Corporate SSO".into()),
            enforce_sso: Some(true),
        };
        let json = serde_json::to_string(&cfg).unwrap();
        let decoded: SSOConfig = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.domain, "example.com");
        assert_eq!(decoded.enforce_sso, Some(true));
    }

    #[test]
    fn sso_config_optional_fields_missing() {
        let json = r#"{"domain":"d","protocol":"p"}"#;
        let cfg: SSOConfig = serde_json::from_str(json).unwrap();
        assert!(cfg.name.is_none());
        assert!(cfg.enforce_sso.is_none());
    }
}
