use crate::ApiClient;
use crate::connect_call::connect_call;
use crate::error::ApiError;
use agentsmesh_types::*;
use agentsmesh_types::proto_extension_v1 as ext_proto;

// =============================================================================
// Connect-RPC (binary wire). See proto-naming-conventions.md §2.5.
// =============================================================================
//
// These methods call the Connect handlers in
// backend/internal/api/connect/extension/. Procedure paths derive from
// `proto.extension.v1.SkillRegistryService.<Method>` (conventions §12).

impl ApiClient {
    pub async fn list_skill_registries_connect(
        &self,
        req: &ext_proto::ListSkillRegistriesRequest,
    ) -> Result<ext_proto::ListSkillRegistriesResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.SkillRegistryService/ListSkillRegistries",
            req,
        )
        .await
    }

    pub async fn create_skill_registry_connect(
        &self,
        req: &ext_proto::CreateSkillRegistryRequest,
    ) -> Result<ext_proto::SkillRegistry, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.SkillRegistryService/CreateSkillRegistry",
            req,
        )
        .await
    }

    pub async fn sync_skill_registry_connect(
        &self,
        req: &ext_proto::SyncSkillRegistryRequest,
    ) -> Result<ext_proto::SkillRegistry, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.SkillRegistryService/SyncSkillRegistry",
            req,
        )
        .await
    }

    pub async fn delete_skill_registry_connect(
        &self,
        req: &ext_proto::DeleteSkillRegistryRequest,
    ) -> Result<ext_proto::DeleteSkillRegistryResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.SkillRegistryService/DeleteSkillRegistry",
            req,
        )
        .await
    }

    pub async fn toggle_platform_registry_connect(
        &self,
        req: &ext_proto::TogglePlatformRegistryRequest,
    ) -> Result<ext_proto::TogglePlatformRegistryResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.SkillRegistryService/TogglePlatformRegistry",
            req,
        )
        .await
    }

    pub async fn list_skill_registry_overrides_connect(
        &self,
        req: &ext_proto::ListSkillRegistryOverridesRequest,
    ) -> Result<ext_proto::ListSkillRegistryOverridesResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.SkillRegistryService/ListSkillRegistryOverrides",
            req,
        )
        .await
    }

    // ---- MarketService ----

    pub async fn list_market_skills_connect(
        &self,
        req: &ext_proto::ListMarketSkillsRequest,
    ) -> Result<ext_proto::ListMarketSkillsResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.MarketService/ListMarketSkills",
            req,
        )
        .await
    }

    pub async fn list_market_mcp_servers_connect(
        &self,
        req: &ext_proto::ListMarketMcpServersRequest,
    ) -> Result<ext_proto::ListMarketMcpServersResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.MarketService/ListMarketMcpServers",
            req,
        )
        .await
    }

    // ---- RepoSkillService ----

    pub async fn list_repo_skills_connect(
        &self,
        req: &ext_proto::ListRepoSkillsRequest,
    ) -> Result<ext_proto::ListRepoSkillsResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoSkillService/ListRepoSkills",
            req,
        )
        .await
    }

    pub async fn install_skill_from_market_connect(
        &self,
        req: &ext_proto::InstallSkillFromMarketRequest,
    ) -> Result<ext_proto::InstalledSkill, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoSkillService/InstallSkillFromMarket",
            req,
        )
        .await
    }

    pub async fn install_skill_from_github_connect(
        &self,
        req: &ext_proto::InstallSkillFromGitHubRequest,
    ) -> Result<ext_proto::InstalledSkill, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoSkillService/InstallSkillFromGitHub",
            req,
        )
        .await
    }

    pub async fn update_skill_connect(
        &self,
        req: &ext_proto::UpdateSkillRequest,
    ) -> Result<ext_proto::InstalledSkill, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoSkillService/UpdateSkill",
            req,
        )
        .await
    }

    pub async fn uninstall_skill_connect(
        &self,
        req: &ext_proto::UninstallSkillRequest,
    ) -> Result<ext_proto::UninstallSkillResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoSkillService/UninstallSkill",
            req,
        )
        .await
    }

    // ---- RepoMcpService ----

    pub async fn list_repo_mcp_servers_connect(
        &self,
        req: &ext_proto::ListRepoMcpServersRequest,
    ) -> Result<ext_proto::ListRepoMcpServersResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoMcpService/ListRepoMcpServers",
            req,
        )
        .await
    }

    pub async fn install_mcp_from_market_connect(
        &self,
        req: &ext_proto::InstallMcpFromMarketRequest,
    ) -> Result<ext_proto::InstalledMcpServer, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoMcpService/InstallMcpFromMarket",
            req,
        )
        .await
    }

    pub async fn install_custom_mcp_server_connect(
        &self,
        req: &ext_proto::InstallCustomMcpServerRequest,
    ) -> Result<ext_proto::InstalledMcpServer, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoMcpService/InstallCustomMcpServer",
            req,
        )
        .await
    }

    pub async fn update_mcp_server_connect(
        &self,
        req: &ext_proto::UpdateMcpServerRequest,
    ) -> Result<ext_proto::InstalledMcpServer, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoMcpService/UpdateMcpServer",
            req,
        )
        .await
    }

    pub async fn uninstall_mcp_server_connect(
        &self,
        req: &ext_proto::UninstallMcpServerRequest,
    ) -> Result<ext_proto::UninstallMcpServerResponse, ApiError> {
        connect_call(
            self,
            "/proto.extension.v1.RepoMcpService/UninstallMcpServer",
            req,
        )
        .await
    }
}

// =============================================================================
// Legacy REST methods — preserved for dual-track migration.
// =============================================================================

impl ApiClient {
    pub async fn list_skill_registries(&self) -> Result<SkillRegistryListResponse, ApiError> {
        self.get(&self.org_path("/skill-registries")).await
    }

    pub async fn create_skill_registry(
        &self,
        data: &CreateSkillRegistryRequest,
    ) -> Result<SkillRegistry, ApiError> {
        self.post_resource(&self.org_path("/skill-registries"), data, "registry").await
    }

    pub async fn sync_skill_registry(
        &self,
        id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.post(
            &self.org_path(&format!("/skill-registries/{id}/sync")),
            &serde_json::json!({}),
        )
        .await
    }

    pub async fn toggle_skill_registry(
        &self,
        id: i64,
        data: &ToggleRegistryRequest,
    ) -> Result<SkillRegistry, ApiError> {
        self.put(
            &self.org_path(&format!("/skill-registries/{id}/toggle")),
            data,
        )
        .await
    }

    pub async fn delete_skill_registry(&self, id: i64) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!("/skill-registries/{id}")))
            .await
    }

    pub async fn list_skill_registry_overrides(
        &self,
    ) -> Result<SkillRegistryOverrideListResponse, ApiError> {
        self.get(&self.org_path("/skill-registry-overrides"))
            .await
    }

    pub async fn list_market_skills(
        &self,
        query: Option<&str>,
        category: Option<&str>,
    ) -> Result<MarketSkillListResponse, ApiError> {
        let mut path = self.org_path("/market/skills");
        let mut params = Vec::new();
        if let Some(q) = query {
            params.push(format!("q={q}"));
        }
        if let Some(c) = category {
            params.push(format!("category={c}"));
        }
        if !params.is_empty() {
            path = format!("{path}?{}", params.join("&"));
        }
        self.get(&path).await
    }

    pub async fn list_market_mcp_servers(
        &self,
        query: Option<&str>,
        limit: Option<u32>,
        offset: Option<u32>,
    ) -> Result<MarketMcpServerListResponse, ApiError> {
        let mut path = self.org_path("/market/mcp-servers");
        let mut params = Vec::new();
        if let Some(q) = query {
            params.push(format!("q={q}"));
        }
        if let Some(l) = limit {
            params.push(format!("limit={l}"));
        }
        if let Some(o) = offset {
            params.push(format!("offset={o}"));
        }
        if !params.is_empty() {
            path = format!("{path}?{}", params.join("&"));
        }
        self.get(&path).await
    }

    pub async fn list_repo_skills(
        &self,
        repo_id: i64,
        scope: Option<&str>,
    ) -> Result<RepoSkillInstallListResponse, ApiError> {
        let mut path = self.org_path(&format!("/repositories/{repo_id}/skills"));
        if let Some(s) = scope {
            path = format!("{path}?scope={s}");
        }
        self.get(&path).await
    }

    pub async fn install_skill_from_market(
        &self,
        repo_id: i64,
        data: &InstallMarketSkillRequest,
    ) -> Result<RepoSkillInstall, ApiError> {
        self.post_resource(
            &self.org_path(&format!("/repositories/{repo_id}/skills/install-from-market")),
            data, "skill",
        ).await
    }

    pub async fn install_skill_from_github(
        &self,
        repo_id: i64,
        data: &InstallGithubSkillRequest,
    ) -> Result<RepoSkillInstall, ApiError> {
        self.post_resource(
            &self.org_path(&format!("/repositories/{repo_id}/skills/install-from-github")),
            data, "skill",
        ).await
    }

    pub async fn update_skill_install(
        &self,
        repo_id: i64,
        install_id: i64,
        data: &UpdateSkillInstallRequest,
    ) -> Result<RepoSkillInstall, ApiError> {
        self.put_resource(
            &self.org_path(&format!("/repositories/{repo_id}/skills/{install_id}")),
            data, "skill",
        ).await
    }

    pub async fn uninstall_skill(
        &self,
        repo_id: i64,
        install_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!(
            "/repositories/{repo_id}/skills/{install_id}"
        )))
        .await
    }

    pub async fn list_repo_mcp_servers(
        &self,
        repo_id: i64,
        scope: Option<&str>,
    ) -> Result<RepoMcpServerInstallListResponse, ApiError> {
        let mut path = self.org_path(&format!("/repositories/{repo_id}/mcp-servers"));
        if let Some(s) = scope {
            path = format!("{path}?scope={s}");
        }
        self.get(&path).await
    }

    pub async fn install_mcp_from_market(
        &self,
        repo_id: i64,
        data: &InstallMarketMcpRequest,
    ) -> Result<RepoMcpServerInstall, ApiError> {
        self.post_resource(
            &self.org_path(&format!("/repositories/{repo_id}/mcp-servers/install-from-market")),
            data, "mcp_server",
        ).await
    }

    pub async fn install_custom_mcp_server(
        &self,
        repo_id: i64,
        data: &InstallCustomMcpRequest,
    ) -> Result<RepoMcpServerInstall, ApiError> {
        self.post_resource(
            &self.org_path(&format!("/repositories/{repo_id}/mcp-servers/install-custom")),
            data, "mcp_server",
        ).await
    }

    pub async fn update_mcp_install(
        &self,
        repo_id: i64,
        install_id: i64,
        data: &UpdateMcpInstallRequest,
    ) -> Result<RepoMcpServerInstall, ApiError> {
        self.put_resource(
            &self.org_path(&format!("/repositories/{repo_id}/mcp-servers/{install_id}")),
            data, "mcp_server",
        ).await
    }

    pub async fn uninstall_mcp_server(
        &self,
        repo_id: i64,
        install_id: i64,
    ) -> Result<EmptyResponse, ApiError> {
        self.delete(&self.org_path(&format!(
            "/repositories/{repo_id}/mcp-servers/{install_id}"
        )))
        .await
    }
}
