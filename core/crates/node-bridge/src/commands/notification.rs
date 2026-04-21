use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn notification_get_preferences(&self) -> napi::Result<String> {
        let svc = self.notification.lock().await;
            svc.get_preferences().await.map_err(err)
    }

    #[napi]
    pub async fn notification_set_preference(&self, json: String) -> napi::Result<String> {
        let svc = self.notification.lock().await;
            svc.set_preference(&json).await.map_err(err)
    }

}
