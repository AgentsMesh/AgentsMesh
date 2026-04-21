use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn grant_list(&self, resource_type: String, resource_id: String) -> napi::Result<String> {
        let svc = self.grant.lock().await;
        svc.list(&resource_type, &resource_id).await.map_err(err)
    }

    #[napi]
    pub async fn grant_create(&self, resource_type: String, resource_id: String, user_id: i64) -> napi::Result<String> {
        let svc = self.grant.lock().await;
        svc.grant(&resource_type, &resource_id, user_id).await.map_err(err)
    }

    #[napi]
    pub async fn grant_revoke(&self, resource_type: String, resource_id: String, grant_id: i64) -> napi::Result<()> {
        let svc = self.grant.lock().await;
        svc.revoke(&resource_type, &resource_id, grant_id).await.map_err(err)
    }

}
