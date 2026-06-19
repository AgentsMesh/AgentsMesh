// renderer cache (snake_case RunnerData JSON) → state proto bytes. Inverse of
// projections/runner.runnerToCache; mirrors the wasm runners_bytes() readers so
// the shared web selectors decode desktop and web identically.
import { create, toBinary } from "@bufbuild/protobuf";
import { RunnerSchema } from "@agentsmesh/proto/runner_api/v1/runner_pb";
import {
  ReplaceCachedRunnersRequestSchema,
  ReplaceAvailableRunnersRequestSchema,
} from "@agentsmesh/proto/runner_state/v1/runner_state_pb";

type Obj = Record<string, unknown>;
const str = (v: unknown): string => (v as string) ?? "";

function cacheToStateRunner(r: Obj) {
  return {
    id: BigInt((r.id as number) ?? 0),
    nodeId: str(r.node_id),
    description: str(r.description),
    status: str(r.status),
    lastHeartbeat: str(r.last_heartbeat),
    currentPods: (r.current_pods as number) ?? 0,
    maxConcurrentPods: (r.max_concurrent_pods as number) ?? 0,
    runnerVersion: str(r.runner_version),
    isEnabled: !!r.is_enabled,
    visibility: str(r.visibility),
    registeredByUserId: r.registered_by_user_id != null ? BigInt(r.registered_by_user_id as number) : undefined,
    hostInfoJson: r.host_info != null ? JSON.stringify(r.host_info) : "",
    availableAgents: (r.available_agents as string[]) ?? [],
    tags: (r.tags as string[]) ?? [],
    createdAt: str(r.created_at),
    updatedAt: str(r.updated_at),
  };
}

export function runnersBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as Obj[];
  return toBinary(ReplaceCachedRunnersRequestSchema,
    create(ReplaceCachedRunnersRequestSchema, { runners: list.map(cacheToStateRunner) }));
}

export function availableRunnersBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as Obj[];
  return toBinary(ReplaceAvailableRunnersRequestSchema,
    create(ReplaceAvailableRunnersRequestSchema, { runners: list.map(cacheToStateRunner) }));
}

export function currentRunnerBytes(currentJson: string | null): Uint8Array {
  if (!currentJson) return new Uint8Array();
  return toBinary(RunnerSchema, create(RunnerSchema, cacheToStateRunner(JSON.parse(currentJson) as Obj)));
}
