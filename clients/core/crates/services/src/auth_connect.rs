use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_auth_v1 as auth_proto;
use prost::Message;

// AuthConnectService — thin orchestrator over the api-client Connect
// bindings for `proto.auth.v1`. Mirrors SSOService (services/src/sso.rs):
// each method accepts prost-encoded request bytes and returns prost-encoded
// response bytes (conventions §2.5 — binary in, binary out).
//
// USER-SCOPED + mostly PUBLIC: every AuthService RPC except Logout works
// without a bearer token (login/register obtain the token; verify/reset
// flow off email-delivered tokens, not session tokens). The api-client's
// `connect_call` helper silently omits Authorization when the auth store
// is empty, so the same surface drives the public AuthService and the
// authenticated AuthSessionService without special-casing.
//
// AuthManager's existing login/refresh/logout REST surface stays mounted
// in parallel — the Rust stateful auth flow is owned by the auth crate,
// not by this Connect bridge. This service handles the public
// register/verify/forgot/reset wire migration so the corresponding TS
// authApi entries can flip from JSON-over-REST to binary-over-Connect.
pub struct AuthConnectService {
    client: Arc<ApiClient>,
}

impl AuthConnectService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn login_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::LoginRequest::decode(request_bytes)
            .map_err(|e| format!("decode login request: {e}"))?;
        tracing::info!(target: "auth", "login");
        let resp = self.client.auth_login_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn register_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::RegisterRequest::decode(request_bytes)
            .map_err(|e| format!("decode register request: {e}"))?;
        tracing::info!(target: "auth", "register");
        let resp = self.client.auth_register_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn refresh_token_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::RefreshTokenRequest::decode(request_bytes)
            .map_err(|e| format!("decode refresh_token request: {e}"))?;
        tracing::debug!(target: "auth", "refresh token");
        let resp = self
            .client
            .auth_refresh_token_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn verify_email_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::VerifyEmailRequest::decode(request_bytes)
            .map_err(|e| format!("decode verify_email request: {e}"))?;
        tracing::info!(target: "auth", "verify email");
        let resp = self
            .client
            .auth_verify_email_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn resend_verification_connect(
        &self,
        request_bytes: &[u8],
    ) -> Result<Vec<u8>, String> {
        let req = auth_proto::ResendVerificationRequest::decode(request_bytes)
            .map_err(|e| format!("decode resend_verification request: {e}"))?;
        tracing::info!(target: "auth", "resend verification");
        let resp = self
            .client
            .auth_resend_verification_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn forgot_password_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::ForgotPasswordRequest::decode(request_bytes)
            .map_err(|e| format!("decode forgot_password request: {e}"))?;
        tracing::info!(target: "auth", "forgot password");
        let resp = self
            .client
            .auth_forgot_password_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn reset_password_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::ResetPasswordRequest::decode(request_bytes)
            .map_err(|e| format!("decode reset_password request: {e}"))?;
        tracing::info!(target: "auth", "reset password");
        let resp = self
            .client
            .auth_reset_password_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn oauth_redirect_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::OAuthRedirectRequest::decode(request_bytes)
            .map_err(|e| format!("decode oauth_redirect request: {e}"))?;
        tracing::info!(target: "auth", provider = %req.provider, "oauth redirect");
        let resp = self
            .client
            .auth_oauth_redirect_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn oauth_callback_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::OAuthCallbackRequest::decode(request_bytes)
            .map_err(|e| format!("decode oauth_callback request: {e}"))?;
        tracing::info!(target: "auth", provider = %req.provider, "oauth callback");
        let resp = self
            .client
            .auth_oauth_callback_connect(&req)
            .await
            .map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn logout_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = auth_proto::LogoutRequest::decode(request_bytes)
            .map_err(|e| format!("decode logout request: {e}"))?;
        tracing::info!(target: "auth", "logout");
        let resp = self.client.auth_logout_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
