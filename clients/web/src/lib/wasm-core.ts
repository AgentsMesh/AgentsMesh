import initWasm, {
  version, WasmApiClient, WasmAuthManager, WasmEventsManager, WasmWebSocket,
  relay_encode_input as _rei, relay_decode_message as _rdm,
  relay_encode_resize as _rer, relay_encode_ping as _rep,
  relay_encode_control as _rec, relay_encode_resync as _rers,
  relay_encode_acp_command as _reac,
} from "agentsmesh-wasm";
import { markServiceReady, setPlatformInit } from "@agentsmesh/service-runtime";
import { getApiBaseUrl } from "./env";
import { activateWasmRelayBackend } from "@/stores/relayBackend";
import { activateWasmEventsBackend } from "@/lib/realtime/eventsBackend";
import { registerAll } from "./wasm-getters";

async function doWasmInit(): Promise<void> {
  await initWasm();
  let baseUrl = getApiBaseUrl();
  if (!baseUrl && typeof window !== "undefined") baseUrl = window.location.origin;
  // AuthManager owns the persisted token store; ApiClient borrows it.
  // Plan I6 (single source of truth) — never two parallel auth instances.
  const authManager = new WasmAuthManager(baseUrl);
  const apiClient = new WasmApiClient(baseUrl, authManager);
  registerAll(apiClient, authManager);
  activateWasmRelayBackend(WasmWebSocket);
  activateWasmEventsBackend(WasmWebSocket);
  markServiceReady();
  console.log("[WASM Core] Initialized, version:", version());
}

setPlatformInit(doWasmInit);

export { ensurePlatformReady as initWasmCore } from "@agentsmesh/service-runtime";
export { isServiceReady as isWasmReady, NOOP_PROXY, parseWasmAny } from "@agentsmesh/service-runtime";

export { WasmEventsManager, WasmWebSocket };
export { _rei as relay_encode_input, _rdm as relay_decode_message };
export { _rer as relay_encode_resize, _rep as relay_encode_ping };
export { _rec as relay_encode_control, _rers as relay_encode_resync };
export { _reac as relay_encode_acp_command };

export {
  getApiClient, getAuthManager, getPodState, getPodService,
  getTicketService, getChannelService, getRunnerService,
  getLoopService, getAutopilotService, getMeshService,
  getRunnerState, getMeshState, getTicketState, getChannelState,
  getLoopState, getAcpManager, getOrgState, getUserState,
  getGitProviderState, getRepoState, getAutopilotState, getRelayManager,
  getBillingService, getRepositoryService, getExtensionService,
  getInvitationService, getApiKeyService, getBindingService,
  getGrantService,
  getMessageService, getNotificationService, getPromoCodeService,
  getTokenUsageService, getSSOService, getUserApiService,
  getUserCredentialService, getOrgApiService, getAgentService,
  getTicketRelationsService, getFileService, getSupportTicketService,
  getAuthConnectService, getBlockstoreService,
} from "@agentsmesh/service-runtime";
