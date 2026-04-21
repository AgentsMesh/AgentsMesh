use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn user_get_me(&self) -> napi::Result<String> {
        let svc = self.user.lock().await;
            svc.get_me().await.map_err(err)
    }

    #[napi]
    pub async fn user_get_organizations(&self) -> napi::Result<String> {
        let svc = self.user.lock().await;
            svc.get_organizations().await.map_err(err)
    }

}
