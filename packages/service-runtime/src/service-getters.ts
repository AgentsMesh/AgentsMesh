/* eslint-disable @typescript-eslint/no-explicit-any */

export const NOOP_PROXY = new Proxy({} as any, {
  get: (_target, prop) => {
    if (prop === "then") return undefined;
    return () => {
      if (typeof prop === "string" && (prop.endsWith("_json") || prop === "org_path")) return "[]";
      return undefined;
    };
  },
});

let ready = false;
const i: Record<string, any> = {};

export function isServiceReady(): boolean { return ready; }
export function markServiceReady(): void { ready = true; }

const g = <T>(k: string): T => (ready ? i[k] : NOOP_PROXY) as T;

export function registerServiceProvider(provider: Record<string, any>) {
  for (const [key, value] of Object.entries(provider)) {
    i[key] = value;
  }
}

export function parseWasmAny<T>(val: unknown): T | null {
  if (!val) return null;
  return typeof val === "string" ? JSON.parse(val) : (val as T);
}

export const getApiClient = () => g<any>("apiClient");
export const getAuthManager = () => g<any>("authManager");
export const getPodState = () => g<any>("podState");
export const getPodService = () => g<any>("podService");
export const getTicketService = () => g<any>("ticketService");
export const getChannelService = () => g<any>("channelService");
export const getRunnerService = () => g<any>("runnerService");
export const getLoopService = () => g<any>("loopService");
export const getAutopilotService = () => g<any>("autopilotService");
export const getMeshService = () => g<any>("meshService");
export const getBillingService = () => g<any>("billingService");
export const getRepositoryService = () => g<any>("repositoryService");
export const getExtensionService = () => g<any>("extensionService");
export const getInvitationService = () => g<any>("invitationService");
export const getGrantService = () => g<any>("grantService");
export const getApiKeyService = () => g<any>("apiKeyService");
export const getBindingService = () => g<any>("bindingService");
export const getMessageService = () => g<any>("messageService");
export const getNotificationService = () => g<any>("notificationService");
export const getPromoCodeService = () => g<any>("promoCodeService");
export const getTokenUsageService = () => g<any>("tokenUsageService");
export const getSSOService = () => g<any>("ssoService");
export const getUserApiService = () => g<any>("userApiService");
export const getUserCredentialService = () => g<any>("userCredentialService");
export const getOrgApiService = () => g<any>("orgApiService");
export const getAgentService = () => g<any>("agentService");
export const getTicketRelationsService = () => g<any>("ticketRelationsService");
export const getFileService = () => g<any>("fileService");
export const getSupportTicketService = () => g<any>("supportTicketService");
export const getAuthConnectService = () => g<any>("authConnectService");
export const getRunnerState = () => g<any>("runnerState");
export const getMeshState = () => g<any>("meshState");
export const getTicketState = () => g<any>("ticketState");
export const getChannelState = () => g<any>("channelState");
export const getLoopState = () => g<any>("loopState");
export const getAcpManager = () => g<any>("acpManager");
export const getOrgState = () => g<any>("orgState");
export const getUserState = () => g<any>("userState");
export const getGitProviderState = () => g<any>("gitProviderState");
export const getRepoState = () => g<any>("repoState");
export const getAutopilotState = () => g<any>("autopilotState");
export const getRelayManager = () => g<any>("relayManager");
export const getBlockstoreService = () => g<any>("blockstoreService");

/**
 * Optional getter — returns undefined when no provider has registered a
 * local-runner service (web/iOS builds, where the desktop adapter is absent).
 * Renderer UI uses this to feature-detect and hide onboarding cards on
 * platforms that can't host a local runner.
 */
export const getLocalRunnerService = () =>
  ready ? (i["localRunnerService"] as any | undefined) : undefined;
