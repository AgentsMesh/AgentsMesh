import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import { unwrapIpcServiceError } from "./connect-ipc-error";

// Connect-RPC routing config + fallback proxy for the desktop renderer.
// Kept separate from provider.ts (which imports every ElectronXxxService,
// each pulling @agentsmesh/proto subpaths) so this — the routing brain — is
// importable in isolation for unit tests.

// user_credential proto bundles three backend services (git credential, agent
// credential, repository provider) under a single wasm-side facade. The default
// protoPath is UserRepositoryProviderService; route the other two here.
export const USER_CREDENTIAL_METHOD_OVERRIDES: Record<string, string> = {
  // git credential — renderer verb matches the proto RPC; only the service differs
  listGitCredentials: "proto.user_credential.v1.UserGitCredentialService",
  getGitCredential: "proto.user_credential.v1.UserGitCredentialService",
  createGitCredential: "proto.user_credential.v1.UserGitCredentialService",
  updateGitCredential: "proto.user_credential.v1.UserGitCredentialService",
  deleteGitCredential: "proto.user_credential.v1.UserGitCredentialService",
  getDefaultGitCredential: "proto.user_credential.v1.UserGitCredentialService",
  setDefaultGitCredential: "proto.user_credential.v1.UserGitCredentialService",
  clearDefaultGitCredential: "proto.user_credential.v1.UserGitCredentialService",
  // agent credential — renderer verb drops the proto's "Profile" suffix, so it
  // needs BOTH a service route (here) and a name remap (USER_CREDENTIAL_NAME_*).
  listAgentCredentials: "proto.user_credential.v1.UserAgentCredentialService",
  listAgentCredentialsForAgent: "proto.user_credential.v1.UserAgentCredentialService",
  getAgentCredential: "proto.user_credential.v1.UserAgentCredentialService",
  createAgentCredential: "proto.user_credential.v1.UserAgentCredentialService",
  updateAgentCredential: "proto.user_credential.v1.UserAgentCredentialService",
  deleteAgentCredential: "proto.user_credential.v1.UserAgentCredentialService",
  setDefaultAgentCredential: "proto.user_credential.v1.UserAgentCredentialService",
};
// agent-credential verbs drop the proto's "Profile" suffix.
export const USER_CREDENTIAL_NAME_OVERRIDES: Record<string, string> = {
  listAgentCredentials: "ListAgentCredentialProfiles",
  listAgentCredentialsForAgent: "ListAgentCredentialProfilesForAgent",
  getAgentCredential: "GetAgentCredentialProfile",
  createAgentCredential: "CreateAgentCredentialProfile",
  updateAgentCredential: "UpdateAgentCredentialProfile",
  deleteAgentCredential: "DeleteAgentCredentialProfile",
  setDefaultAgentCredential: "SetDefaultAgentCredentialProfile",
};

// billing splits into BillingService (default) and BillingPublicService
// (pre-auth pricing / deployment info). Four subscription verbs also drop the
// proto's "Subscription" / "BillingCycle" suffix on the wasm side.
export const BILLING_SERVICE_OVERRIDES: Record<string, string> = {
  getPublicPricing: "proto.billing.v1.BillingPublicService",
  getPublicDeploymentInfo: "proto.billing.v1.BillingPublicService",
};
export const BILLING_NAME_OVERRIDES: Record<string, string> = {
  reactivate: "ReactivateSubscription",
  requestCancel: "RequestCancelSubscription",
  upgrade: "UpgradeSubscription",
  changeCycle: "ChangeBillingCycle",
};

// autopilot's proto methods carry a "Controller(s)" suffix the wasm verbs drop
// (listAutopilotsConnect → must POST ListAutopilotControllers). Remap each.
// (getIterations already matches its proto name, so it's intentionally absent.)
export const AUTOPILOT_METHOD_NAMES: Record<string, string> = {
  listAutopilots: "ListAutopilotControllers",
  getAutopilot: "GetAutopilotController",
  createAutopilot: "CreateAutopilotController",
  pauseAutopilot: "PauseAutopilotController",
  resumeAutopilot: "ResumeAutopilotController",
  stopAutopilot: "StopAutopilotController",
  takeoverAutopilot: "TakeoverAutopilotController",
  handbackAutopilot: "HandbackAutopilotController",
  approveAutopilot: "ApproveAutopilotController",
};

// agent bundles AgentService (catalog) + UserAgentConfigService (user config).
// Route the four user-config verbs to the sibling service.
export const AGENT_USER_CONFIG_OVERRIDES: Record<string, string> = {
  listUserAgentConfigs: "proto.agent.v1.UserAgentConfigService",
  getUserAgentConfig: "proto.agent.v1.UserAgentConfigService",
  setUserAgentConfig: "proto.agent.v1.UserAgentConfigService",
  deleteUserAgentConfig: "proto.agent.v1.UserAgentConfigService",
};

// extension splits into SkillRegistryService (default), RepoMcpService /
// RepoSkillService (per-repo install/manage), MarketService (catalog browse).
// presignSkillUpload / installSkillFromUploadedFile are deliberately absent:
// ElectronExtensionService implements those *Connect methods directly (dedicated
// IPC channels), so the proxy never sees them.
export const EXTENSION_SERVICE_OVERRIDES: Record<string, string> = {
  listMarketMcpServers: "proto.extension.v1.MarketService",
  listMarketSkills: "proto.extension.v1.MarketService",
  installCustomMcpServer: "proto.extension.v1.RepoMcpService",
  installMcpFromMarket: "proto.extension.v1.RepoMcpService",
  listRepoMcpServers: "proto.extension.v1.RepoMcpService",
  uninstallMcpServer: "proto.extension.v1.RepoMcpService",
  updateMcpServer: "proto.extension.v1.RepoMcpService",
  installSkillFromGithub: "proto.extension.v1.RepoSkillService",
  installSkillFromMarket: "proto.extension.v1.RepoSkillService",
  listRepoSkills: "proto.extension.v1.RepoSkillService",
  uninstallSkill: "proto.extension.v1.RepoSkillService",
  updateSkill: "proto.extension.v1.RepoSkillService",
};
// RepoSkillService spells the RPC "GitHub"; the wasm verb shortened it to
// "Github", which the default first-letter-cap derivation can't recover.
export const EXTENSION_NAME_OVERRIDES: Record<string, string> = {
  installSkillFromGithub: "InstallSkillFromGitHub",
};

// Web's wasm services expose `<verb>Connect(Uint8Array) -> Uint8Array` methods.
// Most ElectronXxxService classes only carry the legacy JSON surface, so a
// `*Connect` lookup misses. This proxy forwards any unimplemented `*Connect` /
// `*_connect` access onto the generic `connectCall` IPC handler, deriving the
// proto (service, method) from the camelCased verb.
//
// `methodOverrides` routes a verb to a sibling backend service (a TS facade
// spanning multiple proto services). `methodNameOverrides` remaps the derived
// method name (wasm verb diverging from the proto RPC name).
export function withConnectFallback<T extends object>(
  service: T,
  protoPath: string,
  methodOverrides?: Record<string, string>,
  methodNameOverrides?: Record<string, string>,
): T {
  return new Proxy(service, {
    get(target, prop) {
      const value = Reflect.get(target, prop);
      if (value !== undefined) return value;
      if (typeof prop !== "string") return undefined;

      let camel: string;
      if (prop.endsWith("Connect")) {
        camel = prop.slice(0, -"Connect".length);
      } else if (prop.endsWith("_connect")) {
        const snake = prop.slice(0, -"_connect".length);
        camel = snake.replace(/_([a-z0-9])/g, (_, c) => c.toUpperCase());
      } else {
        return undefined;
      }
      const protoMethod = methodNameOverrides?.[camel]
        ?? camel.charAt(0).toUpperCase() + camel.slice(1);
      // Fail closed at access time: an empty stem ("Connect"/"_connect") or a
      // verb the snake→camel pass can't normalize (leftover "_") can't name a
      // real RPC. Throwing here (not at call time) keeps `in`/`typeof` honest
      // and runs the check once per property access, not per call.
      if (!/^[A-Z][A-Za-z0-9]*$/.test(protoMethod)) {
        throw new Error(`connect-fallback: cannot derive a proto method from "${prop}"`);
      }
      const targetService = methodOverrides?.[camel] ?? protoPath;
      return async (request: Uint8Array): Promise<Uint8Array> => {
        try {
          return coerceConnectResponse(
            await invoke<number[] | Uint8Array>(
              "connectCall", targetService, protoMethod, Array.from(request),
            ),
          );
        } catch (e) {
          throw unwrapIpcServiceError(e);
        }
      };
    },
  });
}
