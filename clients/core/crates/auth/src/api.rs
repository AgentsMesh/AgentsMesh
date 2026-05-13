use agentsmesh_types::proto_auth_v1 as auth_proto;
use agentsmesh_types::proto_user_v1 as user_proto;
use agentsmesh_types::{AuthSession, AuthTokens, RegisterRequest};

use crate::connect::connect_call;
use crate::error::AuthError;
use crate::manager::{now_unix_secs, AuthManager};

// Maps the prost `auth_proto::User` shape (returned by AuthService) to
// the serde `agentsmesh_types::User` AuthState already persists. Done
// once here so callers downstream stay shape-agnostic — anything that
// reads the user out of AuthState (web UI, FFI, bootstrap) sees the
// same serde DTO the legacy REST path produced.
fn user_from_auth_proto(u: auth_proto::User) -> agentsmesh_types::User {
    agentsmesh_types::User {
        id: u.id,
        email: u.email,
        username: u.username,
        name: u.name,
        avatar_url: u.avatar_url,
        is_email_verified: u.is_email_verified,
    }
}

// `proto.user.v1.User` (GetMe response) carries more fields than the
// serde User AuthState persists; we down-project to keep AuthState's
// surface stable. The auth-proto User and user-proto User overlap on
// the fields AuthManager actually needs (id / email / username / name
// / avatar_url / is_email_verified) so this mapping is loss-tolerant.
fn user_from_user_proto(u: user_proto::User) -> agentsmesh_types::User {
    agentsmesh_types::User {
        id: u.id,
        email: u.email,
        username: u.username,
        name: u.name,
        avatar_url: u.avatar_url,
        is_email_verified: Some(u.is_email_verified),
    }
}

fn session_from_login(resp: auth_proto::LoginResponse) -> Result<AuthSession, AuthError> {
    let user = resp
        .user
        .ok_or_else(|| AuthError::InvalidResponse("login response missing user".into()))?;
    Ok(AuthSession {
        token: resp.token,
        refresh_token: resp.refresh_token,
        user: user_from_auth_proto(user),
        expires_in: Some(resp.expires_in),
        message: None,
    })
}

fn session_from_register(resp: auth_proto::RegisterResponse) -> Result<AuthSession, AuthError> {
    let user = resp
        .user
        .ok_or_else(|| AuthError::InvalidResponse("register response missing user".into()))?;
    Ok(AuthSession {
        token: resp.token,
        refresh_token: resp.refresh_token,
        user: user_from_auth_proto(user),
        expires_in: Some(resp.expires_in),
        message: resp.message,
    })
}

fn session_from_verify(resp: auth_proto::VerifyEmailResponse) -> Result<AuthSession, AuthError> {
    let user = resp
        .user
        .ok_or_else(|| AuthError::InvalidResponse("verify response missing user".into()))?;
    Ok(AuthSession {
        token: resp.token,
        refresh_token: resp.refresh_token,
        user: user_from_auth_proto(user),
        expires_in: Some(resp.expires_in),
        message: Some(resp.message),
    })
}

impl AuthManager {
    pub async fn login(&self, email: &str, password: &str) -> Result<AuthSession, AuthError> {
        let req = auth_proto::LoginRequest {
            email: email.to_string(),
            password: password.to_string(),
        };
        let resp: auth_proto::LoginResponse =
            connect_call(self, "/proto.auth.v1.AuthService/Login", &req, None).await?;
        let session = session_from_login(resp)?;
        self.apply_session(&session);
        Ok(session)
    }

    pub async fn register(&self, data: &RegisterRequest) -> Result<AuthSession, AuthError> {
        let req = auth_proto::RegisterRequest {
            email: data.email.clone(),
            username: data.username.clone(),
            password: data.password.clone(),
            name: Some(data.name.clone()),
        };
        let resp: auth_proto::RegisterResponse =
            connect_call(self, "/proto.auth.v1.AuthService/Register", &req, None).await?;
        let session = session_from_register(resp)?;
        self.apply_session(&session);
        Ok(session)
    }

    pub async fn logout(&self) -> Result<(), AuthError> {
        let auth = self.bearer_header()?;

        // Best-effort server logout. Network errors and non-2xx responses
        // (token already revoked, server 5xx) MUST NOT leave the caller
        // logged in — plan I3 forbids the in-memory-user-but-no-token
        // middle state. Local cleanup runs unconditionally; server-side
        // failure is logged but swallowed so the user lands on /login.
        let server_result = connect_call::<_, auth_proto::LogoutResponse>(
            self,
            "/proto.auth.v1.AuthSessionService/Logout",
            &auth_proto::LogoutRequest {},
            Some(&auth),
        )
        .await;

        self.reset_local();

        match server_result {
            Ok(_) => {}
            Err(AuthError::Server { status, .. }) => {
                tracing::warn!(
                    "auth logout: server returned {}; local state cleared anyway",
                    status
                );
            }
            Err(e) => {
                tracing::warn!(
                    "auth logout: server unreachable ({e}); local state cleared anyway"
                );
            }
        }
        Ok(())
    }

    pub async fn refresh_token(&self) -> Result<AuthTokens, AuthError> {
        // Snapshot the refresh token we intend to spend BEFORE blocking on
        // the lock. If a concurrent refresh on the same manager already
        // rotated tokens while we waited, our snapshot will differ from
        // the post-lock state — short-circuit and return the new tokens.
        let pre_lock_refresh = self
            .read_state()
            .refresh_token()
            .map(String::from)
            .ok_or(AuthError::NotAuthenticated)?;

        let _guard = self.refresh_lock.lock().await;

        let current_refresh = self.read_state().refresh_token().map(String::from);
        if current_refresh.as_deref() != Some(pre_lock_refresh.as_str()) {
            // Another caller already rotated. Return the in-memory tokens
            // without burning a second `/auth/refresh` round-trip — that
            // call would 401 (refresh_token rotates one-shot server-side)
            // and incorrectly cleanup an otherwise-healthy session.
            let state = self.read_state();
            let s = state.session.as_ref().ok_or(AuthError::NotAuthenticated)?;
            return Ok(AuthTokens {
                token: s.access_token.clone(),
                refresh_token: s.refresh_token.clone(),
                expires_in: Some(s.expires_at - now_unix_secs()),
            });
        }

        let req = auth_proto::RefreshTokenRequest {
            refresh_token: pre_lock_refresh,
        };
        let resp: auth_proto::RefreshTokenResponse = connect_call(
            self,
            "/proto.auth.v1.AuthService/RefreshToken",
            &req,
            None,
        )
        .await?;

        let tokens = AuthTokens {
            token: resp.token,
            refresh_token: resp.refresh_token,
            expires_in: Some(resp.expires_in),
        };

        self.write_state()
            .apply_tokens(&tokens, self.base_url(), now_unix_secs());
        self.persist();
        Ok(tokens)
    }

    pub async fn verify_email(&self, token: &str) -> Result<AuthSession, AuthError> {
        let req = auth_proto::VerifyEmailRequest {
            token: token.to_string(),
        };
        let resp: auth_proto::VerifyEmailResponse =
            connect_call(self, "/proto.auth.v1.AuthService/VerifyEmail", &req, None).await?;
        let session = session_from_verify(resp)?;
        self.apply_session(&session);
        Ok(session)
    }

    pub async fn forgot_password(&self, email: &str) -> Result<(), AuthError> {
        let req = auth_proto::ForgotPasswordRequest {
            email: email.to_string(),
        };
        let _: auth_proto::ForgotPasswordResponse = connect_call(
            self,
            "/proto.auth.v1.AuthService/ForgotPassword",
            &req,
            None,
        )
        .await?;
        Ok(())
    }

    pub async fn reset_password(
        &self,
        token: &str,
        new_password: &str,
    ) -> Result<(), AuthError> {
        let req = auth_proto::ResetPasswordRequest {
            token: token.to_string(),
            new_password: new_password.to_string(),
        };
        let _: auth_proto::ResetPasswordResponse = connect_call(
            self,
            "/proto.auth.v1.AuthService/ResetPassword",
            &req,
            None,
        )
        .await?;
        Ok(())
    }

    pub async fn fetch_me(&self) -> Result<agentsmesh_types::User, AuthError> {
        let auth = self.bearer_header()?;
        let resp: user_proto::User = connect_call(
            self,
            "/proto.user.v1.UserService/GetMe",
            &user_proto::GetMeRequest {},
            Some(&auth),
        )
        .await?;

        let user = user_from_user_proto(resp);
        self.write_state().set_user(user.clone());
        Ok(user)
    }

    pub fn apply_session(&self, session: &AuthSession) {
        self.write_state()
            .apply_session(session, self.base_url(), now_unix_secs());
        self.persist();
    }
}
