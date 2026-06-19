// renderer cache (snake_case JSON, nested circuit_breaker) → state proto bytes.
// Mirror image of the wasm controllers_bytes/etc. readers so the shared web
// selectors decode desktop and web identically (fromBinary(StateSchema) →
// controllerSnapshotToCache). Re-flattens circuit_breaker.{state,reason} into the
// snapshot's circuit_breaker_state/reason. Field names MUST match what
// autopilotSnapshotToCache.controllerSnapshotToCache reads.
import { create, toBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedControllersRequestSchema,
  ReplaceCachedIterationsRequestSchema,
  SetCurrentControllerRequestSchema,
  AutopilotControllerSnapshotSchema,
  AutopilotIterationSnapshotSchema,
  type AutopilotControllerSnapshot,
  type AutopilotIterationSnapshot,
} from "@agentsmesh/proto/autopilot_state/v1/autopilot_state_pb";

type Obj = Record<string, unknown>;
const ACTIVE = ["initializing", "running", "paused", "user_takeover", "waiting_approval"];

const optBig = (v: unknown): bigint | undefined =>
  v === undefined || v === null ? undefined : BigInt(v as number);
const optStr = (v: unknown): string | undefined =>
  typeof v === "string" && v.length > 0 ? v : undefined;

function cacheToControllerSnapshot(c: Obj): AutopilotControllerSnapshot {
  const cb = c.circuit_breaker as Obj | undefined;
  return create(AutopilotControllerSnapshotSchema, {
    autopilotControllerKey: (c.autopilot_controller_key as string) ?? "",
    podKey: (c.pod_key as string) ?? "",
    status: optStr(c.status),
    phase: optStr(c.phase),
    prompt: optStr(c.prompt),
    maxIterations: optBig(c.max_iterations),
    iterationTimeoutSec: optBig(c.iteration_timeout_sec),
    noProgressThreshold: optBig(c.no_progress_threshold),
    sameErrorThreshold: optBig(c.same_error_threshold),
    approvalTimeoutMin: optBig(c.approval_timeout_min),
    currentIteration: optBig(c.current_iteration),
    controlAgentSlug: optStr(c.control_agent_slug),
    circuitBreakerState: optStr(cb?.state),
    circuitBreakerReason: optStr(cb?.reason),
    createdAt: optStr(c.created_at),
    updatedAt: optStr(c.updated_at),
  });
}

function cacheToIterationSnapshot(i: Obj): AutopilotIterationSnapshot {
  return create(AutopilotIterationSnapshotSchema, {
    id: BigInt((i.id as number) ?? 0),
    controllerKey: (i.controller_key as string) ?? "",
    iterationNumber: optBig(i.iteration ?? i.iteration_number),
    status: optStr(i.phase ?? i.status),
    result: optStr(i.summary ?? i.result),
    startedAt: optStr(i.created_at ?? i.started_at),
    completedAt: optStr(i.completed_at),
  });
}

export function controllersBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as Obj[];
  return toBinary(ReplaceCachedControllersRequestSchema,
    create(ReplaceCachedControllersRequestSchema, { controllers: list.map(cacheToControllerSnapshot) }));
}

export function currentControllerBytes(cacheJson: string | null): Uint8Array {
  if (!cacheJson) return new Uint8Array();
  const c = JSON.parse(cacheJson) as Obj;
  return toBinary(SetCurrentControllerRequestSchema,
    create(SetCurrentControllerRequestSchema, { controller: cacheToControllerSnapshot(c) }));
}

export function controllerByPodKeyBytes(cacheJson: string, podKey: string): Uint8Array {
  const list = JSON.parse(cacheJson) as Obj[];
  const c = list.find((x) =>
    x.pod_key === podKey && ACTIVE.includes((x.phase as string) ?? ""));
  if (!c) return new Uint8Array();
  return toBinary(SetCurrentControllerRequestSchema,
    create(SetCurrentControllerRequestSchema, { controller: cacheToControllerSnapshot(c) }));
}

export function iterationsBytes(key: string, itersJson: string): Uint8Array {
  const iters = JSON.parse(itersJson) as Obj[];
  return toBinary(ReplaceCachedIterationsRequestSchema,
    create(ReplaceCachedIterationsRequestSchema, {
      autopilotControllerKey: key, iterations: iters.map(cacheToIterationSnapshot),
    }));
}
