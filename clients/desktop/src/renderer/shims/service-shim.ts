export {
  NOOP_PROXY, isServiceReady as isWasmReady,
  registerServiceProvider, parseWasmAny,
  ensurePlatformReady as initWasmCore,
  getApiClient, getAuthManager, getPodState, getPodService,
  getTicketService, getChannelService, getRunnerService,
  getLoopService, getAutopilotService, getMeshService,
  getBillingService, getRepositoryService, getExtensionService,
  getInvitationService, getApiKeyService, getBindingService,
  getGrantService,
  getMessageService, getNotificationService, getPromoCodeService,
  getTokenUsageService, getSSOService, getUserApiService,
  getUserCredentialService, getOrgApiService, getAgentService,
  getTicketRelationsService, getFileService, getSupportTicketService,
  getAuthConnectService, getRunnerState, getMeshState, getTicketState,
  getChannelState, getLoopState, getAcpManager, getOrgState,
  getUserState, getGitProviderState, getRepoState,
  getAutopilotState, getRelayManager, getBlockstoreService,
} from "@agentsmesh/service-runtime";

// Relay protocol codec — pure JS mirror of clients/core/crates/protocol. Desktop
// renderer owns the relay WebSocket directly; no IPC hop is involved.
export {
  relay_encode_input,
  relay_encode_resize,
  relay_encode_ping,
  relay_encode_control,
  relay_encode_resync,
  relay_encode_acp_command,
  relay_decode_message,
} from "@/lib/relay/codec-pure";
