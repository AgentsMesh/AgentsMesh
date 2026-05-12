use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::*;
use agentsmesh_types::proto_extension_v1 as ext_proto;
use prost::Message;

pub struct ExtensionService {
    client: Arc<ApiClient>,
}

impl ExtensionService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) --------
    //
    // Each method accepts a prost-encoded request body (`Vec<u8>`) and returns
    // a prost-encoded response body — matching the wasm bridge's
    // `Result<Vec<u8>, String>` surface (conventions §2.5).
    //
    // org_slug is sourced from the caller-supplied request, not from
    // AuthManager — keeps these methods unit-testable without an org context
    // in the token store. The wasm bridge populates org_slug before encoding.

    pub async fn list_skill_registries_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::ListSkillRegistriesRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_skill_registries request: {e}"))?;
        let resp = self.client.list_skill_registries_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_skill_registry_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::CreateSkillRegistryRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_skill_registry request: {e}"))?;
        let resp = self.client.create_skill_registry_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn sync_skill_registry_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::SyncSkillRegistryRequest::decode(request_bytes)
            .map_err(|e| format!("decode sync_skill_registry request: {e}"))?;
        let resp = self.client.sync_skill_registry_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_skill_registry_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::DeleteSkillRegistryRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_skill_registry request: {e}"))?;
        let resp = self.client.delete_skill_registry_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn toggle_platform_registry_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::TogglePlatformRegistryRequest::decode(request_bytes)
            .map_err(|e| format!("decode toggle_platform_registry request: {e}"))?;
        let resp = self.client.toggle_platform_registry_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_skill_registry_overrides_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::ListSkillRegistryOverridesRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_skill_registry_overrides request: {e}"))?;
        let resp = self.client.list_skill_registry_overrides_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    // ---- MarketService ----

    pub async fn list_market_skills_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::ListMarketSkillsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_market_skills request: {e}"))?;
        let resp = self.client.list_market_skills_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_market_mcp_servers_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::ListMarketMcpServersRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_market_mcp_servers request: {e}"))?;
        let resp = self.client.list_market_mcp_servers_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    // ---- RepoSkillService ----

    pub async fn list_repo_skills_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::ListRepoSkillsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_repo_skills request: {e}"))?;
        let resp = self.client.list_repo_skills_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn install_skill_from_market_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::InstallSkillFromMarketRequest::decode(request_bytes)
            .map_err(|e| format!("decode install_skill_from_market request: {e}"))?;
        let resp = self.client.install_skill_from_market_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn install_skill_from_github_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::InstallSkillFromGitHubRequest::decode(request_bytes)
            .map_err(|e| format!("decode install_skill_from_github request: {e}"))?;
        let resp = self.client.install_skill_from_github_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_skill_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::UpdateSkillRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_skill request: {e}"))?;
        let resp = self.client.update_skill_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn uninstall_skill_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::UninstallSkillRequest::decode(request_bytes)
            .map_err(|e| format!("decode uninstall_skill request: {e}"))?;
        let resp = self.client.uninstall_skill_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    // ---- RepoMcpService ----

    pub async fn list_repo_mcp_servers_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::ListRepoMcpServersRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_repo_mcp_servers request: {e}"))?;
        let resp = self.client.list_repo_mcp_servers_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn install_mcp_from_market_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::InstallMcpFromMarketRequest::decode(request_bytes)
            .map_err(|e| format!("decode install_mcp_from_market request: {e}"))?;
        let resp = self.client.install_mcp_from_market_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn install_custom_mcp_server_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::InstallCustomMcpServerRequest::decode(request_bytes)
            .map_err(|e| format!("decode install_custom_mcp_server request: {e}"))?;
        let resp = self.client.install_custom_mcp_server_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_mcp_server_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::UpdateMcpServerRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_mcp_server request: {e}"))?;
        let resp = self.client.update_mcp_server_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn uninstall_mcp_server_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = ext_proto::UninstallMcpServerRequest::decode(request_bytes)
            .map_err(|e| format!("decode uninstall_mcp_server request: {e}"))?;
        let resp = self.client.uninstall_mcp_server_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    // -------- Legacy REST (JSON wire) — preserved during dual-track --------

    pub async fn list_skill_registries(&self) -> Result<String, String> {
        let resp = self.client.list_skill_registries().await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn create_skill_registry(&self, json: &str) -> Result<String, String> {
        let req: CreateSkillRegistryRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client.create_skill_registry(&req).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn sync_skill_registry(&self, id: i64) -> Result<(), String> {
        self.client.sync_skill_registry(id).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn toggle_skill_registry(&self, id: i64, json: &str) -> Result<String, String> {
        let req: ToggleRegistryRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client.toggle_skill_registry(id, &req).await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn delete_skill_registry(&self, id: i64) -> Result<(), String> {
        self.client.delete_skill_registry(id).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn list_skill_registry_overrides(&self) -> Result<String, String> {
        let resp = self.client.list_skill_registry_overrides().await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn list_market_skills(
        &self, query: Option<String>, category: Option<String>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_market_skills(query.as_deref(), category.as_deref())
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn list_market_mcp_servers(
        &self, query: Option<String>, limit: Option<u32>, offset: Option<u32>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_market_mcp_servers(query.as_deref(), limit, offset)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn list_repo_skills(
        &self, repo_id: i64, scope: Option<String>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_repo_skills(repo_id, scope.as_deref())
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn install_skill_from_market(
        &self, repo_id: i64, json: &str,
    ) -> Result<String, String> {
        let req: InstallMarketSkillRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .install_skill_from_market(repo_id, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn install_skill_from_github(
        &self, repo_id: i64, json: &str,
    ) -> Result<String, String> {
        let req: InstallGithubSkillRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .install_skill_from_github(repo_id, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn update_skill(
        &self, repo_id: i64, install_id: i64, json: &str,
    ) -> Result<String, String> {
        let req: UpdateSkillInstallRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .update_skill_install(repo_id, install_id, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn uninstall_skill(&self, repo_id: i64, install_id: i64) -> Result<(), String> {
        self.client.uninstall_skill(repo_id, install_id).await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn list_repo_mcp_servers(
        &self, repo_id: i64, scope: Option<String>,
    ) -> Result<String, String> {
        let resp = self.client
            .list_repo_mcp_servers(repo_id, scope.as_deref())
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn install_mcp_from_market(
        &self, repo_id: i64, json: &str,
    ) -> Result<String, String> {
        let req: InstallMarketMcpRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .install_mcp_from_market(repo_id, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn install_custom_mcp_server(
        &self, repo_id: i64, json: &str,
    ) -> Result<String, String> {
        let req: InstallCustomMcpRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .install_custom_mcp_server(repo_id, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn update_mcp_server(
        &self, repo_id: i64, install_id: i64, json: &str,
    ) -> Result<String, String> {
        let req: UpdateMcpInstallRequest = serde_json::from_str(json).map_err(crate::wire)?;
        let resp = self.client
            .update_mcp_install(repo_id, install_id, &req)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }

    pub async fn uninstall_mcp_server(
        &self, repo_id: i64, install_id: i64,
    ) -> Result<(), String> {
        self.client
            .uninstall_mcp_server(repo_id, install_id)
            .await.map_err(crate::wire)?;
        Ok(())
    }

    pub async fn install_skill_from_upload(
        &self, repo_id: i64, file_data: Vec<u8>,
        file_name: &str, scope: Option<String>,
    ) -> Result<String, String> {
        let part = reqwest::multipart::Part::bytes(file_data).file_name(file_name.to_string());
        let mut form = reqwest::multipart::Form::new().part("file", part);
        if let Some(s) = scope { form = form.text("scope", s); }
        let endpoint = self.client.org_path(&format!("/repositories/{repo_id}/skills/install-from-upload"));
        let resp = self.client
            .post_multipart::<serde_json::Value>(&endpoint, form)
            .await.map_err(crate::wire)?;
        serde_json::to_string(&resp).map_err(crate::wire)
    }
}
