// Loop view-model types moved to the zero-dep @agentsmesh/service-interface
// contract layer so the web fromProtoLoop projection and the desktop
// electron-adapter projection share one definition. Re-exported here to
// preserve existing `@/lib/viewModels/loop` import paths.
export type {
  LoopStatus,
  ExecutionMode,
  SandboxStrategy,
  ConcurrencyPolicy,
  RunStatus,
  LoopData,
  LoopRunData,
  CreateLoopRequest,
  UpdateLoopRequest,
} from "@agentsmesh/service-interface";
