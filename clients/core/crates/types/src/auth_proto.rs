// Hand-maintained `prost::Message` mirrors of
// `proto/auth/v1/auth.proto`. Tag numbers match the .proto byte-for-byte;
// `tools/validate_prost_tags` runs at build time to catch drift (watch
// list §8). NO `Serialize`/`Deserialize` derives — binary wire only
// (conventions §2.5, §3).
//
// USER-SCOPED service (conventions §3.5 exception #1): no `org_slug`
// field. Most RPCs are PUBLIC — Login/Register/Refresh/OAuth/Verify don't
// require a bearer token. Logout is the only authenticated RPC.

#[derive(Clone, PartialEq, prost::Message)]
pub struct User {
    #[prost(int64, tag = "1")]
    pub id: i64,
    #[prost(string, tag = "2")]
    pub email: String,
    #[prost(string, tag = "3")]
    pub username: String,
    #[prost(string, optional, tag = "4")]
    pub name: Option<String>,
    #[prost(string, optional, tag = "5")]
    pub avatar_url: Option<String>,
    #[prost(bool, optional, tag = "6")]
    pub is_email_verified: Option<bool>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LoginRequest {
    #[prost(string, tag = "1")]
    pub email: String,
    #[prost(string, tag = "2")]
    pub password: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LoginResponse {
    #[prost(string, tag = "1")]
    pub token: String,
    #[prost(string, tag = "2")]
    pub refresh_token: String,
    #[prost(int64, tag = "3")]
    pub expires_in: i64,
    #[prost(message, optional, tag = "4")]
    pub user: Option<User>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RegisterRequest {
    #[prost(string, tag = "1")]
    pub email: String,
    #[prost(string, tag = "2")]
    pub username: String,
    #[prost(string, tag = "3")]
    pub password: String,
    #[prost(string, optional, tag = "4")]
    pub name: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RegisterResponse {
    #[prost(string, tag = "1")]
    pub token: String,
    #[prost(string, tag = "2")]
    pub refresh_token: String,
    #[prost(int64, tag = "3")]
    pub expires_in: i64,
    #[prost(message, optional, tag = "4")]
    pub user: Option<User>,
    #[prost(string, optional, tag = "5")]
    pub message: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RefreshTokenRequest {
    #[prost(string, tag = "1")]
    pub refresh_token: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct RefreshTokenResponse {
    #[prost(string, tag = "1")]
    pub token: String,
    #[prost(string, tag = "2")]
    pub refresh_token: String,
    #[prost(int64, tag = "3")]
    pub expires_in: i64,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LogoutRequest {}

#[derive(Clone, PartialEq, prost::Message)]
pub struct LogoutResponse {
    #[prost(string, tag = "1")]
    pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct VerifyEmailRequest {
    #[prost(string, tag = "1")]
    pub token: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct VerifyEmailResponse {
    #[prost(string, tag = "1")]
    pub token: String,
    #[prost(string, tag = "2")]
    pub refresh_token: String,
    #[prost(int64, tag = "3")]
    pub expires_in: i64,
    #[prost(message, optional, tag = "4")]
    pub user: Option<User>,
    #[prost(string, tag = "5")]
    pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ResendVerificationRequest {
    #[prost(string, tag = "1")]
    pub email: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ResendVerificationResponse {
    #[prost(string, tag = "1")]
    pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ForgotPasswordRequest {
    #[prost(string, tag = "1")]
    pub email: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ForgotPasswordResponse {
    #[prost(string, tag = "1")]
    pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ResetPasswordRequest {
    #[prost(string, tag = "1")]
    pub token: String,
    #[prost(string, tag = "2")]
    pub new_password: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct ResetPasswordResponse {
    #[prost(string, tag = "1")]
    pub message: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct OAuthRedirectRequest {
    #[prost(string, tag = "1")]
    pub provider: String,
    #[prost(string, optional, tag = "2")]
    pub redirect: Option<String>,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct OAuthRedirectResponse {
    #[prost(string, tag = "1")]
    pub auth_url: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct OAuthCallbackRequest {
    #[prost(string, tag = "1")]
    pub provider: String,
    #[prost(string, tag = "2")]
    pub code: String,
    #[prost(string, tag = "3")]
    pub state: String,
}

#[derive(Clone, PartialEq, prost::Message)]
pub struct OAuthCallbackResponse {
    #[prost(string, tag = "1")]
    pub token: String,
    #[prost(string, tag = "2")]
    pub refresh_token: String,
    #[prost(int64, tag = "3")]
    pub expires_in: i64,
    #[prost(message, optional, tag = "4")]
    pub user: Option<User>,
    #[prost(string, tag = "5")]
    pub redirect: String,
}

#[cfg(test)]
mod tests {
    use super::*;
    use prost::Message;

    fn sample_user() -> User {
        User {
            id: 42,
            email: "dev@example.com".into(),
            username: "dev".into(),
            name: Some("Dev User".into()),
            avatar_url: Some("https://cdn.example.com/avatar.png".into()),
            is_email_verified: Some(true),
        }
    }

    #[test]
    fn user_round_trip_preserves_every_field() {
        let original = sample_user();
        let bytes = original.encode_to_vec();
        let decoded = User::decode(&*bytes).unwrap();
        assert_eq!(original, decoded,
            "tag swap or transcription mistake would surface as field-value swap here");
    }

    #[test]
    fn login_response_round_trip() {
        let original = LoginResponse {
            token: "access-jwt".into(),
            refresh_token: "refresh-jwt".into(),
            expires_in: 3600,
            user: Some(sample_user()),
        };
        let bytes = original.encode_to_vec();
        let decoded = LoginResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.token, "access-jwt");
        assert_eq!(decoded.refresh_token, "refresh-jwt");
        assert_eq!(decoded.expires_in, 3600);
        assert!(decoded.user.is_some());
    }

    #[test]
    fn register_response_round_trip_with_message() {
        let original = RegisterResponse {
            token: "t".into(),
            refresh_token: "r".into(),
            expires_in: 3600,
            user: Some(sample_user()),
            message: Some("Registration successful. Please check your email.".into()),
        };
        let bytes = original.encode_to_vec();
        let decoded = RegisterResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.message, Some("Registration successful. Please check your email.".into()));
    }

    #[test]
    fn register_response_round_trip_without_message() {
        // REST handler omits `message` when email service is degraded.
        // Proto encodes `Option<String>::None` as absent — must decode back.
        let original = RegisterResponse {
            token: "t".into(),
            refresh_token: "r".into(),
            expires_in: 3600,
            user: Some(sample_user()),
            message: None,
        };
        let bytes = original.encode_to_vec();
        let decoded = RegisterResponse::decode(&*bytes).unwrap();
        assert_eq!(original, decoded);
        assert_eq!(decoded.message, None);
    }

    #[test]
    fn refresh_token_round_trip() {
        let req = RefreshTokenRequest { refresh_token: "old-refresh".into() };
        let req_bytes = req.encode_to_vec();
        assert_eq!(req, RefreshTokenRequest::decode(&*req_bytes).unwrap());

        let resp = RefreshTokenResponse {
            token: "new-access".into(),
            refresh_token: "new-refresh".into(),
            expires_in: 3600,
        };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, RefreshTokenResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn logout_round_trip() {
        let req = LogoutRequest {};
        let req_bytes = req.encode_to_vec();
        assert!(req_bytes.is_empty(), "empty message encodes to zero bytes");
        assert_eq!(req, LogoutRequest::decode(&*req_bytes).unwrap());

        let resp = LogoutResponse { message: "Logged out successfully".into() };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, LogoutResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn verify_email_round_trip() {
        let req = VerifyEmailRequest { token: "verify-token".into() };
        let req_bytes = req.encode_to_vec();
        assert_eq!(req, VerifyEmailRequest::decode(&*req_bytes).unwrap());

        let resp = VerifyEmailResponse {
            token: "t".into(),
            refresh_token: "r".into(),
            expires_in: 3600,
            user: Some(sample_user()),
            message: "Email verified successfully".into(),
        };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, VerifyEmailResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn resend_verification_round_trip() {
        let req = ResendVerificationRequest { email: "dev@example.com".into() };
        let req_bytes = req.encode_to_vec();
        assert_eq!(req, ResendVerificationRequest::decode(&*req_bytes).unwrap());

        let resp = ResendVerificationResponse { message: "Verification email sent".into() };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, ResendVerificationResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn forgot_password_round_trip() {
        let req = ForgotPasswordRequest { email: "dev@example.com".into() };
        let req_bytes = req.encode_to_vec();
        assert_eq!(req, ForgotPasswordRequest::decode(&*req_bytes).unwrap());

        let resp = ForgotPasswordResponse { message: "If the email exists, a password reset link will be sent".into() };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, ForgotPasswordResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn reset_password_round_trip() {
        let req = ResetPasswordRequest {
            token: "reset-token".into(),
            new_password: "newP@ssw0rd".into(),
        };
        let req_bytes = req.encode_to_vec();
        assert_eq!(req, ResetPasswordRequest::decode(&*req_bytes).unwrap());

        let resp = ResetPasswordResponse { message: "Password reset successfully".into() };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, ResetPasswordResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn oauth_redirect_round_trip() {
        let req = OAuthRedirectRequest {
            provider: "github".into(),
            redirect: Some("/dashboard".into()),
        };
        let req_bytes = req.encode_to_vec();
        assert_eq!(req, OAuthRedirectRequest::decode(&*req_bytes).unwrap());

        let resp = OAuthRedirectResponse {
            auth_url: "https://github.com/login/oauth/authorize?client_id=x&state=y".into(),
        };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, OAuthRedirectResponse::decode(&*resp_bytes).unwrap());
    }

    #[test]
    fn oauth_callback_round_trip() {
        let req = OAuthCallbackRequest {
            provider: "github".into(),
            code: "oauth-code".into(),
            state: "oauth-state".into(),
        };
        let req_bytes = req.encode_to_vec();
        assert_eq!(req, OAuthCallbackRequest::decode(&*req_bytes).unwrap());

        let resp = OAuthCallbackResponse {
            token: "t".into(),
            refresh_token: "r".into(),
            expires_in: 3600,
            user: Some(sample_user()),
            redirect: "/dashboard".into(),
        };
        let resp_bytes = resp.encode_to_vec();
        assert_eq!(resp, OAuthCallbackResponse::decode(&*resp_bytes).unwrap());
    }

    // Pinned by conventions §3: prost's `Option<T>` distinguishes "absent"
    // from "explicit empty value". For Register.name (optional), a request
    // that omits the field must decode back to None, while explicit empty
    // string must round-trip to Some("").
    #[test]
    fn optional_name_distinguishable_from_empty() {
        let absent = RegisterRequest {
            email: "a@b.com".into(),
            username: "u".into(),
            password: "p".into(),
            name: None,
        };
        let explicit_empty = RegisterRequest {
            email: "a@b.com".into(),
            username: "u".into(),
            password: "p".into(),
            name: Some(String::new()),
        };
        assert_ne!(absent.encode_to_vec(), explicit_empty.encode_to_vec(),
            "absent vs explicit empty string must encode differently");

        let r1 = RegisterRequest::decode(&*absent.encode_to_vec()).unwrap();
        let r2 = RegisterRequest::decode(&*explicit_empty.encode_to_vec()).unwrap();
        assert_eq!(r1.name, None);
        assert_eq!(r2.name, Some(String::new()));
    }
}
