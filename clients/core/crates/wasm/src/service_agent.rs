use std::sync::Arc;

use agentsmesh_api_client::ApiClient;
use agentsmesh_types::*;
use agentsmesh_types::proto_pod_v1 as pod_proto;
use prost::Message;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmAgentService {
    client: Arc<ApiClient>,
}

#[wasm_bindgen]
impl WasmAgentService {
    pub(crate) fn new(client: Arc<ApiClient>) -> Self {
        Self { client }
    }

    pub async fn list_agents(&self) -> Result<String, String> {
        let resp = self.client.list_agents().await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn get_config_schema(&self, agent_slug: &str) -> Result<String, String> {
        let resp = self.client
            .get_agent_config_schema(agent_slug)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn list_user_configs(&self) -> Result<String, String> {
        let resp = self.client.list_user_agent_configs().await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn get_user_config(&self, agent_slug: &str) -> Result<String, String> {
        let resp = self.client
            .get_user_agent_config(agent_slug)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn set_user_config(
        &self, agent_slug: &str, json: &str,
    ) -> Result<String, String> {
        let req: SetUserAgentConfigRequest = serde_json::from_str(json).map_err(agentsmesh_services::wire)?;
        let resp = self.client
            .set_user_agent_config(agent_slug, &req)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn delete_user_config(&self, agent_slug: &str) -> Result<(), String> {
        self.client
            .delete_user_agent_config(agent_slug)
            .await.map_err(agentsmesh_services::wire)?;
        Ok(())
    }

    pub async fn get_agentpod_settings(&self) -> Result<String, String> {
        let resp = self.client.get_agentpod_settings().await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn update_agentpod_settings(&self, json: &str) -> Result<String, String> {
        let req: AgentPodSettings = serde_json::from_str(json).map_err(agentsmesh_services::wire)?;
        let resp = self.client
            .update_agentpod_settings(&req)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn list_providers(&self) -> Result<String, String> {
        let resp = self.client
            .list_agentpod_providers()
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn create_provider(&self, json: &str) -> Result<String, String> {
        let req: CreateAIProviderRequest = serde_json::from_str(json).map_err(agentsmesh_services::wire)?;
        let resp = self.client
            .create_agentpod_provider(&req)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn update_provider(&self, id: i64, json: &str) -> Result<String, String> {
        let req: UpdateAIProviderRequest = serde_json::from_str(json).map_err(agentsmesh_services::wire)?;
        let resp = self.client
            .update_agentpod_provider(id, &req)
            .await.map_err(agentsmesh_services::wire)?;
        serde_json::to_string(&resp).map_err(agentsmesh_services::wire)
    }

    pub async fn delete_provider(&self, id: i64) -> Result<(), String> {
        self.client
            .delete_agentpod_provider(id)
            .await.map_err(agentsmesh_services::wire)?;
        Ok(())
    }

    pub async fn set_default_provider(&self, id: i64) -> Result<(), String> {
        self.client
            .set_default_agentpod_provider(id)
            .await.map_err(agentsmesh_services::wire)?;
        Ok(())
    }

    // -------- Connect-RPC (binary wire) — AgentPodSettingsService --------

    pub async fn get_agentpod_settings_connect(&self) -> Result<Vec<u8>, String> {
        let resp = self.client.get_agentpod_settings_connect().await.map_err(agentsmesh_services::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_agentpod_settings_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::UpdateSettingsRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_agentpod_settings request: {e}"))?;
        let resp = self.client.update_agentpod_settings_connect(&req).await.map_err(agentsmesh_services::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn list_agentpod_providers_connect(&self) -> Result<Vec<u8>, String> {
        let resp = self.client.list_agentpod_providers_connect().await.map_err(agentsmesh_services::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn create_agentpod_provider_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::CreateProviderRequest::decode(request_bytes)
            .map_err(|e| format!("decode create_agentpod_provider request: {e}"))?;
        let resp = self.client.create_agentpod_provider_connect(&req).await.map_err(agentsmesh_services::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn update_agentpod_provider_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::UpdateProviderRequest::decode(request_bytes)
            .map_err(|e| format!("decode update_agentpod_provider request: {e}"))?;
        let resp = self.client.update_agentpod_provider_connect(&req).await.map_err(agentsmesh_services::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn delete_agentpod_provider_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::DeleteProviderRequest::decode(request_bytes)
            .map_err(|e| format!("decode delete_agentpod_provider request: {e}"))?;
        let resp = self.client.delete_agentpod_provider_connect(&req).await.map_err(agentsmesh_services::wire)?;
        Ok(resp.encode_to_vec())
    }

    pub async fn set_default_agentpod_provider_connect(&self, request_bytes: &[u8]) -> Result<Vec<u8>, String> {
        let req = pod_proto::SetDefaultProviderRequest::decode(request_bytes)
            .map_err(|e| format!("decode set_default_agentpod_provider request: {e}"))?;
        let resp = self.client.set_default_agentpod_provider_connect(&req).await.map_err(agentsmesh_services::wire)?;
        Ok(resp.encode_to_vec())
    }
}
