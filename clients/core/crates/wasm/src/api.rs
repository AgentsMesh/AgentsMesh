use std::sync::Arc;

use agentsmesh_api_client::{ApiClient, AuthTokenStore};
use agentsmesh_events::{EventSubscriptionManager, EventSubscriptionManagerOptions};
use agentsmesh_state::app_state::AppRuntime;
use agentsmesh_transport::runtime::PlatformRuntime;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct WasmApiClient {
    client: Arc<ApiClient>,
    base_url: String,
    /// Singleton AppRuntime for this client. Created once at construction
    /// time and shared by all services + the events manager. The events
    /// manager's dispatch hook is wired into `AppRuntime.state.dispatch`
    /// at this point, so any event delivered through the connection_loop
    /// updates the same `AppState` that services + selectors read from.
    runtime: Arc<AppRuntime>,
}

#[wasm_bindgen]
impl WasmApiClient {
    #[wasm_bindgen(constructor)]
    pub fn new(base_url: String, auth: &crate::auth::WasmAuthManager) -> Self {
        let store: Arc<dyn AuthTokenStore> = auth.token_store_arc();
        let client = Arc::new(ApiClient::new(base_url.clone(), store));
        let events = Arc::new(EventSubscriptionManager::with_runtime(
            PlatformRuntime,
            client.clone(),
            EventSubscriptionManagerOptions::default(),
        ));
        let runtime = AppRuntime::new(events);
        Self { client, base_url, runtime }
    }

    #[wasm_bindgen(getter)]
    pub fn base_url(&self) -> String {
        self.base_url.clone()
    }

    // ── AppRuntime state views ──
    // These return per-domain view structs over the SINGLE shared
    // `AppState`. All views observe the same writes — including writes
    // from `EventSubscriptionManager.dispatch_event` (via the dispatch
    // hook installed in `AppRuntime::new`). Stable wasm-bindgen API,
    // safe to call from JS many times.

    pub fn get_pod_state(&self) -> crate::state_pod::WasmPodState {
        crate::state_pod::WasmPodState::from_runtime(self.runtime.state.clone())
    }

    pub fn get_channel_state(&self) -> crate::state_channel::WasmChannelState {
        crate::state_channel::WasmChannelState::from_runtime(self.runtime.state.clone())
    }

    pub fn get_ticket_state(&self) -> crate::state_ticket::WasmTicketState {
        crate::state_ticket::WasmTicketState::from_runtime(self.runtime.state.clone())
    }

    pub fn get_runner_state(&self) -> crate::state_runner::WasmRunnerState {
        crate::state_runner::WasmRunnerState::from_runtime(self.runtime.state.clone())
    }

    pub fn get_loop_state(&self) -> crate::state_loop::WasmLoopState {
        crate::state_loop::WasmLoopState::from_runtime(self.runtime.state.clone())
    }

    pub fn get_mesh_state(&self) -> crate::state_mesh::WasmMeshState {
        crate::state_mesh::WasmMeshState::from_runtime(self.runtime.state.clone())
    }

    pub fn get_autopilot_state(&self) -> crate::state_autopilot::WasmAutopilotState {
        crate::state_autopilot::WasmAutopilotState::from_runtime(self.runtime.state.clone())
    }

    pub fn get_repo_state(&self) -> crate::state_repo::WasmRepoState {
        crate::state_repo::WasmRepoState::from_runtime(self.runtime.state.clone())
    }

    pub fn get_acp_manager(&self) -> crate::state_acp::WasmAcpSessionManager {
        crate::state_acp::WasmAcpSessionManager::from_runtime(self.runtime.state.clone())
    }

    pub fn get_loopal_manager(&self) -> crate::state_loopal::WasmLoopalManager {
        crate::state_loopal::WasmLoopalManager::from_runtime(self.runtime.state.clone())
    }

    // ── Pending side-effect drains ──
    // Rust SSOT dispatch queues side-effects (toast, browser notification,
    // refetch keys) into AppState.pending_*. JS drains these per tick;
    // each drain is atomic on the Rust side (take + clear).

    pub fn take_pending_toasts(&self) -> String {
        let toasts = self.runtime.state.write().take_pending_toasts();
        serde_json::to_string(&toasts).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn take_pending_browser_notifications(&self) -> String {
        let notifs = self.runtime.state.write().take_pending_browser_notifications();
        serde_json::to_string(&notifs).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn take_pending_refetch_ticket_slugs(&self) -> String {
        let slugs = self.runtime.state.write().take_pending_refetch_ticket_slugs();
        serde_json::to_string(&slugs).unwrap_or_else(|_| "[]".to_string())
    }

    pub fn take_pending_refetch_pod_keys(&self) -> String {
        let keys = self.runtime.state.write().take_pending_refetch_pod_keys();
        serde_json::to_string(&keys).unwrap_or_else(|_| "[]".to_string())
    }

    /// Tick counter — increments after every event dispatched to AppState.
    /// React selectors use this as the snapshot for `useSyncExternalStore`.
    pub fn tick(&self) -> f64 {
        self.runtime.events.tick() as f64
    }

    /// Clear all org-scoped state. Used on org switch; preserves the
    /// realtime connection + dispatch hook registration.
    pub fn reset_for_org_switch(&self) {
        self.runtime.state.write().reset_for_org_switch();
    }

    pub fn create_pod_service(&self) -> crate::service_pod::WasmPodService {
        crate::service_pod::WasmPodService::new(self.client.clone())
    }

    /// Create a WasmEventsManager backed by this client's ApiClient.
    /// Replaces the legacy `new WasmEventsManager(ws_url)` — token, base
    /// URL, and org slug now flow through the shared ApiClient instead.
    ///
    /// Returns the SAME `EventSubscriptionManager` instance that
    /// `AppRuntime` was wired against during construction, so the
    /// dispatch hook fires on every event delivered through this
    /// manager's subscriptions.
    pub fn create_events_manager(&self) -> crate::events_manager::WasmEventsManager {
        crate::events_manager::WasmEventsManager::from_shared(self.runtime.events.clone())
    }

    pub fn create_ticket_service(&self) -> crate::service_ticket::WasmTicketService {
        crate::service_ticket::WasmTicketService::new(self.client.clone())
    }

    pub fn create_channel_service(&self) -> crate::service_channel::WasmChannelService {
        crate::service_channel::WasmChannelService::new(self.client.clone())
    }

    pub fn create_runner_service(&self) -> crate::service_runner::WasmRunnerService {
        crate::service_runner::WasmRunnerService::new(self.client.clone())
    }

    pub fn create_loop_service(&self) -> crate::service_loop::WasmLoopService {
        crate::service_loop::WasmLoopService::new(self.client.clone())
    }

    pub fn create_autopilot_service(&self) -> crate::service_autopilot::WasmAutopilotService {
        crate::service_autopilot::WasmAutopilotService::new(self.client.clone())
    }

    pub fn create_mesh_service(&self) -> crate::service_mesh::WasmMeshService {
        crate::service_mesh::WasmMeshService::new(self.client.clone())
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

    pub fn create_env_bundle_service(
        &self,
    ) -> crate::service_env_bundle::WasmEnvBundleService {
        crate::service_env_bundle::WasmEnvBundleService::new(self.client.clone())
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
