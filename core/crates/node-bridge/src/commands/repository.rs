use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn repository_list(&self) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.list().await.map_err(err)
    }

    #[napi]
    pub async fn repository_get(&self, id: i64) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.get(id).await.map_err(err)
    }

    #[napi]
    pub async fn repository_create(&self, json: String) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.create(&json).await.map_err(err)
    }

    #[napi]
    pub async fn repository_update(&self, id: i64, json: String) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.update(id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn repository_delete(&self, id: i64) -> napi::Result<()> {
        let svc = self.repository.lock().await;
            svc.delete(id).await.map_err(err)
    }

    #[napi]
    pub async fn repository_list_branches(&self, id: i64) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.list_branches(id).await.map_err(err)
    }

    #[napi]
    pub async fn repository_sync_branches(&self, id: i64, json: String) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.sync_branches(id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn repository_register_webhook(&self, id: i64) -> napi::Result<()> {
        let svc = self.repository.lock().await;
            svc.register_webhook(id).await.map_err(err)
    }

    #[napi]
    pub async fn repository_delete_webhook(&self, id: i64) -> napi::Result<()> {
        let svc = self.repository.lock().await;
            svc.delete_webhook(id).await.map_err(err)
    }

    #[napi]
    pub async fn repository_get_webhook_status(&self, id: i64) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.get_webhook_status(id).await.map_err(err)
    }

    #[napi]
    pub async fn repository_get_webhook_secret(&self, id: i64) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.get_webhook_secret(id).await.map_err(err)
    }

    #[napi]
    pub async fn repository_list_merge_requests(&self, id: i64, branch: Option<String>, mr_state: Option<String>) -> napi::Result<String> {
        let svc = self.repository.lock().await;
            svc.list_merge_requests(id, branch, mr_state).await.map_err(err)
    }

    #[napi]
    pub async fn repository_mark_webhook_configured(&self, id: i64) -> napi::Result<()> {
        let svc = self.repository.lock().await;
            svc.mark_webhook_configured(id).await.map_err(err)
    }

}
