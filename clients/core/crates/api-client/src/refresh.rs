use prost::Message;
use reqwest::header::{HeaderName, HeaderValue};
use tracing::warn;

use crate::client::ApiClient;
use agentsmesh_types::proto_auth_v1 as auth_proto;

impl ApiClient {
    pub(crate) async fn handle_token_refresh(&self, failed_token: Option<&str>) -> bool {
        let _guard = self.refresh_lock.lock().await;

        let current_token = self.auth_store.get_token();
        if current_token.as_deref() != failed_token {
            return current_token.is_some();
        }

        let Some(refresh_token) = self.auth_store.get_refresh_token() else {
            warn!("no refresh token available");
            self.auth_store.clear_tokens();
            return false;
        };

        // Drive token refresh through Connect-RPC — the REST
        // `/api/v1/auth/refresh` handler is gone. Proto.auth.v1 owns the
        // refresh data plane (see backend/internal/api/connect/auth).
        let req = auth_proto::RefreshTokenRequest { refresh_token };
        let url = format!(
            "{}/proto.auth.v1.AuthService/RefreshToken",
            self.base_url
        );
        let result = self
            .http
            .post(&url)
            .header(
                HeaderName::from_static("content-type"),
                HeaderValue::from_static("application/proto"),
            )
            .header(
                HeaderName::from_static("connect-protocol-version"),
                HeaderValue::from_static("1"),
            )
            .body(req.encode_to_vec())
            .send()
            .await;

        match result {
            Ok(resp) if resp.status().is_success() => match resp.bytes().await {
                Ok(body) => match auth_proto::RefreshTokenResponse::decode(body) {
                    Ok(data) => {
                        self.auth_store.set_tokens(
                            data.token,
                            data.refresh_token,
                            Some(data.expires_in),
                        );
                        true
                    }
                    Err(e) => {
                        warn!("failed to decode refresh response: {e}");
                        self.auth_store.clear_tokens();
                        false
                    }
                },
                Err(e) => {
                    warn!("failed to read refresh response body: {e}");
                    self.auth_store.clear_tokens();
                    false
                }
            },
            Ok(resp) => {
                warn!("token refresh failed with status: {}", resp.status());
                self.auth_store.clear_tokens();
                false
            }
            Err(e) => {
                warn!("token refresh network error: {e}");
                self.auth_store.clear_tokens();
                false
            }
        }
    }
}
