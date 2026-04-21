use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn extension_list_skill_registries(&self) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.list_skill_registries().await.map_err(err)
    }

    #[napi]
    pub async fn extension_create_skill_registry(&self, json: String) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.create_skill_registry(&json).await.map_err(err)
    }

    #[napi]
    pub async fn extension_sync_skill_registry(&self, id: i64) -> napi::Result<()> {
        let svc = self.extension.lock().await;
            svc.sync_skill_registry(id).await.map_err(err)
    }

    #[napi]
    pub async fn extension_toggle_skill_registry(&self, id: i64, json: String) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.toggle_skill_registry(id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn extension_delete_skill_registry(&self, id: i64) -> napi::Result<()> {
        let svc = self.extension.lock().await;
            svc.delete_skill_registry(id).await.map_err(err)
    }

    #[napi]
    pub async fn extension_list_skill_registry_overrides(&self) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.list_skill_registry_overrides().await.map_err(err)
    }

    #[napi]
    pub async fn extension_list_market_skills(&self, query: Option<String>, category: Option<String>) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.list_market_skills(query, category).await.map_err(err)
    }

    #[napi]
    pub async fn extension_list_market_mcp_servers(&self, query: Option<String>, limit: Option<u32>, offset: Option<u32>) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.list_market_mcp_servers(query, limit, offset).await.map_err(err)
    }

    #[napi]
    pub async fn extension_list_repo_skills(&self, repo_id: i64, scope: Option<String>) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.list_repo_skills(repo_id, scope).await.map_err(err)
    }

    #[napi]
    pub async fn extension_install_skill_from_market(&self, repo_id: i64, json: String) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.install_skill_from_market(repo_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn extension_install_skill_from_github(&self, repo_id: i64, json: String) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.install_skill_from_github(repo_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn extension_update_skill(&self, repo_id: i64, install_id: i64, json: String) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.update_skill(repo_id, install_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn extension_uninstall_skill(&self, repo_id: i64, install_id: i64) -> napi::Result<()> {
        let svc = self.extension.lock().await;
            svc.uninstall_skill(repo_id, install_id).await.map_err(err)
    }

    #[napi]
    pub async fn extension_list_repo_mcp_servers(&self, repo_id: i64, scope: Option<String>) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.list_repo_mcp_servers(repo_id, scope).await.map_err(err)
    }

    #[napi]
    pub async fn extension_install_mcp_from_market(&self, repo_id: i64, json: String) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.install_mcp_from_market(repo_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn extension_install_custom_mcp_server(&self, repo_id: i64, json: String) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.install_custom_mcp_server(repo_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn extension_update_mcp_server(&self, repo_id: i64, install_id: i64, json: String) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.update_mcp_server(repo_id, install_id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn extension_uninstall_mcp_server(&self, repo_id: i64, install_id: i64) -> napi::Result<()> {
        let svc = self.extension.lock().await;
            svc.uninstall_mcp_server(repo_id, install_id).await.map_err(err)
    }

    #[napi]
    pub async fn extension_install_skill_from_upload(&self, repo_id: i64, file_data: Vec<u8>, file_name: String, scope: Option<String>) -> napi::Result<String> {
        let svc = self.extension.lock().await;
            svc.install_skill_from_upload(repo_id, file_data, &file_name, scope).await.map_err(err)
    }

}
