/* eslint-disable @typescript-eslint/no-explicit-any */
import type {
  WasmApiClient, WasmAuthManager, WasmPodService, WasmPodState,
  WasmTicketService, WasmChannelService, WasmRunnerService,
  WasmLoopService, WasmAutopilotService, WasmMeshService,
  WasmBillingService, WasmRepositoryService, WasmExtensionService,
  WasmInvitationService, WasmApiKeyService, WasmBindingService,
  WasmGrantService, WasmNotificationService, WasmPromoCodeService,
  WasmTokenUsageService, WasmSSOService, WasmUserApiService,
  WasmUserCredentialService, WasmEnvBundleService, WasmOrgApiService,
  WasmAgentService, WasmTicketRelationsService, WasmFileService,
  WasmSupportTicketService, WasmAuthConnectService, WasmBlockstoreService,
  WasmRunnerState, WasmMeshState, WasmTicketState, WasmChannelState,
  WasmLoopState, WasmAcpSessionManager, WasmLoopalManager, WasmRepoState,
  WasmAutopilotState, WasmRelayManager,
} from "agentsmesh-wasm";
import type { ILocalRunnerService } from "@agentsmesh/service-interface";

// SSR / hydration fallback. Returns "[]" for `*_json` reads so SSR-rendered
// components don't crash before the wasm bridge is registered; all other
// accesses no-op. Cast to `any` is necessary — this is intentionally
// untyped because it stands in for any service shape.
export const NOOP_PROXY = new Proxy({} as any, {
  get: (_target, prop) => {
    if (prop === "then") return undefined;
    return () => {
      if (typeof prop === "string" && (prop.endsWith("_json") || prop === "org_path")) return "[]";
      return undefined;
    };
  },
});

// Typed registry — every wasm-bindgen surface that goes through
// registerServiceProvider() must have a slot here. Adding a new service:
// (1) add the WasmXxx import above, (2) add a slot here, (3) add a getter
// below. The Bazel build hard-fails if registerServiceProvider() is called
// with a key not in this interface.
export interface ServiceRegistry {
  apiClient: WasmApiClient;
  authManager: WasmAuthManager;
  podState: WasmPodState;
  podService: WasmPodService;
  ticketService: WasmTicketService;
  channelService: WasmChannelService;
  runnerService: WasmRunnerService;
  loopService: WasmLoopService;
  autopilotService: WasmAutopilotService;
  meshService: WasmMeshService;
  billingService: WasmBillingService;
  repositoryService: WasmRepositoryService;
  extensionService: WasmExtensionService;
  invitationService: WasmInvitationService;
  grantService: WasmGrantService;
  apiKeyService: WasmApiKeyService;
  bindingService: WasmBindingService;
  notificationService: WasmNotificationService;
  promoCodeService: WasmPromoCodeService;
  tokenUsageService: WasmTokenUsageService;
  ssoService: WasmSSOService;
  userApiService: WasmUserApiService;
  userCredentialService: WasmUserCredentialService;
  envBundleService: WasmEnvBundleService;
  orgApiService: WasmOrgApiService;
  agentService: WasmAgentService;
  ticketRelationsService: WasmTicketRelationsService;
  fileService: WasmFileService;
  supportTicketService: WasmSupportTicketService;
  authConnectService: WasmAuthConnectService;
  blockstoreService: WasmBlockstoreService;
  runnerState: WasmRunnerState;
  meshState: WasmMeshState;
  ticketState: WasmTicketState;
  channelState: WasmChannelState;
  loopState: WasmLoopState;
  acpManager: WasmAcpSessionManager;
  // Loopal control-panel state. Optional — only the web build registers it;
  // desktop falls back to NOOP_PROXY (Loopal console shows empty panels).
  loopalManager?: WasmLoopalManager;
  repoState: WasmRepoState;
  autopilotState: WasmAutopilotState;
  relayManager: WasmRelayManager;
  // Platform-conditional — desktop adapter only.
  localRunnerService?: ILocalRunnerService;
}

interface ServiceRegistryStore {
  ready: boolean;
  instances: Partial<ServiceRegistry>;
}

function makeStore(): ServiceRegistryStore { return { ready: false, instances: {} }; }

// On the browser we promote the storage onto `globalThis` so it survives
// Next.js dev-chunk re-evaluations — multiple module instances of this
// file would otherwise each carry their own (empty) store. On the server
// we keep module-scoped state — globalThis on SSR side leaks between
// requests and we'd hand a half-initialised client store to the next
// renderer pass.
const isBrowser = typeof window !== "undefined";
const moduleStore: ServiceRegistryStore = makeStore();

function registry(): ServiceRegistryStore {
  if (!isBrowser) return moduleStore;
  const g = globalThis as unknown as { __amesh_svc_registry?: ServiceRegistryStore };
  if (!g.__amesh_svc_registry) g.__amesh_svc_registry = moduleStore;
  return g.__amesh_svc_registry;
}

export function isServiceReady(): boolean { return registry().ready; }
export function markServiceReady(): void {
  registry().ready = true;
  // Mirror to globalThis so e2e tests (Playwright + Electron) can
  // synchronize navigation with wasm-core readiness — without this flag
  // the hash router's route guards fire before Connect-RPC cache
  // populators land, bouncing every cold-route nav back to /workspace.
  // See clients/desktop/e2e/helpers/nav.ts:waitForServicesReady().
  if (typeof globalThis !== "undefined") {
    (globalThis as { __amesh_ready__?: boolean }).__amesh_ready__ = true;
  }
}

// NOOP_PROXY cast: TS can't know this proxy satisfies any specific WasmXxx
// shape, but we deliberately want fall-through behavior pre-hydration.
const g = <K extends keyof ServiceRegistry>(k: K): ServiceRegistry[K] => {
  const reg = registry();
  return (reg.ready ? reg.instances[k] : NOOP_PROXY) as ServiceRegistry[K];
};

export function registerServiceProvider(provider: Partial<ServiceRegistry>) {
  const reg = registry();
  for (const [key, value] of Object.entries(provider)) {
    if (value === undefined) continue;
    (reg.instances as Record<string, unknown>)[key] = value;
  }
}

export function parseWasmAny<T>(val: unknown): T | null {
  if (!val) return null;
  return typeof val === "string" ? JSON.parse(val) : (val as T);
}

export const getApiClient = () => g("apiClient");
export const getAuthManager = () => g("authManager");
export const getPodState = () => g("podState");
export const getPodService = () => g("podService");
export const getTicketService = () => g("ticketService");
export const getChannelService = () => g("channelService");
export const getRunnerService = () => g("runnerService");
export const getLoopService = () => g("loopService");
export const getAutopilotService = () => g("autopilotService");
export const getMeshService = () => g("meshService");
export const getBillingService = () => g("billingService");
export const getRepositoryService = () => g("repositoryService");
export const getExtensionService = () => g("extensionService");
export const getInvitationService = () => g("invitationService");
export const getGrantService = () => g("grantService");
export const getApiKeyService = () => g("apiKeyService");
export const getBindingService = () => g("bindingService");
export const getNotificationService = () => g("notificationService");
export const getPromoCodeService = () => g("promoCodeService");
export const getTokenUsageService = () => g("tokenUsageService");
export const getSSOService = () => g("ssoService");
export const getUserApiService = () => g("userApiService");
export const getUserCredentialService = () => g("userCredentialService");
export const getEnvBundleService = () => g("envBundleService");
export const getOrgApiService = () => g("orgApiService");
export const getAgentService = () => g("agentService");
export const getTicketRelationsService = () => g("ticketRelationsService");
export const getFileService = () => g("fileService");
export const getSupportTicketService = () => g("supportTicketService");
export const getAuthConnectService = () => g("authConnectService");
export const getRunnerState = () => g("runnerState");
export const getMeshState = () => g("meshState");
export const getTicketState = () => g("ticketState");
export const getChannelState = () => g("channelState");
export const getLoopState = () => g("loopState");
export const getAcpManager = () => g("acpManager");
export const getLoopalManager = (): WasmLoopalManager => {
  const reg = registry();
  return (reg.ready ? reg.instances.loopalManager ?? NOOP_PROXY : NOOP_PROXY) as WasmLoopalManager;
};
export const getRepoState = () => g("repoState");
export const getAutopilotState = () => g("autopilotState");
export const getRelayManager = () => g("relayManager");
export const getBlockstoreService = () => g("blockstoreService");

// Optional — returns undefined when no provider has registered a local-runner
// service (web/iOS builds, where the desktop adapter is absent). Renderer UI
// uses this to feature-detect and hide onboarding cards on platforms that
// can't host a local runner.
export const getLocalRunnerService = (): ILocalRunnerService | undefined => {
  const reg = registry();
  return reg.ready ? reg.instances.localRunnerService : undefined;
};
