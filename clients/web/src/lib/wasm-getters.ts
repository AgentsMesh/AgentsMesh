import {
  WasmApiClient, WasmAuthManager,
  WasmPodService, WasmTicketService, WasmChannelService,
  WasmRunnerService, WasmLoopService, WasmAutopilotService,
  WasmMeshService, WasmBillingService, WasmRepositoryService,
  WasmExtensionService, WasmInvitationService, WasmApiKeyService,
  WasmGrantService,
  WasmBindingService, WasmNotificationService,
  WasmPromoCodeService, WasmTokenUsageService, WasmSSOService,
  WasmUserApiService, WasmUserCredentialService, WasmEnvBundleService, WasmOrgApiService,
  WasmAgentService, WasmTicketRelationsService, WasmFileService,
  WasmSupportTicketService, WasmAuthConnectService,
  WasmBlockstoreService,
  WasmRelayManager,
} from "agentsmesh-wasm";
import { registerServiceProvider } from "@agentsmesh/service-runtime";

// AuthManager + ApiClient share the same token store (Plan I6).
// Caller constructs AuthManager first, passes it to ApiClient, then both here.
// org_slug is read from AuthManager's PersistedSession on every request — no
// renderer-side `set_org_slug()` needed.
//
// As of the Rust SSOT refactor (Phase 2), all WasmXxxState instances are
// VIEWS over the SINGLE `AppRuntime.state` owned by the WasmApiClient.
// Events delivered via the realtime stream (handled in Rust through
// `EventSubscriptionManager` → `AppState.dispatch`) are immediately
// visible to every selector here. Do NOT `new WasmPodState()` etc. —
// those construct disjoint state and silently drop realtime events.
export function registerAll(client: WasmApiClient, authManager: WasmAuthManager) {
  registerServiceProvider({
    apiClient: client,
    authManager,
    podState: client.get_pod_state(),
    podService: client.create_pod_service(),
    ticketService: client.create_ticket_service(),
    channelService: client.create_channel_service(),
    runnerService: client.create_runner_service(),
    loopService: client.create_loop_service(),
    autopilotService: client.create_autopilot_service(),
    meshService: client.create_mesh_service(),
    billingService: client.create_billing_service(),
    repositoryService: client.create_repository_service(),
    extensionService: client.create_extension_service(),
    invitationService: client.create_invitation_service(),
    grantService: client.create_grant_service(),
    apiKeyService: client.create_apikey_service(),
    bindingService: client.create_binding_service(),
    notificationService: client.create_notification_service(),
    promoCodeService: client.create_promocode_service(),
    tokenUsageService: client.create_token_usage_service(),
    ssoService: client.create_sso_service(),
    userApiService: client.create_user_api_service(),
    userCredentialService: client.create_user_credential_service(),
    envBundleService: client.create_env_bundle_service(),
    orgApiService: client.create_org_api_service(),
    agentService: client.create_agent_service(),
    ticketRelationsService: client.create_ticket_relations_service(),
    fileService: client.create_file_service(),
    supportTicketService: client.create_support_ticket_service(),
    authConnectService: client.create_auth_connect_service(),
    blockstoreService: client.create_blockstore_service(),
    runnerState: client.get_runner_state(),
    meshState: client.get_mesh_state(),
    ticketState: client.get_ticket_state(),
    channelState: client.get_channel_state(),
    loopState: client.get_loop_state(),
    acpManager: client.get_acp_manager(),
    loopalManager: client.get_loopal_manager(),
    repoState: client.get_repo_state(),
    autopilotState: client.get_autopilot_state(),
    relayManager: new WasmRelayManager(),
  });
}

// Re-export everything from service-runtime for backward compatibility
export {
  NOOP_PROXY, isServiceReady as isWasmReady,
  registerServiceProvider, parseWasmAny,
  getApiClient, getAuthManager, getPodState, getPodService,
  getTicketService, getChannelService, getRunnerService,
  getLoopService, getAutopilotService, getMeshService,
  getBillingService, getRepositoryService, getExtensionService,
  getInvitationService, getApiKeyService, getBindingService,
  getGrantService,
  getNotificationService, getPromoCodeService,
  getTokenUsageService, getSSOService, getUserApiService,
  getUserCredentialService, getEnvBundleService, getOrgApiService, getAgentService,
  getTicketRelationsService, getFileService, getSupportTicketService,
  getAuthConnectService, getRunnerState, getMeshState, getTicketState,
  getChannelState, getLoopState, getAcpManager, getLoopalManager,
  getRepoState,
  getAutopilotState, getRelayManager, getBlockstoreService,
} from "@agentsmesh/service-runtime";

