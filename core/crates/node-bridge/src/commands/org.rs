use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn org_list(&self) -> napi::Result<String> {
        let svc = self.org.lock().await;
            svc.list().await.map_err(err)
    }

    #[napi]
    pub async fn org_get(&self, slug: String) -> napi::Result<String> {
        let svc = self.org.lock().await;
            svc.get(&slug).await.map_err(err)
    }

    #[napi]
    pub async fn org_create(&self, json: String) -> napi::Result<String> {
        let svc = self.org.lock().await;
            svc.create(&json).await.map_err(err)
    }

    #[napi]
    pub async fn org_update(&self, slug: String, json: String) -> napi::Result<String> {
        let svc = self.org.lock().await;
            svc.update(&slug, &json).await.map_err(err)
    }

    #[napi]
    pub async fn org_delete(&self, slug: String) -> napi::Result<()> {
        let svc = self.org.lock().await;
            svc.delete(&slug).await.map_err(err)
    }

    #[napi]
    pub async fn org_list_members(&self, slug: String) -> napi::Result<String> {
        let svc = self.org.lock().await;
            svc.list_members(&slug).await.map_err(err)
    }

    #[napi]
    pub async fn org_invite_member(&self, slug: String, json: String) -> napi::Result<String> {
        let svc = self.org.lock().await;
            svc.invite_member(&slug, &json).await.map_err(err)
    }

    #[napi]
    pub async fn org_remove_member(&self, slug: String, user_id: i64) -> napi::Result<()> {
        let svc = self.org.lock().await;
            svc.remove_member(&slug, user_id).await.map_err(err)
    }

    #[napi]
    pub async fn org_update_member_role(&self, slug: String, user_id: i64, json: String) -> napi::Result<String> {
        let svc = self.org.lock().await;
            svc.update_member_role(&slug, user_id, &json).await.map_err(err)
    }

}
