import { describe, it, expect, beforeEach } from "vitest";
import {
  withConnectFallback,
  AUTOPILOT_METHOD_NAMES,
  AGENT_USER_CONFIG_OVERRIDES,
  EXTENSION_SERVICE_OVERRIDES,
  EXTENSION_NAME_OVERRIDES,
  USER_CREDENTIAL_METHOD_OVERRIDES,
  USER_CREDENTIAL_NAME_OVERRIDES,
  BILLING_SERVICE_OVERRIDES,
  BILLING_NAME_OVERRIDES,
} from "./connect-fallback";

// Regression guard for the derived backend Connect path. The renderer reaches
// an RPC as `getXxxService().<verb>Connect(bytes)`; withConnectFallback derives
// (service, method). Most derive 1:1, but several need overrides — autopilot
// (proto "Controller(s)" suffix the wasm verb drops), agent / extension /
// userCredential (TS facade spanning multiple proto services), billing
// (BillingPublicService split + "Subscription" suffix the wasm verb drops).
//
// Only facades that actually reach the proxy in production are asserted here.
// invitation / promocode are deliberately absent: ElectronInvitationConnectService
// and ElectronPromoCodeService implement every `*Connect` method directly (their
// own dedicated IPC channels), so `Reflect.get` returns the real method and the
// proxy is never consulted — asserting an override path for them would test a
// derivation that never runs in production.
describe("connect fallback routing", () => {
  let calls: Array<{ service: string; method: string }>;

  beforeEach(() => {
    calls = [];
    (globalThis as { window?: unknown }).window = {
      electronAPI: {
        invoke: async (channel: string, ...args: unknown[]) => {
          if (channel === "connectCall") {
            calls.push({ service: args[0] as string, method: args[1] as string });
          }
          return [];
        },
      },
    };
  });

  const autopilot = () =>
    withConnectFallback({}, "proto.autopilot.v1.AutopilotControllerService", undefined, AUTOPILOT_METHOD_NAMES);
  const agent = () =>
    withConnectFallback({}, "proto.agent.v1.AgentService", AGENT_USER_CONFIG_OVERRIDES);
  const extension = () =>
    withConnectFallback({}, "proto.extension.v1.SkillRegistryService", EXTENSION_SERVICE_OVERRIDES, EXTENSION_NAME_OVERRIDES);
  const userCredential = () =>
    withConnectFallback({}, "proto.user_credential.v1.UserRepositoryProviderService", USER_CREDENTIAL_METHOD_OVERRIDES, USER_CREDENTIAL_NAME_OVERRIDES);
  const billing = () =>
    withConnectFallback({}, "proto.billing.v1.BillingService", BILLING_SERVICE_OVERRIDES, BILLING_NAME_OVERRIDES);

  const cases: Array<[() => object, string, string, string]> = [
    // autopilot — name override (keeps/restores "Controller(s)")
    [autopilot, "listAutopilotsConnect", "proto.autopilot.v1.AutopilotControllerService", "ListAutopilotControllers"],
    [autopilot, "getAutopilotConnect", "proto.autopilot.v1.AutopilotControllerService", "GetAutopilotController"],
    [autopilot, "pauseAutopilotConnect", "proto.autopilot.v1.AutopilotControllerService", "PauseAutopilotController"],
    [autopilot, "getIterationsConnect", "proto.autopilot.v1.AutopilotControllerService", "GetIterations"],
    // agent — sibling UserAgentConfigService; catalog stays on AgentService
    [agent, "getUserAgentConfigConnect", "proto.agent.v1.UserAgentConfigService", "GetUserAgentConfig"],
    [agent, "setUserAgentConfigConnect", "proto.agent.v1.UserAgentConfigService", "SetUserAgentConfig"],
    [agent, "listAgentsConnect", "proto.agent.v1.AgentService", "ListAgents"],
    // extension — four sub-services + GitHub casing
    [extension, "createSkillRegistryConnect", "proto.extension.v1.SkillRegistryService", "CreateSkillRegistry"],
    [extension, "listMarketSkillsConnect", "proto.extension.v1.MarketService", "ListMarketSkills"],
    [extension, "installMcpFromMarketConnect", "proto.extension.v1.RepoMcpService", "InstallMcpFromMarket"],
    [extension, "listRepoSkillsConnect", "proto.extension.v1.RepoSkillService", "ListRepoSkills"],
    [extension, "installSkillFromGithubConnect", "proto.extension.v1.RepoSkillService", "InstallSkillFromGitHub"],
    // user_credential — three sibling services; git verb matches, agent verb
    // ALSO drops "Profile" (service + name), repository-provider is the default
    [userCredential, "listGitCredentialsConnect", "proto.user_credential.v1.UserGitCredentialService", "ListGitCredentials"],
    [userCredential, "createAgentCredentialConnect", "proto.user_credential.v1.UserAgentCredentialService", "CreateAgentCredentialProfile"],
    [userCredential, "listAgentCredentialsForAgentConnect", "proto.user_credential.v1.UserAgentCredentialService", "ListAgentCredentialProfilesForAgent"],
    [userCredential, "createRepositoryProviderConnect", "proto.user_credential.v1.UserRepositoryProviderService", "CreateRepositoryProvider"],
    // billing — public RPCs live on the sibling BillingPublicService; four
    // subscription verbs drop the proto's "Subscription"/"BillingCycle" suffix
    [billing, "reactivate_connect", "proto.billing.v1.BillingService", "ReactivateSubscription"],
    [billing, "request_cancel_connect", "proto.billing.v1.BillingService", "RequestCancelSubscription"],
    [billing, "upgrade_connect", "proto.billing.v1.BillingService", "UpgradeSubscription"],
    [billing, "change_cycle_connect", "proto.billing.v1.BillingService", "ChangeBillingCycle"],
    [billing, "get_public_pricing_connect", "proto.billing.v1.BillingPublicService", "GetPublicPricing"],
    [billing, "get_public_deployment_info_connect", "proto.billing.v1.BillingPublicService", "GetPublicDeploymentInfo"],
  ];

  for (const [facade, verb, service, method] of cases) {
    it(`${verb} → ${service.split(".").pop()}/${method}`, async () => {
      await (facade() as Record<string, (b: Uint8Array) => Promise<unknown>>)[verb](new Uint8Array());
      expect(calls).toEqual([{ service, method }]);
    });
  }

  it("default service derives 1:1 for both camel + snake surfaces", async () => {
    const pod = withConnectFallback({}, "proto.pod.v1.PodService") as Record<string, (b: Uint8Array) => Promise<unknown>>;
    await pod.listPodsConnect(new Uint8Array());
    await pod.list_pods_connect(new Uint8Array());
    expect(calls).toEqual([
      { service: "proto.pod.v1.PodService", method: "ListPods" },
      { service: "proto.pod.v1.PodService", method: "ListPods" },
    ]);
  });

  it("resolves the coerced Uint8Array from the IPC response", async () => {
    (globalThis as { window?: unknown }).window = {
      electronAPI: { invoke: async () => [1, 2, 3] },
    };
    const pod = withConnectFallback({}, "proto.pod.v1.PodService") as Record<string, (b: Uint8Array) => Promise<Uint8Array>>;
    const out = await pod.listPodsConnect(new Uint8Array());
    expect(out).toBeInstanceOf(Uint8Array);
    expect(Array.from(out)).toEqual([1, 2, 3]);
  });

  it("fails closed at access time on a malformed derivation", () => {
    const svc = withConnectFallback({}, "proto.pod.v1.PodService") as Record<string, unknown>;
    expect(() => svc.Connect).toThrow(/cannot derive/);
    expect(calls).toEqual([]);
  });

  it("fails closed when the IPC response is not binary", async () => {
    (globalThis as { window?: unknown }).window = {
      electronAPI: { invoke: async () => ({ unexpected: true }) },
    };
    const svc = withConnectFallback({}, "proto.pod.v1.PodService") as Record<string, (b: Uint8Array) => Promise<unknown>>;
    await expect(svc.listPodsConnect(new Uint8Array())).rejects.toThrow(/not binary/);
  });
});
