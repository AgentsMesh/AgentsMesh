use napi_derive::napi;
use crate::AppState;

#[napi]
impl AppState {
    #[napi]
    pub async fn runner_runners_json(&self) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            Ok(svc.runners_json())
    }

    #[napi]
    pub async fn runner_available_runners_json(&self) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            Ok(svc.available_runners_json())
    }

    #[napi]
    pub async fn runner_current_runner_json(&self) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            Ok(svc.current_runner_json().unwrap_or_default())
    }

    #[napi]
    pub async fn runner_get_runner_json(&self, id: i64) -> napi::Result<String> {
        let svc = self.runner.lock().await;
            Ok(svc.get_runner_json(id).unwrap_or_default())
    }

    #[napi]
    pub async fn runner_set_runners(&self, json: String) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.set_runners(&json);
            Ok(())
    }

    #[napi]
    pub async fn runner_set_available_runners(&self, json: String) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.set_available_runners(&json);
            Ok(())
    }

    #[napi]
    pub async fn runner_set_current_runner(&self, json: String) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.set_current_runner(&json);
            Ok(())
    }

    #[napi]
    pub async fn runner_update_runner_local(&self, id: f64, json: String) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.update_runner_local(id, &json);
            Ok(())
    }

    #[napi]
    pub async fn runner_update_runner_status(&self, id: i64, status: String) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.update_runner_status(id, &status);
            Ok(())
    }

    #[napi]
    pub async fn runner_remove_runner_local(&self, id: i64) -> napi::Result<()> {
        let svc = self.runner.lock().await;
            svc.remove_runner_local(id);
            Ok(())
    }

}
