use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn invitation_list(&self) -> napi::Result<String> {
        let svc = self.invitation.lock().await;
            svc.list().await.map_err(err)
    }

    #[napi]
    pub async fn invitation_create(&self, json: String) -> napi::Result<String> {
        let svc = self.invitation.lock().await;
            svc.create(&json).await.map_err(err)
    }

    #[napi]
    pub async fn invitation_revoke(&self, id: i64) -> napi::Result<()> {
        let svc = self.invitation.lock().await;
            svc.revoke(id).await.map_err(err)
    }

    #[napi]
    pub async fn invitation_resend(&self, id: i64) -> napi::Result<()> {
        let svc = self.invitation.lock().await;
            svc.resend(id).await.map_err(err)
    }

    #[napi]
    pub async fn invitation_get_by_token(&self, token: String) -> napi::Result<String> {
        let svc = self.invitation.lock().await;
            svc.get_by_token(&token).await.map_err(err)
    }

    #[napi]
    pub async fn invitation_accept(&self, token: String) -> napi::Result<()> {
        let svc = self.invitation.lock().await;
            svc.accept(&token).await.map_err(err)
    }

    #[napi]
    pub async fn invitation_list_pending(&self) -> napi::Result<String> {
        let svc = self.invitation.lock().await;
            svc.list_pending().await.map_err(err)
    }

}
