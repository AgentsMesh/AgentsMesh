use std::sync::Arc;

use futures::lock::Mutex;
use reqwest::{Client, Method};
use serde::de::DeserializeOwned;
use serde::Serialize;

use crate::error::ApiError;
use crate::request::RequestOptions;
use crate::token_store::AuthTokenStore;

pub struct ApiClient {
    pub(crate) http: Client,
    pub(crate) base_url: String,
    pub(crate) auth_store: Arc<dyn AuthTokenStore>,
    pub(crate) refresh_lock: Mutex<()>,
}

impl ApiClient {
    pub fn new(base_url: String, auth_store: Arc<dyn AuthTokenStore>) -> Self {
        Self {
            http: Client::new(),
            base_url,
            auth_store,
            refresh_lock: Mutex::new(()),
        }
    }

    pub fn org_path(&self, path: &str) -> String {
        match self.auth_store.get_current_org_slug() {
            Some(slug) => format!("/api/v1/orgs/{slug}{path}"),
            None => format!("/api/v1{path}"),
        }
    }

    pub async fn get<T: DeserializeOwned>(&self, endpoint: &str) -> Result<T, ApiError> {
        self.request(Method::GET, endpoint, RequestOptions::default())
            .await
    }

    pub async fn get_resource<T: DeserializeOwned>(
        &self, endpoint: &str, wrapper_key: &str,
    ) -> Result<T, ApiError> {
        let val: serde_json::Value = self.get(endpoint).await?;
        let inner = val.get(wrapper_key).unwrap_or(&val);
        serde_json::from_value(inner.clone()).map_err(ApiError::Json)
    }

    fn unwrap_resource<T: DeserializeOwned>(
        val: serde_json::Value, wrapper_key: &str,
    ) -> Result<T, ApiError> {
        let inner = val.get(wrapper_key).unwrap_or(&val);
        serde_json::from_value(inner.clone()).map_err(ApiError::Json)
    }

    pub async fn post_resource<T: DeserializeOwned>(
        &self, endpoint: &str, body: &impl Serialize, wrapper_key: &str,
    ) -> Result<T, ApiError> {
        let val: serde_json::Value = self.post(endpoint, body).await?;
        Self::unwrap_resource(val, wrapper_key)
    }

    pub async fn put_resource<T: DeserializeOwned>(
        &self, endpoint: &str, body: &impl Serialize, wrapper_key: &str,
    ) -> Result<T, ApiError> {
        let val: serde_json::Value = self.put(endpoint, body).await?;
        Self::unwrap_resource(val, wrapper_key)
    }

    pub async fn patch_resource<T: DeserializeOwned>(
        &self, endpoint: &str, body: &impl Serialize, wrapper_key: &str,
    ) -> Result<T, ApiError> {
        let val: serde_json::Value = self.patch(endpoint, body).await?;
        Self::unwrap_resource(val, wrapper_key)
    }

    pub async fn post<T: DeserializeOwned>(
        &self,
        endpoint: &str,
        body: &impl Serialize,
    ) -> Result<T, ApiError> {
        let opts = RequestOptions {
            body: Some(serde_json::to_value(body)?),
            ..Default::default()
        };
        self.request(Method::POST, endpoint, opts).await
    }

    pub async fn put<T: DeserializeOwned>(
        &self,
        endpoint: &str,
        body: &impl Serialize,
    ) -> Result<T, ApiError> {
        let opts = RequestOptions {
            body: Some(serde_json::to_value(body)?),
            ..Default::default()
        };
        self.request(Method::PUT, endpoint, opts).await
    }

    pub async fn delete<T: DeserializeOwned>(&self, endpoint: &str) -> Result<T, ApiError> {
        self.request(Method::DELETE, endpoint, RequestOptions::default())
            .await
    }

    pub async fn patch<T: DeserializeOwned>(
        &self,
        endpoint: &str,
        body: &impl Serialize,
    ) -> Result<T, ApiError> {
        let opts = RequestOptions {
            body: Some(serde_json::to_value(body)?),
            ..Default::default()
        };
        self.request(Method::PATCH, endpoint, opts).await
    }

    pub async fn public_get<T: DeserializeOwned>(&self, endpoint: &str) -> Result<T, ApiError> {
        self.request_no_auth(Method::GET, endpoint, RequestOptions::default())
            .await
    }

    pub async fn public_get_resource<T: DeserializeOwned>(
        &self, endpoint: &str, wrapper_key: &str,
    ) -> Result<T, ApiError> {
        let val: serde_json::Value = self.public_get(endpoint).await?;
        let inner = val.get(wrapper_key).unwrap_or(&val);
        serde_json::from_value(inner.clone()).map_err(ApiError::Json)
    }

    pub async fn public_post<T: DeserializeOwned>(
        &self,
        endpoint: &str,
        body: &impl Serialize,
    ) -> Result<T, ApiError> {
        let opts = RequestOptions {
            body: Some(serde_json::to_value(body)?),
            ..Default::default()
        };
        self.request_no_auth(Method::POST, endpoint, opts).await
    }

    pub async fn put_raw_bytes(
        &self, url: &str, content_type: &str, body: Vec<u8>,
    ) -> Result<(), ApiError> {
        // `url` here is the full upload target supplied by the caller
        // (e.g. an S3 presigned URL), NOT prefixed with `self.base_url`.
        // The error's `url` field reflects that, so a 5xx from S3 says
        // "@ https://bucket.s3.amazonaws.com/..." not the AgentsMesh API.
        let resp = self.http
            .put(url)
            .header("Content-Type", content_type)
            .body(body)
            .send()
            .await?;
        if !resp.status().is_success() {
            return Err(ApiError::Http {
                status: resp.status().as_u16(),
                status_text: resp.status().canonical_reason().unwrap_or("Unknown").to_string(),
                code: None, server_message: None, data: None,
                url: Some(url.to_string()),
            });
        }
        Ok(())
    }
}
