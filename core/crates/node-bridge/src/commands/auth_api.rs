use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn auth_api_register(&self, json: String) -> napi::Result<String> {
        let svc = self.auth_api.lock().await;
            svc.register(&json).await.map_err(err)
    }

    #[napi]
    pub async fn auth_api_verify_email(&self, token: String) -> napi::Result<String> {
        let svc = self.auth_api.lock().await;
            svc.verify_email(&token).await.map_err(err)
    }

    #[napi]
    pub async fn auth_api_resend_verification(&self, email: String) -> napi::Result<String> {
        let svc = self.auth_api.lock().await;
            svc.resend_verification(&email).await.map_err(err)
    }

    #[napi]
    pub async fn auth_api_forgot_password(&self, email: String) -> napi::Result<String> {
        let svc = self.auth_api.lock().await;
            svc.forgot_password(&email).await.map_err(err)
    }

    #[napi]
    pub async fn auth_api_reset_password(&self, json: String) -> napi::Result<String> {
        let svc = self.auth_api.lock().await;
            svc.reset_password(&json).await.map_err(err)
    }

}
