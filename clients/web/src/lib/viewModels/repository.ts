// Repository view-model types moved to the zero-dep @agentsmesh/service-interface
// contract layer so the web fromProtoRepository projection and the desktop
// electron-adapter projection share one definition. Re-exported here to
// preserve existing `@/lib/viewModels/repository` import paths.
export type {
  WebhookStatus,
  WebhookResult,
  WebhookSecretResponse,
  RepositoryData,
  CreateRepositoryRequest,
  UpdateRepositoryRequest,
} from "@agentsmesh/service-interface";
