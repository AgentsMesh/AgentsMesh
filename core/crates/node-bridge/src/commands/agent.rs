use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn agent_list_agents(&self) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.list_agents().await.map_err(err)
    }

    #[napi]
    pub async fn agent_get_config_schema(&self, agent_slug: String) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.get_config_schema(&agent_slug).await.map_err(err)
    }

    #[napi]
    pub async fn agent_list_user_configs(&self) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.list_user_configs().await.map_err(err)
    }

    #[napi]
    pub async fn agent_get_user_config(&self, agent_slug: String) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.get_user_config(&agent_slug).await.map_err(err)
    }

    #[napi]
    pub async fn agent_set_user_config(&self, agent_slug: String, json: String) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.set_user_config(&agent_slug, &json).await.map_err(err)
    }

    #[napi]
    pub async fn agent_delete_user_config(&self, agent_slug: String) -> napi::Result<()> {
        let svc = self.agent.lock().await;
            svc.delete_user_config(&agent_slug).await.map_err(err)
    }

    #[napi]
    pub async fn agent_get_agentpod_settings(&self) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.get_agentpod_settings().await.map_err(err)
    }

    #[napi]
    pub async fn agent_update_agentpod_settings(&self, json: String) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.update_agentpod_settings(&json).await.map_err(err)
    }

    #[napi]
    pub async fn agent_list_providers(&self) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.list_providers().await.map_err(err)
    }

    #[napi]
    pub async fn agent_create_provider(&self, json: String) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.create_provider(&json).await.map_err(err)
    }

    #[napi]
    pub async fn agent_update_provider(&self, id: i64, json: String) -> napi::Result<String> {
        let svc = self.agent.lock().await;
            svc.update_provider(id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn agent_delete_provider(&self, id: i64) -> napi::Result<()> {
        let svc = self.agent.lock().await;
            svc.delete_provider(id).await.map_err(err)
    }

    #[napi]
    pub async fn agent_set_default_provider(&self, id: i64) -> napi::Result<()> {
        let svc = self.agent.lock().await;
            svc.set_default_provider(id).await.map_err(err)
    }

}
