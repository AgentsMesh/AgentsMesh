// Base utilities
export { request, ApiError } from "./base";
export type { RequestOptions, ApiErrorData } from "./base";

// Error utilities
export { getApiErrorCode, isApiErrorCode, isApiStatus, getLocalizedErrorMessage } from "./errors";

// Auth
export { authApi } from "./auth";

// User
export { userApi } from "./user";

// Organization
export { organizationApi } from "./organization";
export type { OrganizationMember } from "./organization";

// Pod
export { podApi } from "./pod";
export type { PodData } from "./pod";

// Channel
export { channelApi } from "./channel";
export type { ChannelData, ChannelMessage } from "./channel";

// Ticket
export { ticketApi } from "./ticket";
export type {
  TicketType,
  TicketStatus,
  TicketPriority,
  TicketData,
  TicketRelation,
  TicketCommit,
  TicketComment,
  BoardColumn,
} from "./ticket";

// Runner
export { runnerApi } from "./runner";
export type { RunnerData, GRPCRegistrationToken, RunnerPodData, SandboxStatus, RelayConnectionInfo } from "./runner";

// Agent
export { agentApi, userAgentConfigApi } from "./agent";
export type {
  AgentTypeData,
  UserAgentConfigData,
  ConfigField,
  ConfigFieldOption,
  ConfigSchema,
} from "./agent";

// Repository
export { repositoryApi } from "./repository";
export type {
  RepositoryData,
  CreateRepositoryRequest,
  UpdateRepositoryRequest,
  WebhookStatus,
  WebhookResult,
  WebhookSecretResponse,
} from "./repository";

// User Repository Provider (Personal Settings)
export { userRepositoryProviderApi } from "./user-repository-provider";
export type {
  RepositoryProviderData,
  RepositoryData as UserRemoteRepositoryData,
  CreateRepositoryProviderRequest,
  UpdateRepositoryProviderRequest,
} from "./user-repository-provider";

// User Git Credential (Personal Settings)
export { userGitCredentialApi, CredentialType, getCredentialTypeLabel, isRunnerLocalCredential } from "./user-git-credential";
export type {
  CredentialTypeValue,
  GitCredentialData,
  RunnerLocalCredentialData,
  CreateGitCredentialRequest,
  UpdateGitCredentialRequest,
  SetDefaultRequest,
} from "./user-git-credential";

// User Agent Credential (Personal Settings - Agent API credentials)
export { userAgentCredentialApi, isRunnerHostProfile, getProfileStatusLabel } from "./user-agent-credential";
export type {
  CredentialProfileData,
  CredentialProfilesByAgentType,
  CreateCredentialProfileRequest,
  UpdateCredentialProfileRequest,
  RunnerHostInfo,
} from "./user-agent-credential";

// Binding
export { bindingApi } from "./binding";
export type { PodBinding } from "./binding";

// Mesh
export { meshApi } from "./mesh";
export type {
  MeshNodeData,
  MeshEdgeData,
  ChannelInfoData,
  RunnerInfoData,
  MeshTopologyData,
} from "./mesh";

// Message
export { messageApi } from "./message";
export type { AgentMessage, DeadLetterEntry } from "./message";

// Billing
export { billingApi } from "./billing";
export type {
  SubscriptionPlan,
  UsageOverview,
  BillingOverview,
  Subscription,
  OrderType,
  BillingCycle,
  PaymentProvider,
  CheckoutRequest,
  CheckoutResponse,
  CheckoutStatus,
  SeatUsage,
  Invoice,
  DeploymentInfo,
} from "./billing";

// AgentPod
export { agentpodApi } from "./agentpod";
export type {
  AIProviderType,
  UserAgentPodSettings,
  UserAIProvider,
  UpdateSettingsRequest,
  CreateProviderRequest,
  UpdateProviderRequest,
} from "./agentpod";

// Invitation
export { invitationApi } from "./invitation";
export type {
  Invitation,
  InvitationInfo,
  PendingInvitation,
} from "./invitation";

// Promo Code
export { promoCodeApi } from "./promocode";
export type {
  PromoCodeType,
  ValidatePromoCodeResponse,
  RedeemPromoCodeResponse,
  PromoCodeRedemption,
} from "./promocode";

// AutopilotController
export { autopilotApi } from "./autopilot";

// API Key
export { apiKeyApi } from "./apikey";
export type { APIKeyData, CreateAPIKeyRequest, UpdateAPIKeyRequest } from "./apikey";
export type {
  AutopilotPhase,
  CircuitBreakerState,
  AutopilotControllerData,
  AutopilotIterationData,
  CreateAutopilotControllerRequest,
  ApproveRequest,
} from "./autopilot";
