import { ElectronPodService } from './pod';
import { ElectronRunnerService } from './runner';
import { ElectronTicketService } from './ticket';
import { ElectronChannelService } from './channel';
import { ElectronLoopService } from './loop';
import { ElectronAutopilotService } from './autopilot';
import { ElectronMeshService } from './mesh';
import { ElectronBillingService } from './billing';
import { ElectronExtensionService } from './extension';
import { ElectronRepositoryService } from './repository';
import { ElectronInvitationConnectService } from './invitation';
import { ElectronApiKeyService } from './apikey';
import { ElectronBindingService } from './binding';
import { ElectronNotificationService } from './notification';
import { ElectronOrgService } from './org';
import { ElectronUserService } from './user';
import { ElectronUserCredentialService } from './user_credential';
import { ElectronEnvBundleService } from './env_bundle';
import { ElectronAgentService } from './agent';
import { ElectronSSOService } from './sso';
import { ElectronFileService } from './file';
import { ElectronGrantService } from './grant';
import { ElectronSupportTicketService } from './support_ticket';
import { ElectronTicketRelationsService } from './ticket_relations';
import { ElectronTokenUsageService } from './token_usage';
import { ElectronPromoCodeService } from './promocode';
import { ElectronAuthService } from './auth';
import { ElectronAuthConnectService } from './auth_connect';
import { ElectronBlockstoreService } from './blockstore';
import { ElectronLocalRunnerService } from './local_runner';
import { ElectronEventsManager } from './realtime';
import {
  withConnectFallback,
  USER_CREDENTIAL_METHOD_OVERRIDES,
  USER_CREDENTIAL_NAME_OVERRIDES,
  BILLING_SERVICE_OVERRIDES,
  BILLING_NAME_OVERRIDES,
  AUTOPILOT_METHOD_NAMES,
  AGENT_USER_CONFIG_OVERRIDES,
  EXTENSION_SERVICE_OVERRIDES,
  EXTENSION_NAME_OVERRIDES,
} from "./connect-fallback";
import {
  ElectronAcpManager, ElectronRelayManager, ElectronTicketState,
} from './state_adapters';

/**
 * Mirrors the subset of `WasmApiClient` that the renderer still depends on —
 * which after R6/R7 is just `create_events_manager()` for the realtime
 * subscription manager. All historical raw HTTP methods (get/post/put/...)
 * have been removed; everything goes through typed Connect services now.
 */
class ElectronApiClientProxy {
  constructor(private readonly _auth: ElectronAuthService) {
    void this._auth;
  }

  /**
   * Wire-compatible with `WasmApiClient.create_events_manager()`. The
   * returned object fan-outs main-process EventBus IPC events into the
   * renderer's `EventSubscriptionManager` (clients/web/src/lib/realtime/).
   * See `packages/electron-adapter/src/realtime.ts` + `clients/desktop/
   * src/main/realtime.ts` for the full bridge.
   */
  create_events_manager(): ElectronEventsManager {
    return new ElectronEventsManager();
  }
}

export function createElectronServiceProvider(baseUrl = '') {
  // Services carry the full state (each IXxxService is a superset of IXxxState).
  // The provider returns the same instance for both keys so renderer readers
  // that grab `xxxState` see the cache Service writes to on fetch / upsert.
  // This is what makes Pod / Runner / Mesh / Ticket / Loop / Autopilot actually
  // populate in desktop first-load — previously `xxxState` was a stub class
  // returning `"[]"` that shadowed the real data.
  const podService = new ElectronPodService();
  const runnerService = new ElectronRunnerService();
  const ticketService = new ElectronTicketService();
  const channelService = new ElectronChannelService();
  const loopService = new ElectronLoopService();
  const autopilotService = new ElectronAutopilotService();
  const meshService = new ElectronMeshService();
  const repositoryService = new ElectronRepositoryService();
  // AuthManager is constructed first so ApiClient can borrow it as the org
  // slug source (Plan I6 SSOT). Order matters here.
  const authManager = new ElectronAuthService(baseUrl);

  return {
    apiClient: new ElectronApiClientProxy(authManager),
    authManager,
    podService: withConnectFallback(podService, "proto.pod.v1.PodService"),
    runnerService: withConnectFallback(runnerService, "proto.runner_api.v1.RunnerService"),
    ticketService: withConnectFallback(ticketService, "proto.ticket.v1.TicketService"),
    channelService: withConnectFallback(channelService, "proto.channel.v1.ChannelService"),
    loopService: withConnectFallback(loopService, "proto.loop.v1.LoopService"),
    autopilotService: withConnectFallback(autopilotService, "proto.autopilot.v1.AutopilotControllerService", undefined, AUTOPILOT_METHOD_NAMES),
    meshService: withConnectFallback(meshService, "proto.mesh.v1.MeshService"),
    billingService: withConnectFallback(new ElectronBillingService(), "proto.billing.v1.BillingService", BILLING_SERVICE_OVERRIDES, BILLING_NAME_OVERRIDES),
    extensionService: withConnectFallback(new ElectronExtensionService(), "proto.extension.v1.SkillRegistryService", EXTENSION_SERVICE_OVERRIDES, EXTENSION_NAME_OVERRIDES),
    repositoryService: withConnectFallback(repositoryService, "proto.repository.v1.RepositoryService"),
    invitationService: withConnectFallback(new ElectronInvitationConnectService(), "proto.invitation.v1.InvitationService"),
    apiKeyService: withConnectFallback(new ElectronApiKeyService(), "proto.apikey.v1.ApiKeyService"),
    bindingService: withConnectFallback(new ElectronBindingService(), "proto.binding.v1.BindingService"),
    notificationService: withConnectFallback(new ElectronNotificationService(), "proto.notification.v1.NotificationService"),
    orgApiService: withConnectFallback(new ElectronOrgService(), "proto.org.v1.OrgService"),
    userApiService: withConnectFallback(new ElectronUserService(), "proto.user.v1.UserService"),
    userCredentialService: withConnectFallback(
      new ElectronUserCredentialService(),
      "proto.user_credential.v1.UserRepositoryProviderService",
      USER_CREDENTIAL_METHOD_OVERRIDES,
      USER_CREDENTIAL_NAME_OVERRIDES,
    ),
    envBundleService: withConnectFallback(new ElectronEnvBundleService(), "proto.env_bundle.v1.EnvBundleService"),
    agentService: withConnectFallback(new ElectronAgentService(), "proto.agent.v1.AgentService", AGENT_USER_CONFIG_OVERRIDES),
    ssoService: withConnectFallback(new ElectronSSOService(), "proto.sso.v1.SSOService"),
    fileService: withConnectFallback(new ElectronFileService(), "proto.file.v1.FileService"),
    grantService: withConnectFallback(new ElectronGrantService(), "proto.grant.v1.GrantService"),
    supportTicketService: withConnectFallback(new ElectronSupportTicketService(), "proto.support_ticket.v1.SupportTicketService"),
    ticketRelationsService: withConnectFallback(new ElectronTicketRelationsService(), "proto.ticket_relations.v1.TicketRelationsService"),
    tokenUsageService: withConnectFallback(new ElectronTokenUsageService(), "proto.token_usage.v1.TokenUsageService"),
    promoCodeService: withConnectFallback(new ElectronPromoCodeService(), "proto.promocode.v1.PromoCodeService"),
    authConnectService: withConnectFallback(new ElectronAuthConnectService(), "proto.auth.v1.AuthService"),
    blockstoreService: withConnectFallback(new ElectronBlockstoreService(), "proto.blockstore.v1.BlockstoreService"),
    localRunnerService: new ElectronLocalRunnerService(),
    // State facets — share the Service instance, not a separate stub.
    podState: podService,
    runnerState: runnerService,
    meshState: meshService,
    ticketState: new ElectronTicketState(),
    channelState: channelService,
    loopState: loopService,
    autopilotState: autopilotService,
    repoState: repositoryService,
    acpManager: new ElectronAcpManager(),
    relayManager: new ElectronRelayManager(),
  };
}

