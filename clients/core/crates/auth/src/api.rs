use agentsmesh_types::{AuthSession, AuthTokens, RegisterRequest};

use crate::error::{parse_error_response, AuthError};
use crate::manager::{now_unix_secs, AuthManager};

impl AuthManager {
    pub async fn login(&self, email: &str, password: &str) -> Result<AuthSession, AuthError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/auth/login", self.base_url))
            .json(&serde_json::json!({ "email": email, "password": password }))
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }

        let session: AuthSession = resp
            .json()
            .await
            .map_err(|e| AuthError::InvalidResponse(e.to_string()))?;

        self.apply_session(&session);
        Ok(session)
    }

    pub async fn register(&self, data: &RegisterRequest) -> Result<AuthSession, AuthError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/auth/register", self.base_url))
            .json(data)
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }

        let session: AuthSession = resp
            .json()
            .await
            .map_err(|e| AuthError::InvalidResponse(e.to_string()))?;

        self.apply_session(&session);
        Ok(session)
    }

    pub async fn logout(&self) -> Result<(), AuthError> {
        let auth = self.bearer_header()?;

        let server_result = self
            .http
            .post(format!("{}/api/v1/auth/logout", self.base_url))
            .header("Authorization", auth)
            .send()
            .await;

        self.reset_local();

        match server_result {
            Ok(resp) if !resp.status().is_success() => {
                tracing::warn!(
                    "auth logout: server returned {}; local state cleared anyway",
                    resp.status()
                );
            }
            Err(e) => {
                tracing::warn!(
                    "auth logout: server unreachable ({e}); local state cleared anyway"
                );
            }
            Ok(_) => {}
        }
        Ok(())
    }

    pub async fn refresh_token(&self) -> Result<AuthTokens, AuthError> {
        // Snapshot the refresh token BEFORE blocking on the lock so we can
        // detect a concurrent rotation by another caller on the same manager.
        let pre_lock_refresh = self
            .read_state()
            .refresh_token()
            .map(String::from)
            .ok_or(AuthError::NotAuthenticated)?;

        let _guard = self.refresh_lock.lock().await;

        let current_refresh = self.read_state().refresh_token().map(String::from);
        if current_refresh.as_deref() != Some(pre_lock_refresh.as_str()) {
            // Another caller already rotated. Returning the in-memory tokens
            // avoids a second `/auth/refresh` round-trip that would 401
            // (refresh_token rotates one-shot server-side) and incorrectly
            // clean up an otherwise-healthy session.
            let state = self.read_state();
            let s = state.session.as_ref().ok_or(AuthError::NotAuthenticated)?;
            return Ok(AuthTokens {
                token: s.access_token.clone(),
                refresh_token: s.refresh_token.clone(),
                expires_in: Some(s.expires_at - now_unix_secs()),
            });
        }

        let resp = self
            .http
            .post(format!("{}/api/v1/auth/refresh", self.base_url))
            .json(&serde_json::json!({ "refresh_token": pre_lock_refresh }))
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }

        let tokens: AuthTokens = resp
            .json()
            .await
            .map_err(|e| AuthError::InvalidResponse(e.to_string()))?;

        self.write_state()
            .apply_tokens(&tokens, &self.base_url, now_unix_secs());
        self.persist();
        Ok(tokens)
    }

    pub async fn verify_email(&self, token: &str) -> Result<AuthSession, AuthError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/auth/verify-email", self.base_url))
            .json(&serde_json::json!({ "token": token }))
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }

        let session: AuthSession = resp
            .json()
            .await
            .map_err(|e| AuthError::InvalidResponse(e.to_string()))?;

        self.apply_session(&session);
        Ok(session)
    }

    pub async fn forgot_password(&self, email: &str) -> Result<(), AuthError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/auth/forgot-password", self.base_url))
            .json(&serde_json::json!({ "email": email }))
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }
        Ok(())
    }

    pub async fn reset_password(
        &self,
        token: &str,
        new_password: &str,
    ) -> Result<(), AuthError> {
        let resp = self
            .http
            .post(format!("{}/api/v1/auth/reset-password", self.base_url))
            .json(&serde_json::json!({
                "token": token,
                "new_password": new_password,
            }))
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }
        Ok(())
    }

    pub async fn fetch_me(&self) -> Result<agentsmesh_types::User, AuthError> {
        let auth = self.bearer_header()?;
        let resp = self
            .http
            .get(format!("{}/api/v1/users/me", self.base_url))
            .header("Authorization", auth)
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }

        #[derive(serde::Deserialize)]
        struct Wrapper {
            user: agentsmesh_types::User,
        }
        let wrapper: Wrapper = resp
            .json()
            .await
            .map_err(|e| AuthError::InvalidResponse(e.to_string()))?;

        self.write_state().set_user(wrapper.user.clone());
        Ok(wrapper.user)
    }

    pub fn apply_session(&self, session: &AuthSession) {
        self.write_state()
            .apply_session(session, &self.base_url, now_unix_secs());
        self.persist();
    }
}
