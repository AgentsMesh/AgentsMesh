use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn binding_request_binding(&self, json: String, pod_key: Option<String>) -> napi::Result<String> {
        let svc = self.binding.lock().await;
            svc.request_binding(&json, pod_key).await.map_err(err)
    }

    #[napi]
    pub async fn binding_accept_binding(&self, json: String) -> napi::Result<String> {
        let svc = self.binding.lock().await;
            svc.accept_binding(&json).await.map_err(err)
    }

    #[napi]
    pub async fn binding_reject_binding(&self, json: String) -> napi::Result<()> {
        let svc = self.binding.lock().await;
            svc.reject_binding(&json).await.map_err(err)
    }

    #[napi]
    pub async fn binding_request_scopes(&self, binding_id: i64, json: String) -> napi::Result<String> {
        let svc = self.binding.lock().await;
            svc.request_scopes(binding_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn binding_approve_scopes(&self, binding_id: i64, json: String) -> napi::Result<String> {
        let svc = self.binding.lock().await;
            svc.approve_scopes(binding_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn binding_unbind(&self, json: String) -> napi::Result<()> {
        let svc = self.binding.lock().await;
            svc.unbind(&json).await.map_err(err)
    }

    #[napi]
    pub async fn binding_list_bindings(&self, status: Option<String>) -> napi::Result<String> {
        let svc = self.binding.lock().await;
            svc.list_bindings(status).await.map_err(err)
    }

    #[napi]
    pub async fn binding_get_pending_bindings(&self) -> napi::Result<String> {
        let svc = self.binding.lock().await;
            svc.get_pending_bindings().await.map_err(err)
    }

    #[napi]
    pub async fn binding_get_bound_pods(&self) -> napi::Result<String> {
        let svc = self.binding.lock().await;
            svc.get_bound_pods().await.map_err(err)
    }

    #[napi]
    pub async fn binding_check_binding(&self, target_pod: String) -> napi::Result<String> {
        let svc = self.binding.lock().await;
            svc.check_binding(&target_pod).await.map_err(err)
    }

}
