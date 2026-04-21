use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn runner_fetch_runners(&self, status: Option<String>) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.fetch_runners(status).await.map_err(err)
    }

    #[napi]
    pub async fn runner_fetch_available_runners(&self) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.fetch_available_runners().await.map_err(err)
    }

    #[napi]
    pub async fn runner_fetch_runner(&self, id: i64) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.fetch_runner(id).await.map_err(err)
    }

    #[napi]
    pub async fn runner_update_runner(&self, id: i64, request_json: String) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.update_runner(id, &request_json).await.map_err(err)
    }

    #[napi]
    pub async fn runner_delete_runner(&self, id: i64) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.delete_runner(id).await.map_err(err)
    }

    #[napi]
    pub async fn runner_create_token(&self, request_json: String) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.create_token(&request_json).await.map_err(err)
    }

    #[napi]
    pub async fn runner_fetch_tokens(&self) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.fetch_tokens().await.map_err(err)
    }

    #[napi]
    pub async fn runner_delete_token(&self, id: i64) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.delete_token(id).await.map_err(err)
    }

    #[napi]
    pub async fn runner_list_runner_logs(&self, id: i64) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.list_runner_logs(id).await.map_err(err)
    }

    #[napi]
    pub async fn runner_request_log_upload(&self, id: i64) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.request_log_upload(id).await.map_err(err)
    }

    #[napi]
    pub async fn runner_upgrade_runner(&self, id: i64, request_json: String) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.upgrade_runner(id, &request_json).await.map_err(err)
    }

    #[napi]
    pub async fn runner_list_runner_pods(&self, id: i64, status: Option<String>, limit: Option<u32>, offset: Option<u32>) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.list_runner_pods(id, status, limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn runner_query_runner_sandboxes(&self, id: i64, request_json: String) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.query_runner_sandboxes(id, &request_json).await.map_err(err)
    }

    #[napi]
    pub async fn runner_get_auth_status(&self, auth_key: String) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.get_auth_status(&auth_key).await.map_err(err)
    }

    #[napi]
    pub async fn runner_authorize_runner(&self, request_json: String) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            svc.authorize_runner(&request_json).await.map_err(err)
    }

}
