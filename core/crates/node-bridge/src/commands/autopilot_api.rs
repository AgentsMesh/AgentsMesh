use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn autopilot_fetch_controllers(&self) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            svc.fetch_controllers().await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_fetch_controller(&self, key: String) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            svc.fetch_controller(&key).await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_create_controller(&self, request_json: String) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            svc.create_controller(&request_json).await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_pause_controller(&self, key: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.pause_controller(&key).await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_resume_controller(&self, key: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.resume_controller(&key).await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_stop_controller(&self, key: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.stop_controller(&key).await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_approve_controller(&self, key: String, request_json: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.approve_controller(&key, &request_json).await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_takeover_controller(&self, key: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.takeover_controller(&key).await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_handback_controller(&self, key: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.handback_controller(&key).await.map_err(err)
    }

    #[napi]
    pub async fn autopilot_fetch_iterations(&self, key: String) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            svc.fetch_iterations(&key).await.map_err(err)
    }

}
