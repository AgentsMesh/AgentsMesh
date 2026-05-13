use std::sync::Arc;

use agentsmesh_api_client::{ApiClient, AuthTokenStore};
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmApiClient {
    client: Arc<ApiClient>,
    base_url: String,
}

#[wasm_bindgen]
impl WasmApiClient {
    /// SSOT-aware constructor: ApiClient is wired to AuthManager's token
    /// store so token writes (login / refresh / bootstrap) reach API calls
    /// without TS-side `set_token()` glue. There is no other constructor —
    /// every WasmApiClient is paired with an AuthManager.
    #[wasm_bindgen(constructor)]
    pub fn new(base_url: String, auth: &crate::auth::WasmAuthManager) -> Self {
        let store: Arc<dyn AuthTokenStore> = auth.token_store_arc();
        let client = Arc::new(ApiClient::new(base_url.clone(), store));
        Self { client, base_url }
    }

    pub fn org_path(&self, path: &str) -> String {
        self.client.org_path(path)
    }

    #[wasm_bindgen(getter)]
    pub fn base_url(&self) -> String {
        self.base_url.clone()
    }

    pub async fn get(&self, endpoint: String) -> Result<String, String> {
        self.client
            .get::<serde_json::Value>(&endpoint)
            .await
            .map(|v| serde_json::to_string(&v).unwrap_or_default())
            .map_err(agentsmesh_services::wire)
    }

    pub async fn post(&self, endpoint: String, body: String) -> Result<String, String> {
        let val: serde_json::Value =
            serde_json::from_str(&body).map_err(agentsmesh_services::wire)?;
        self.client
            .post::<serde_json::Value>(&endpoint, &val)
            .await
            .map(|v| serde_json::to_string(&v).unwrap_or_default())
            .map_err(agentsmesh_services::wire)
    }

    pub async fn put(&self, endpoint: String, body: String) -> Result<String, String> {
        let val: serde_json::Value =
            serde_json::from_str(&body).map_err(agentsmesh_services::wire)?;
        self.client
            .put::<serde_json::Value>(&endpoint, &val)
            .await
            .map(|v| serde_json::to_string(&v).unwrap_or_default())
            .map_err(agentsmesh_services::wire)
    }

    pub async fn delete(&self, endpoint: String) -> Result<String, String> {
        self.client
            .delete::<serde_json::Value>(&endpoint)
            .await
            .map(|v| serde_json::to_string(&v).unwrap_or_default())
            .map_err(agentsmesh_services::wire)
    }

    pub async fn patch(&self, endpoint: String, body: String) -> Result<String, String> {
        let val: serde_json::Value =
            serde_json::from_str(&body).map_err(agentsmesh_services::wire)?;
        self.client
            .patch::<serde_json::Value>(&endpoint, &val)
            .await
            .map(|v| serde_json::to_string(&v).unwrap_or_default())
            .map_err(agentsmesh_services::wire)
    }

    pub async fn public_get(&self, endpoint: String) -> Result<String, String> {
        self.client
            .public_get::<serde_json::Value>(&endpoint)
            .await
            .map(|v| serde_json::to_string(&v).unwrap_or_default())
            .map_err(agentsmesh_services::wire)
    }

    pub async fn public_post(
        &self,
        endpoint: String,
        body: String,
    ) -> Result<String, String> {
        let val: serde_json::Value =
            serde_json::from_str(&body).map_err(agentsmesh_services::wire)?;
        self.client
            .public_post::<serde_json::Value>(&endpoint, &val)
            .await
            .map(|v| serde_json::to_string(&v).unwrap_or_default())
            .map_err(agentsmesh_services::wire)
    }

    /// Create a WasmPodService that shares this client's ApiClient and auth.
    pub fn create_pod_service(&self) -> crate::service_pod::WasmPodService {
        let state = agentsmesh_state::pod_state::PodState::with_storage(crate::new_memory_backend());
        crate::service_pod::WasmPodService::new(self.client.clone(), state)
    }

    pub fn create_ticket_service(&self) -> crate::service_ticket::WasmTicketService {
        let state = agentsmesh_state::ticket_state::TicketState::with_storage(crate::new_memory_backend());
        crate::service_ticket::WasmTicketService::new(self.client.clone(), state)
    }

    pub fn create_channel_service(&self) -> crate::service_channel::WasmChannelService {
        let state = agentsmesh_state::channel_state::ChannelState::with_storage(crate::new_memory_backend());
        crate::service_channel::WasmChannelService::new(self.client.clone(), state)
    }

    pub fn create_runner_service(&self) -> crate::service_runner::WasmRunnerService {
        let state = agentsmesh_state::runner_state::RunnerState::with_storage(crate::new_memory_backend());
        crate::service_runner::WasmRunnerService::new(self.client.clone(), state)
    }

    pub fn create_loop_service(&self) -> crate::service_loop::WasmLoopService {
        let state = agentsmesh_state::loop_state::LoopState::with_storage(crate::new_memory_backend());
        crate::service_loop::WasmLoopService::new(self.client.clone(), state)
    }

    pub fn create_autopilot_service(&self) -> crate::service_autopilot::WasmAutopilotService {
        let state = agentsmesh_state::autopilot_state::AutopilotState::new();
        crate::service_autopilot::WasmAutopilotService::new(self.client.clone(), state)
    }

    pub fn create_mesh_service(&self) -> crate::service_mesh::WasmMeshService {
        let state = agentsmesh_state::mesh_state::MeshState::new();
        crate::service_mesh::WasmMeshService::new(self.client.clone(), state)
    }

    pub fn create_blockstore_service(&self) -> crate::service_blockstore::WasmBlockstoreService {
        let state = agentsmesh_state::blockstore_state::BlockstoreState::new();
        crate::service_blockstore::WasmBlockstoreService::new(self.client.clone(), state)
    }

    pub fn create_billing_service(&self) -> crate::service_billing::WasmBillingService {
        crate::service_billing::WasmBillingService::new(self.client.clone())
    }

    pub fn create_repository_service(&self) -> crate::service_repository::WasmRepositoryService {
        crate::service_repository::WasmRepositoryService::new(self.client.clone())
    }

    pub fn create_extension_service(&self) -> crate::service_extension::WasmExtensionService {
        crate::service_extension::WasmExtensionService::new(self.client.clone())
    }

    pub fn create_invitation_service(&self) -> crate::service_invitation::WasmInvitationService {
        crate::service_invitation::WasmInvitationService::new(self.client.clone())
    }

    pub fn create_grant_service(&self) -> crate::service_grant::WasmGrantService {
        crate::service_grant::WasmGrantService::new(self.client.clone())
    }

    pub fn create_apikey_service(&self) -> crate::service_apikey::WasmApiKeyService {
        crate::service_apikey::WasmApiKeyService::new(self.client.clone())
    }

    pub fn create_binding_service(&self) -> crate::service_binding::WasmBindingService {
        crate::service_binding::WasmBindingService::new(self.client.clone())
    }

    pub fn create_message_service(&self) -> crate::service_message::WasmMessageService {
        crate::service_message::WasmMessageService::new(self.client.clone())
    }

    pub fn create_notification_service(
        &self,
    ) -> crate::service_notification::WasmNotificationService {
        crate::service_notification::WasmNotificationService::new(self.client.clone())
    }

    pub fn create_promocode_service(&self) -> crate::service_promocode::WasmPromoCodeService {
        crate::service_promocode::WasmPromoCodeService::new(self.client.clone())
    }

    pub fn create_token_usage_service(
        &self,
    ) -> crate::service_token_usage::WasmTokenUsageService {
        crate::service_token_usage::WasmTokenUsageService::new(self.client.clone())
    }

    pub fn create_sso_service(&self) -> crate::service_sso::WasmSSOService {
        crate::service_sso::WasmSSOService::new(self.client.clone())
    }

    pub fn create_user_api_service(&self) -> crate::service_user::WasmUserApiService {
        crate::service_user::WasmUserApiService::new(self.client.clone())
    }

    pub fn create_user_credential_service(
        &self,
    ) -> crate::service_user_credential::WasmUserCredentialService {
        crate::service_user_credential::WasmUserCredentialService::new(self.client.clone())
    }

    pub fn create_org_api_service(&self) -> crate::service_org::WasmOrgApiService {
        crate::service_org::WasmOrgApiService::new(self.client.clone())
    }

    pub fn create_agent_service(&self) -> crate::service_agent::WasmAgentService {
        crate::service_agent::WasmAgentService::new(self.client.clone())
    }

    pub fn create_ticket_relations_service(
        &self,
    ) -> crate::service_ticket_relations::WasmTicketRelationsService {
        crate::service_ticket_relations::WasmTicketRelationsService::new(self.client.clone())
    }

    pub fn create_file_service(&self) -> crate::service_file::WasmFileService {
        crate::service_file::WasmFileService::new(self.client.clone())
    }

    pub fn create_support_ticket_service(
        &self,
    ) -> crate::service_support_ticket::WasmSupportTicketService {
        crate::service_support_ticket::WasmSupportTicketService::new(self.client.clone())
    }

    pub fn create_auth_connect_service(
        &self,
    ) -> crate::service_auth_connect::WasmAuthConnectService {
        crate::service_auth_connect::WasmAuthConnectService::new(self.client.clone())
    }
}
