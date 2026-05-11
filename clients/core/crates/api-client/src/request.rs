use std::collections::HashMap;
use std::time::Duration;

use reqwest::Method;
use serde::de::DeserializeOwned;

use crate::client::ApiClient;
use crate::error::ApiError;

pub struct RequestOptions {
    pub headers: Option<HashMap<String, String>>,
    pub body: Option<serde_json::Value>,
    pub skip_auth_refresh: bool,
    pub timeout: Option<Duration>,
}

impl Default for RequestOptions {
    fn default() -> Self {
        Self {
            headers: None,
            body: None,
            skip_auth_refresh: false,
            timeout: None,
        }
    }
}

impl ApiClient {
    pub async fn request<T: DeserializeOwned>(
        &self,
        method: Method,
        endpoint: &str,
        options: RequestOptions,
    ) -> Result<T, ApiError> {
        let token = self.auth_store.get_token();
        match self
            .send_request::<T>(&method, endpoint, &options, token.as_deref())
            .await
        {
            Err(ApiError::Http { status: 401, .. }) if !options.skip_auth_refresh => {
                if self.handle_token_refresh(token.as_deref()).await {
                    let new_token = self.auth_store.get_token();
                    self.send_request(&method, endpoint, &options, new_token.as_deref())
                        .await
                } else {
                    Err(ApiError::AuthExpired)
                }
            }
            result => result,
        }
    }

    pub async fn request_no_auth<T: DeserializeOwned>(
        &self,
        method: Method,
        endpoint: &str,
        options: RequestOptions,
    ) -> Result<T, ApiError> {
        self.send_request(&method, endpoint, &options, None).await
    }

    async fn send_request<T: DeserializeOwned>(
        &self,
        method: &Method,
        endpoint: &str,
        options: &RequestOptions,
        token: Option<&str>,
    ) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, endpoint);
        let mut builder = self
            .http
            .request(method.clone(), &url)
            .header("Content-Type", "application/json");

        if let Some(token) = token {
            builder = builder.header("Authorization", format!("Bearer {token}"));
        }

        if let Some(headers) = &options.headers {
            for (k, v) in headers {
                builder = builder.header(k.as_str(), v.as_str());
            }
        }

        if let Some(body) = &options.body {
            builder = builder.json(body);
        }

        if let Some(timeout) = options.timeout {
            builder = builder.timeout(timeout);
        }

        let resp = builder.send().await?;
        Self::parse_response(resp, &url).await
    }

    async fn parse_response<T: DeserializeOwned>(
        resp: reqwest::Response,
        url: &str,
    ) -> Result<T, ApiError> {
        let status = resp.status();

        if status.is_success() {
            let bytes = resp.bytes().await?;
            if bytes.is_empty() {
                return Ok(serde_json::from_value(serde_json::json!({}))?);
            }
            Ok(serde_json::from_slice(&bytes)?)
        } else {
            let status_code = status.as_u16();
            let status_text = status
                .canonical_reason()
                .unwrap_or("Unknown")
                .to_string();
            let body: serde_json::Value = resp.json().await.unwrap_or_default();

            Err(ApiError::Http {
                status: status_code,
                status_text,
                code: body.get("code").and_then(|v| v.as_str()).map(String::from),
                server_message: body
                    .get("error")
                    .and_then(|v| v.as_str())
                    .map(String::from),
                data: Some(body),
                url: Some(url.to_string()),
            })
        }
    }

    pub async fn post_multipart<T: DeserializeOwned>(
        &self,
        endpoint: &str,
        form: reqwest::multipart::Form,
    ) -> Result<T, ApiError> {
        let token = self.auth_store.get_token();
        self.send_multipart::<T>(endpoint, form, token.as_deref()).await
    }

    async fn send_multipart<T: DeserializeOwned>(
        &self,
        endpoint: &str,
        form: reqwest::multipart::Form,
        token: Option<&str>,
    ) -> Result<T, ApiError> {
        let url = format!("{}{}", self.base_url, endpoint);
        let mut builder = self.http.post(&url);
        if let Some(token) = token {
            builder = builder.header("Authorization", format!("Bearer {token}"));
        }
        builder = builder.multipart(form);
        let resp = builder.send().await?;
        Self::parse_response(resp, &url).await
    }
}
