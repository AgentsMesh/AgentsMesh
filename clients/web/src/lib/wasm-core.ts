import initWasm, {
  version, WasmApiClient, WasmAuthManager, WasmEventsManager, WasmWebSocket,
  init_logger, log_event,
} from "agentsmesh-wasm";
import { markServiceReady, setPlatformInit } from "@agentsmesh/service-runtime";
import { getApiBaseUrl } from "./env";
import { registerAll } from "./wasm-getters";
import { installConsoleCapture } from "./console-capture";
import { logger } from "./logger";

async function doWasmInit(): Promise<void> {
  await initWasm();
  // Wire up the Rust tracing subscriber before anything else runs — every
  // `tracing::*` call across the workspace becomes a no-op until this
  // returns. WASM has no filesystem, so sinks bind to console.* via
  // tracing-wasm. Idempotent on the Rust side.
  try {
    init_logger(process.env.NEXT_PUBLIC_LOG_LEVEL ?? "info");
  } catch (e) {
    console.warn("[WASM Core] init_logger failed:", e);
  }
  // Mirror console.warn/error (and log/info under verbose) into the same
  // subscriber so renderer-side warnings land in the rolling log file
  // alongside Rust events. Safe to call once tracing is wired up.
  installConsoleCapture();
  let baseUrl = getApiBaseUrl();
  if (!baseUrl && typeof window !== "undefined") baseUrl = window.location.origin;
  // AuthManager owns the persisted token store; ApiClient borrows it.
  // Plan I6 (single source of truth) — never two parallel auth instances.
  const authManager = new WasmAuthManager(baseUrl);
  const apiClient = new WasmApiClient(baseUrl, authManager);
  registerAll(apiClient, authManager);
  markServiceReady();
  logger.info("WasmCore", `Initialized, version: ${version()}`);
}

setPlatformInit(doWasmInit);

export { ensurePlatformReady as initWasmCore } from "@agentsmesh/service-runtime";
export { isServiceReady as isWasmReady, NOOP_PROXY, parseWasmAny } from "@agentsmesh/service-runtime";

export { WasmEventsManager, WasmWebSocket };
export { log_event as wasmLogEvent };

export {
  getApiClient, getAuthManager, getPodState, getPodService,
  getTicketService, getChannelService, getRunnerService,
  getLoopService, getAutopilotService, getMeshService,
  getRunnerState, getMeshState, getTicketState, getChannelState,
  getLoopState, getAcpManager, getLoopalManager,
  getRepoState, getAutopilotState, getRelayManager,
  getBillingService, getRepositoryService, getExtensionService,
  getInvitationService, getApiKeyService, getBindingService,
  getGrantService,
  getNotificationService, getPromoCodeService,
  getTokenUsageService, getSSOService, getUserApiService,
  getUserCredentialService, getEnvBundleService, getOrgApiService, getAgentService,
  getTicketRelationsService, getFileService, getSupportTicketService,
  getAuthConnectService, getBlockstoreService,
} from "@agentsmesh/service-runtime";
