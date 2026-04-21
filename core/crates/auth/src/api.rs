use agentsmesh_types::{AuthSession, AuthTokens, RegisterRequest};

use crate::error::{parse_error_response, AuthError};
use crate::manager::AuthManager;

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
        let resp = self
            .http
            .post(format!("{}/api/v1/auth/logout", self.base_url))
            .header("Authorization", auth)
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }

        self.state.write().unwrap_or_else(|e| e.into_inner()).clear();
        self.storage.remove(crate::state::STORAGE_KEY);
        Ok(())
    }

    pub async fn refresh_token(&self) -> Result<AuthTokens, AuthError> {
        let refresh = self
            .state
            .read()
            .unwrap_or_else(|e| e.into_inner())
            .refresh_token
            .clone()
            .ok_or(AuthError::NotAuthenticated)?;

        let resp = self
            .http
            .post(format!("{}/api/v1/auth/refresh", self.base_url))
            .json(&serde_json::json!({ "refresh_token": refresh }))
            .send()
            .await?;

        if !resp.status().is_success() {
            return Err(parse_error_response(resp).await);
        }

        let tokens: AuthTokens = resp
            .json()
            .await
            .map_err(|e| AuthError::InvalidResponse(e.to_string()))?;

        self.state.write().unwrap_or_else(|e| e.into_inner()).apply_tokens(&tokens);
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

    pub fn apply_session(&self, session: &AuthSession) {
        self.state.write().unwrap_or_else(|e| e.into_inner()).apply_session(session);
        self.persist();
    }
}
