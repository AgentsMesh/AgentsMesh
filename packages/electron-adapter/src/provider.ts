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
import { ElectronInvitationService } from './invitation';
import { ElectronApiKeyService } from './apikey';
import { ElectronBindingService } from './binding';
import { ElectronMessageService } from './message';
import { ElectronNotificationService } from './notification';
import { ElectronOrgService } from './org';
import { ElectronUserService } from './user';
import { ElectronUserCredentialService } from './user_credential';
import { ElectronAgentService } from './agent';
import { ElectronSSOService } from './sso';
import { ElectronFileService } from './file';
import { ElectronSupportTicketService } from './support_ticket';
import { ElectronTicketRelationsService } from './ticket_relations';
import { ElectronTokenUsageService } from './token_usage';
import { ElectronPromoCodeService } from './promocode';
import { ElectronAuthService } from './auth';
import { ElectronAuthConnectService } from './auth_connect';
import { ElectronBlockstoreService } from './blockstore';
import { ElectronLocalRunnerService } from './local_runner';
import { invoke } from './invoke';
import {
  ElectronOrgState, ElectronUserState, ElectronGitProviderState, ElectronRepoState,
  ElectronAcpManager, ElectronRelayManager,
} from './state_adapters';

/**
 * Mirrors `WasmApiClient` shape for legacy callers that still go through
 * `lib/api/base.request`. Every raw HTTP method delegates to node-bridge IPC
 * handlers (`api_get` / `api_post` / ...), which call the shared Rust
 * `ApiClient` with auth token + org slug already bound on the native side.
 *
 * Plan I6 SSOT: orgSlug is read from the AuthManager (single source of
 * truth) instead of being mirrored via `set_org_slug()`. ElectronAuthService
 * caches the current org locally so `org_path()` stays synchronous.
 */
class ElectronApiClientProxy {
  constructor(private readonly auth: ElectronAuthService) {}

  /** Must match Rust `ApiClient::org_path`. */
  org_path(path: string): string {
    const slug = this.auth.get_current_org_slug();
    return slug ? `/api/v1/orgs/${slug}${path}` : `/api/v1${path}`;
  }

  async get(endpoint: string): Promise<string> {
    return invoke<string>("apiGet", endpoint);
  }
  async post(endpoint: string, body: string): Promise<string> {
    return invoke<string>("apiPost", endpoint, body ?? "");
  }
  async put(endpoint: string, body: string): Promise<string> {
    return invoke<string>("apiPut", endpoint, body ?? "");
  }
  async patch(endpoint: string, body: string): Promise<string> {
    return invoke<string>("apiPatch", endpoint, body ?? "");
  }
  async delete(endpoint: string): Promise<string> {
    return invoke<string>("apiDelete", endpoint);
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
  // AuthManager is constructed first so ApiClient can borrow it as the org
  // slug source (Plan I6 SSOT). Order matters here.
  const authManager = new ElectronAuthService(baseUrl);

  return {
    apiClient: new ElectronApiClientProxy(authManager),
    authManager,
    podService,
    runnerService,
    ticketService,
    channelService,
    loopService,
    autopilotService,
    meshService,
    billingService: new ElectronBillingService(),
    extensionService: new ElectronExtensionService(),
    repositoryService: new ElectronRepositoryService(),
    invitationService: new ElectronInvitationService(),
    apiKeyService: new ElectronApiKeyService(),
    bindingService: new ElectronBindingService(),
    messageService: new ElectronMessageService(),
    notificationService: new ElectronNotificationService(),
    orgApiService: new ElectronOrgService(),
    userApiService: new ElectronUserService(),
    userCredentialService: new ElectronUserCredentialService(),
    agentService: new ElectronAgentService(),
    ssoService: new ElectronSSOService(),
    fileService: new ElectronFileService(),
    supportTicketService: new ElectronSupportTicketService(),
    ticketRelationsService: new ElectronTicketRelationsService(),
    tokenUsageService: new ElectronTokenUsageService(),
    promoCodeService: new ElectronPromoCodeService(),
    authConnectService: new ElectronAuthConnectService(),
    blockstoreService: new ElectronBlockstoreService(),
    localRunnerService: new ElectronLocalRunnerService(),
    // State facets — share the Service instance, not a separate stub.
    podState: podService,
    runnerState: runnerService,
    meshState: meshService,
    ticketState: ticketService,
    channelState: channelService,
    loopState: loopService,
    autopilotState: autopilotService,
    orgState: new ElectronOrgState(),
    userState: new ElectronUserState(),
    gitProviderState: new ElectronGitProviderState(),
    repoState: new ElectronRepoState(),
    acpManager: new ElectronAcpManager(),
    relayManager: new ElectronRelayManager(),
  };
}
