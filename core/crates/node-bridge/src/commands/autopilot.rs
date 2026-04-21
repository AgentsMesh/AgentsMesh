use napi_derive::napi;
use crate::AppState;

#[napi]
impl AppState {
    #[napi]
    pub async fn autopilot_controllers_json(&self) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            Ok(svc.controllers_json())
    }

    #[napi]
    pub async fn autopilot_current_controller_json(&self) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            Ok(svc.current_controller_json().unwrap_or_default())
    }

    #[napi]
    pub async fn autopilot_get_controller_by_pod_key_json(&self, pod_key: String) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            Ok(svc.get_controller_by_pod_key_json(&pod_key).unwrap_or_default())
    }

    #[napi]
    pub async fn autopilot_get_iterations_json(&self, key: String) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            Ok(svc.get_iterations_json(&key).unwrap_or_default())
    }

    #[napi]
    pub async fn autopilot_get_thinking_json(&self, key: String) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            Ok(svc.get_thinking_json(&key).unwrap_or_default())
    }

    #[napi]
    pub async fn autopilot_get_thinking_history_json(&self, key: String) -> napi::Result<String> {
        let svc = self.autopilot.lock().await;
            Ok(svc.get_thinking_history_json(&key).unwrap_or_default())
    }

    #[napi]
    pub async fn autopilot_set_controllers(&self, json: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.set_controllers(&json);
            Ok(())
    }

    #[napi]
    pub async fn autopilot_set_current_controller(&self, json: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.set_current_controller(&json);
            Ok(())
    }

    #[napi]
    pub async fn autopilot_add_controller(&self, json: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.add_controller(&json);
            Ok(())
    }

    #[napi]
    pub async fn autopilot_update_controller(&self, key: String, json: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.update_controller(&key, &json);
            Ok(())
    }

    #[napi]
    pub async fn autopilot_remove_controller(&self, key: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.remove_controller(&key);
            Ok(())
    }

    #[napi]
    pub async fn autopilot_set_iterations(&self, key: String, json: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.set_iterations(&key, &json);
            Ok(())
    }

    #[napi]
    pub async fn autopilot_add_iteration(&self, key: String, json: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.add_iteration(&key, &json);
            Ok(())
    }

    #[napi]
    pub async fn autopilot_update_thinking(&self, key: String, json: String) -> napi::Result<()> {
        let svc = self.autopilot.lock().await;
            svc.update_thinking(&key, &json);
            Ok(())
    }

}
