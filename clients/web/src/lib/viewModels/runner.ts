// Runner view-model types moved to the zero-dep @agentsmesh/service-interface
// contract layer so the web fromProtoRunner projection and the desktop
// electron-adapter projection share one definition. Re-exported here to
// preserve existing `@/lib/viewModels/runner` import paths.
export type {
  RunnerData,
  RelayConnectionInfo,
  GRPCRegistrationToken,
  RunnerListResponse,
  RunnerDetailResponse,
  RunnerLogData,
  RunnerPodData,
  SandboxStatus,
  RunnerAuthStatus,
  RunnerAuthorizeResponse,
} from "@agentsmesh/service-interface";
