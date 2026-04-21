use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn token_usage_get_dashboard(
        &self,
        start_time: Option<String>,
        end_time: Option<String>,
        agent_slug: Option<String>,
        user_id: Option<i64>,
        model: Option<String>,
        granularity: Option<String>,
    ) -> napi::Result<String> {
        let svc = self.token_usage.lock().await;
        svc.get_dashboard(start_time, end_time, agent_slug, user_id, model, granularity)
            .await
            .map_err(err)
    }

}
