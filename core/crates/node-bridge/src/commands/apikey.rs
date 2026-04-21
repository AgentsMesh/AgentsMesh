use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn apikey_list(&self) -> napi::Result<String> {
        let svc = self.apikey.lock().await;
            svc.list().await.map_err(err)
    }

    #[napi]
    pub async fn apikey_get(&self, id: i64) -> napi::Result<String> {
        let svc = self.apikey.lock().await;
            svc.get(id).await.map_err(err)
    }

    #[napi]
    pub async fn apikey_create(&self, json: String) -> napi::Result<String> {
        let svc = self.apikey.lock().await;
            svc.create(&json).await.map_err(err)
    }

    #[napi]
    pub async fn apikey_update(&self, id: i64, json: String) -> napi::Result<String> {
        let svc = self.apikey.lock().await;
            svc.update(id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn apikey_delete(&self, id: i64) -> napi::Result<()> {
        let svc = self.apikey.lock().await;
            svc.delete(id).await.map_err(err)
    }

    #[napi]
    pub async fn apikey_revoke(&self, id: i64) -> napi::Result<()> {
        let svc = self.apikey.lock().await;
            svc.revoke(id).await.map_err(err)
    }

}
