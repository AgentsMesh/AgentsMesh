use napi_derive::napi;
use crate::{AppState, err};

#[napi]
impl AppState {
    #[napi]
    pub async fn user_credential_list_git_credentials(&self) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.list_git_credentials().await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_create_git_credential(&self, json: String) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.create_git_credential(&json).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_get_git_credential(&self, id: i64) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.get_git_credential(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_update_git_credential(&self, id: i64, json: String) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.update_git_credential(id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_delete_git_credential(&self, id: i64) -> napi::Result<()> {
        let svc = self.user_credential.lock().await;
            svc.delete_git_credential(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_get_default_git_credential(&self) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.get_default_git_credential().await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_set_default_git_credential(&self, json: String) -> napi::Result<()> {
        let svc = self.user_credential.lock().await;
            svc.set_default_git_credential(&json).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_clear_default_git_credential(&self) -> napi::Result<()> {
        let svc = self.user_credential.lock().await;
            svc.clear_default_git_credential().await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_list_agent_credentials(&self) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.list_agent_credentials().await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_list_agent_credentials_for_agent(&self, agent_slug: String) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.list_agent_credentials_for_agent(&agent_slug).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_create_agent_credential(&self, agent_slug: String, json: String) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.create_agent_credential(&agent_slug, &json).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_get_agent_credential(&self, id: i64) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.get_agent_credential(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_update_agent_credential(&self, id: i64, json: String) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.update_agent_credential(id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_delete_agent_credential(&self, id: i64) -> napi::Result<()> {
        let svc = self.user_credential.lock().await;
            svc.delete_agent_credential(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_set_default_agent_credential(&self, id: i64) -> napi::Result<()> {
        let svc = self.user_credential.lock().await;
            svc.set_default_agent_credential(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_list_repo_providers(&self) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.list_repo_providers().await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_create_repo_provider(&self, json: String) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.create_repo_provider(&json).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_get_repo_provider(&self, id: i64) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.get_repo_provider(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_update_repo_provider(&self, id: i64, json: String) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.update_repo_provider(id, &json).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_delete_repo_provider(&self, id: i64) -> napi::Result<()> {
        let svc = self.user_credential.lock().await;
            svc.delete_repo_provider(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_set_default_repo_provider(&self, id: i64) -> napi::Result<()> {
        let svc = self.user_credential.lock().await;
            svc.set_default_repo_provider(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_test_repo_provider(&self, id: i64) -> napi::Result<()> {
        let svc = self.user_credential.lock().await;
            svc.test_repo_provider(id).await.map_err(err)
    }

    #[napi]
    pub async fn user_credential_list_provider_repositories(&self, id: i64, page: Option<u32>, per_page: Option<u32>, search: Option<String>) -> napi::Result<String> {
        let svc = self.user_credential.lock().await;
            svc.list_provider_repositories(id, page, per_page, search).await.map_err(err)
    }

}
