use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn auth_connect_login_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.login_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_register_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.register_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_refresh_token_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.refresh_token_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_verify_email_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.verify_email_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_resend_verification_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.resend_verification_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_forgot_password_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.forgot_password_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_reset_password_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.reset_password_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_oauth_redirect_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.oauth_redirect_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_oauth_callback_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.oauth_callback_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn auth_connect_logout_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.auth_connect.lock().await;
        svc.logout_connect(&request).await.map_err(err)
    }
}
