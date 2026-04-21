use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn pod_pods_json(&self) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            Ok(svc.pods_json())
    }

    #[napi]
    pub async fn pod_current_pod_json(&self) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            Ok(svc.current_pod_json().unwrap_or_default())
    }

    #[napi]
    pub async fn pod_get_pod_json(&self, pod_key: String) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            Ok(svc.get_pod_json(&pod_key).unwrap_or_default())
    }

    #[napi]
    pub async fn pod_upsert_pod(&self, pod_json: String, timestamp: Option<i64>) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.upsert_pod(&pod_json, timestamp);
            Ok(())
    }

    #[napi]
    pub async fn pod_set_pods(&self, pods_json: String) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.set_pods(&pods_json);
            Ok(())
    }

    #[napi]
    pub async fn pod_set_current_pod(&self, pod_json: String) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.set_current_pod(&pod_json);
            Ok(())
    }

    #[napi]
    pub async fn pod_update_pod_status(&self, pod_key: String, status: String, agent_status: Option<String>, error_code: Option<String>, error_message: Option<String>, timestamp: Option<i64>) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.update_pod_status(&pod_key, &status, agent_status, error_code, error_message, timestamp);
            Ok(())
    }

    #[napi]
    pub async fn pod_update_pod_title(&self, pod_key: String, title: String, timestamp: Option<i64>) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.update_pod_title(&pod_key, &title, timestamp);
            Ok(())
    }

    #[napi]
    pub async fn pod_update_pod_alias(&self, pod_key: String, alias: String) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.update_pod_alias(&pod_key, &alias);
            Ok(())
    }

    #[napi]
    pub async fn pod_update_agent_status(&self, pod_key: String, agent_status: String) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.update_agent_status(&pod_key, &agent_status);
            Ok(())
    }

    #[napi]
    pub async fn pod_remove_pod(&self, pod_key: String) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.remove_pod(&pod_key);
            Ok(())
    }

    #[napi]
    pub async fn pod_fetch_pods(&self, status: Option<String>, runner_id: Option<i64>, created_by_id: Option<i64>, limit: Option<i64>, offset: Option<i64>) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            svc.fetch_pods(status, runner_id, created_by_id, limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn pod_fetch_sidebar_pods(&self, filter: String, user_id: Option<i64>) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            svc.fetch_sidebar_pods(&filter, user_id).await.map_err(err)
    }

    #[napi]
    pub async fn pod_load_more_pods(&self, filter: String, user_id: Option<i64>, offset: i64) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            svc.load_more_pods(&filter, user_id, offset).await.map_err(err)
    }

    #[napi]
    pub async fn pod_fetch_pod(&self, pod_key: String) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            svc.fetch_pod(&pod_key).await.map_err(err)
    }

    #[napi]
    pub async fn pod_create_pod(&self, request_json: String) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            svc.create_pod(&request_json).await.map_err(err)
    }

    #[napi]
    pub async fn pod_terminate_pod(&self, pod_key: String) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.terminate_pod(&pod_key).await.map_err(err)
    }

    #[napi]
    pub async fn pod_update_pod_alias_api(&self, pod_key: String, alias: Option<String>) -> napi::Result<()> {
        let svc = self.pod.lock().await;
            svc.update_pod_alias_api(&pod_key, alias).await.map_err(err)
    }

    #[napi]
    pub async fn pod_get_pod_connection(&self, pod_key: String) -> napi::Result<String> {
        let svc = self.pod.lock().await;
            svc.get_pod_connection(&pod_key).await.map_err(err)
    }

}
