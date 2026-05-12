use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::proto_auth_v1 as auth_proto;

// Connect-RPC bindings for `proto.auth.v1` (conventions §2.5: binary wire,
// application/proto). Two services live in the same proto package:
//
//   * AuthService          — all RPCs are PUBLIC (no auth interceptor).
//                            Login / Register / Refresh / OAuth / Verify /
//                            Forgot- and ResetPassword. The user can't
//                            present a bearer token before they have one.
//   * AuthSessionService   — Logout. Authenticated — the bearer token IS
//                            the credential being revoked.
//
// `connect_call` silently omits the Authorization header when the auth
// store is empty (the public RPCs hit the server without it), so the same
// helper drives both services. The public service mount path on the
// backend skips the auth interceptor (`MountPublic` in
// `backend/internal/api/connect/auth/auth.go`).

impl ApiClient {
    // -------- AuthService (public — no token required) --------

    pub async fn auth_login_connect(
        &self,
        req: &auth_proto::LoginRequest,
    ) -> Result<auth_proto::LoginResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/Login", req).await
    }

    pub async fn auth_register_connect(
        &self,
        req: &auth_proto::RegisterRequest,
    ) -> Result<auth_proto::RegisterResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/Register", req).await
    }

    pub async fn auth_refresh_token_connect(
        &self,
        req: &auth_proto::RefreshTokenRequest,
    ) -> Result<auth_proto::RefreshTokenResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/RefreshToken", req).await
    }

    pub async fn auth_verify_email_connect(
        &self,
        req: &auth_proto::VerifyEmailRequest,
    ) -> Result<auth_proto::VerifyEmailResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/VerifyEmail", req).await
    }

    pub async fn auth_resend_verification_connect(
        &self,
        req: &auth_proto::ResendVerificationRequest,
    ) -> Result<auth_proto::ResendVerificationResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/ResendVerification", req).await
    }

    pub async fn auth_forgot_password_connect(
        &self,
        req: &auth_proto::ForgotPasswordRequest,
    ) -> Result<auth_proto::ForgotPasswordResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/ForgotPassword", req).await
    }

    pub async fn auth_reset_password_connect(
        &self,
        req: &auth_proto::ResetPasswordRequest,
    ) -> Result<auth_proto::ResetPasswordResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/ResetPassword", req).await
    }

    pub async fn auth_oauth_redirect_connect(
        &self,
        req: &auth_proto::OAuthRedirectRequest,
    ) -> Result<auth_proto::OAuthRedirectResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/OAuthRedirect", req).await
    }

    pub async fn auth_oauth_callback_connect(
        &self,
        req: &auth_proto::OAuthCallbackRequest,
    ) -> Result<auth_proto::OAuthCallbackResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthService/OAuthCallback", req).await
    }

    // -------- AuthSessionService (auth-required) --------

    pub async fn auth_logout_connect(
        &self,
        req: &auth_proto::LogoutRequest,
    ) -> Result<auth_proto::LogoutResponse, ApiError> {
        connect_call(self, "/proto.auth.v1.AuthSessionService/Logout", req).await
    }
}
