use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::proto_agent_v1 as agent_proto;
use agentsmesh_types::proto_pod_v1 as pod_proto;
use prost::Message;

pub struct AgentService {
    client: Arc<ApiClient>,
}

impl AgentService {
    pub fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    // -------- Connect-RPC (binary wire) — AgentPodSettingsService --------
    //
    // Each `*_connect` method takes prost-encoded bytes and returns
    // prost-encoded bytes (conventions §2.5). User-scoped — auth comes from
    // the Connect interceptor's TenantContext, no payload-derived org_slug.

    pub async fn get_agentpod_settings_connect(&self) -> Result<Vec<u8>, String> {
        tracing::debug!(target: "agent", "get agentpod settings");
        let resp = self.client.get_agentpod_settings_connect().await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_agentpod_settings_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::UpdateSettingsRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_agentpod_settings request: {e}"))?;
        tracing::info!(target: "agent", "update agentpod settings");
        let resp = self.client.update_agentpod_settings_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_agentpod_providers_connect(&self) -> Result<Vec<u8>, String> {
        tracing::debug!(target: "agent", "list agentpod providers");
        let resp = self.client.list_agentpod_providers_connect().await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_agentpod_provider_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::CreateProviderRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_agentpod_provider request: {e}"))?;
        tracing::info!(target: "agent", provider_type = %req.provider_type, "create agentpod provider");
        let resp = self.client.create_agentpod_provider_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_agentpod_provider_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::UpdateProviderRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_agentpod_provider request: {e}"))?;
        tracing::info!(target: "agent", provider_id = req.id, "update agentpod provider");
        let resp = self.client.update_agentpod_provider_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_agentpod_provider_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::DeleteProviderRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_agentpod_provider request: {e}"))?;
        tracing::info!(target: "agent", provider_id = req.id, "delete agentpod provider");
        let resp = self.client.delete_agentpod_provider_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn set_default_agentpod_provider_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::SetDefaultProviderRequest::decode(request_bytes)
            .map_err(|e| format!("decode set_default_agentpod_provider request: {e}"))?;
        tracing::info!(target: "agent", provider_id = req.id, "set default agentpod provider");
        let resp = self.client.set_default_agentpod_provider_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    // -------- Connect-RPC (binary wire) — AgentService + UserAgentConfigService --------
    //
    // Each `*_connect` method takes prost-encoded bytes and returns
    // prost-encoded bytes (conventions §2.5). AgentService is org-scoped
    // (request carries org_slug = 1); UserAgentConfigService is user-scoped
    // (auth interceptor populates UserID, no org_slug).

    pub async fn list_agents_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::ListAgentsRequest::decode(request_bytes)
            .map_err(|e| format!("decode list_agents request: {e}"))?;
        tracing::debug!(target: "agent", org_slug = %req.org_slug, "list agents");
        let resp = self.client.list_agents_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_agent_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::GetAgentRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_agent request: {e}"))?;
        tracing::debug!(target: "agent", org_slug = %req.org_slug, agent_slug = %req.agent_slug, "get agent");
        let resp = self.client.get_agent_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_agent_config_schema_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::GetAgentConfigSchemaRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_agent_config_schema request: {e}"))?;
        tracing::debug!(target: "agent", org_slug = %req.org_slug, agent_slug = %req.agent_slug, "get agent config schema");
        let resp = self.client.get_agent_config_schema_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_custom_agent_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::CreateCustomAgentRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_custom_agent request: {e}"))?;
        tracing::info!(target: "agent", org_slug = %req.org_slug, slug = %req.slug, "create custom agent");
        let resp = self.client.create_custom_agent_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_custom_agent_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::UpdateCustomAgentRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_custom_agent request: {e}"))?;
        tracing::info!(target: "agent", org_slug = %req.org_slug, agent_slug = %req.agent_slug, "update custom agent");
        let resp = self.client.update_custom_agent_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_custom_agent_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::DeleteCustomAgentRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_custom_agent request: {e}"))?;
        tracing::info!(target: "agent", org_slug = %req.org_slug, agent_slug = %req.agent_slug, "delete custom agent");
        let resp = self.client.delete_custom_agent_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_user_agent_configs_connect(&self) -> Result<Vec<u8>, String> {
        tracing::debug!(target: "agent", "list user agent configs");
        let resp = self.client.list_user_agent_configs_connect().await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn get_user_agent_config_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::GetUserAgentConfigRequest::decode(request_bytes)
            .map_err(|e| format!("decode get_user_agent_config request: {e}"))?;
        tracing::debug!(target: "agent", agent_slug = %req.agent_slug, "get user agent config");
        let resp = self.client.get_user_agent_config_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn set_user_agent_config_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::SetUserAgentConfigRequest::decode(request_bytes)
            .map_err(|e| format!("decode set_user_agent_config request: {e}"))?;
        tracing::info!(target: "agent", agent_slug = %req.agent_slug, "set user agent config");
        let resp = self.client.set_user_agent_config_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_user_agent_config_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = agent_proto::DeleteUserAgentConfigRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_user_agent_config request: {e}"))?;
        tracing::info!(target: "agent", agent_slug = %req.agent_slug, "delete user agent config");
        let resp = self.client.delete_user_agent_config_connect(&req).await.map_err(crate::wire)?;
        Ok(resp.encode_to_vec())
    }
}
