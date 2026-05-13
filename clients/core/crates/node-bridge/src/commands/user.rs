use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn user_get_me_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.user.lock().await;
        svc.get_me_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn user_update_me_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.user.lock().await;
        svc.update_me_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn user_change_password_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.user.lock().await;
        svc.change_password_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn user_list_identities_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.user.lock().await;
        svc.list_identities_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn user_delete_identity_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.user.lock().await;
        svc.delete_identity_connect(&request).await.map_err(err)
    }

    #[napi]
    pub async fn user_search_users_connect(&self, request: Vec<u8>) -> napi::Result<Vec<u8>> {
        let svc = self.user.lock().await;
        svc.search_users_connect(&request).await.map_err(err)
    }
}
