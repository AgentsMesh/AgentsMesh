// renderer cache (snake_case PodData JSON) → state proto bytes. Mirror image of
// the wasm `pods_bytes()` / `get_pod_bytes()` readers so the shared web
// selectors decode desktop and web identically (fromBinary + podToCache). The
// wire Pod IS the cache Pod (proto.pod.v1.Pod), so this round-trip must carry
// EVERY field podToCache reads — field names here mirror web's podToProtoPod.
import { create, toBinary } from "@bufbuild/protobuf";
import {
  PodSchema,
  PodRunnerInfoSchema, PodAgentInfoSchema, PodRepositoryInfoSchema,
  PodTicketInfoSchema, PodLoopInfoSchema, PodCreatedByInfoSchema,
} from "@agentsmesh/proto/pod/v1/pod_pb";
import { ReplaceCachedPodsRequestSchema } from "@agentsmesh/proto/pod_state/v1/pod_state_pb";

type Obj = Record<string, unknown>;
const str = (v: unknown): string | undefined => (v == null ? undefined : (v as string));
const bint = (v: unknown): bigint | undefined => (v == null ? undefined : BigInt(v as number));
const sub = (v: unknown): Obj | undefined => (v as Obj | undefined);

export function cacheToStatePod(p: Obj) {
  const runner = sub(p.runner);
  const agent = sub(p.agent);
  const repo = sub(p.repository);
  const ticket = sub(p.ticket);
  const loop = sub(p.loop);
  const createdBy = sub(p.created_by);
  return create(PodSchema, {
    id: bint(p.id) ?? BigInt(0),
    podKey: str(p.pod_key) ?? "",
    status: str(p.status) ?? "",
    agentStatus: str(p.agent_status),
    alias: str(p.alias),
    title: str(p.title),
    runner: runner
      ? create(PodRunnerInfoSchema, { id: bint(runner.id), nodeId: str(runner.node_id), status: str(runner.status) })
      : undefined,
    agent: agent ? create(PodAgentInfoSchema, { name: str(agent.name), slug: str(agent.slug) }) : undefined,
    repository: repo
      ? create(PodRepositoryInfoSchema, {
          id: bint(repo.id), name: str(repo.name), slug: str(repo.slug), providerType: str(repo.provider_type),
        })
      : undefined,
    ticket: ticket
      ? create(PodTicketInfoSchema, { id: bint(ticket.id), slug: str(ticket.slug), title: str(ticket.title) })
      : undefined,
    loop: loop
      ? create(PodLoopInfoSchema, { id: bint(loop.id), name: str(loop.name), slug: str(loop.slug) })
      : undefined,
    createdBy: createdBy
      ? create(PodCreatedByInfoSchema, { id: bint(createdBy.id), username: str(createdBy.username), name: str(createdBy.name) })
      : undefined,
    prompt: str(p.prompt),
    branchName: str(p.branch_name),
    sandboxPath: str(p.sandbox_path),
    interactionMode: str(p.interaction_mode),
    perpetual: p.perpetual as boolean | undefined,
    restartCount: p.restart_count as number | undefined,
    lastRestartAt: str(p.last_restart_at),
    startedAt: str(p.started_at),
    finishedAt: str(p.finished_at),
    lastActivity: str(p.last_activity),
    createdAt: str(p.created_at),
    errorCode: str(p.error_code),
    errorMessage: str(p.error_message),
    resumedByPodKey: str(p.resumed_by_pod_key),
  });
}

export function podsBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as Obj[];
  return toBinary(ReplaceCachedPodsRequestSchema,
    create(ReplaceCachedPodsRequestSchema, { pods: list.map(cacheToStatePod) }));
}

function findPod(cacheJson: string, podKey: string): Obj | undefined {
  return (JSON.parse(cacheJson) as Obj[]).find((p) => (p.pod_key as string) === podKey);
}

export function podBytes(cacheJson: string, podKey: string): Uint8Array {
  const p = findPod(cacheJson, podKey);
  if (!p) return new Uint8Array();
  return toBinary(PodSchema, cacheToStatePod(p));
}

export function currentPodBytes(currentJson: string | null): Uint8Array {
  if (!currentJson) return new Uint8Array();
  return toBinary(PodSchema, cacheToStatePod(JSON.parse(currentJson) as Obj));
}
