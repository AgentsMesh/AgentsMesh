// renderer cache (snake_case LoopData/LoopRunData JSON) → state proto bytes.
// Inverse of projections/loop.{loopToCache,loopRunToCache}; mirrors the wasm
// loops_bytes()/runs_bytes()/current_loop_bytes() readers so the shared web
// selectors decode desktop and web identically. Desktop cache keeps the full
// proto field set (wire projection), so unlike the Rust path this is lossless.
import { create, toBinary } from "@bufbuild/protobuf";
import {
  LoopSchema, LoopRunSchema,
  type Loop as ProtoLoop, type LoopRun as ProtoLoopRun,
} from "@agentsmesh/proto/loop/v1/loop_pb";
import {
  ReplaceCachedLoopsRequestSchema, ReplaceCachedRunsRequestSchema,
  SetCurrentLoopRequestSchema,
} from "@agentsmesh/proto/loop_state/v1/loop_state_pb";
import type { LoopData, LoopRunData } from "@agentsmesh/service-interface";

const bn = (v: number | undefined | null): bigint | undefined =>
  v === undefined || v === null ? undefined : BigInt(v);
const jstr = (v: unknown): string =>
  v === undefined || v === null ? "" : typeof v === "string" ? v : JSON.stringify(v);

export function cacheToProtoLoop(l: LoopData): ProtoLoop {
  return create(LoopSchema, {
    id: bn(l.id) ?? 0n, slug: l.slug, name: l.name, description: l.description ?? "",
    agentSlug: l.agent_slug ?? "", permissionMode: l.permission_mode, promptTemplate: l.prompt_template,
    configOverridesJson: jstr(l.config_overrides), promptVariablesJson: jstr(l.prompt_variables),
    executionMode: l.execution_mode, cronExpression: l.cron_expression ?? "",
    autopilotConfigJson: jstr(l.autopilot_config), callbackUrl: l.callback_url ?? "",
    repositoryId: bn(l.repository_id), runnerId: bn(l.runner_id), branchName: l.branch_name ?? "",
    ticketId: bn(l.ticket_id), credentialProfileId: bn(l.credential_profile_id),
    status: l.status, sandboxStrategy: l.sandbox_strategy, sessionPersistence: l.session_persistence,
    concurrencyPolicy: l.concurrency_policy, maxConcurrentRuns: l.max_concurrent_runs,
    maxRetainedRuns: l.max_retained_runs, timeoutMinutes: l.timeout_minutes,
    totalRuns: BigInt(l.total_runs), successfulRuns: BigInt(l.successful_runs),
    failedRuns: BigInt(l.failed_runs), activeRunCount: BigInt(l.active_run_count),
    avgDurationSec: l.avg_duration_sec, lastRunAt: l.last_run_at,
    createdAt: l.created_at, updatedAt: l.updated_at, usedEnvBundles: l.used_env_bundles ?? [],
  });
}

export function cacheToProtoLoopRun(r: LoopRunData): ProtoLoopRun {
  return create(LoopRunSchema, {
    id: BigInt(r.id), loopId: BigInt(r.loop_id), runNumber: BigInt(r.run_number), status: r.status,
    podKey: r.pod_key, startedAt: r.started_at, completedAt: r.finished_at,
    errorMessage: r.error_message, createdAt: r.created_at,
  });
}

export function loopsBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as LoopData[];
  return toBinary(ReplaceCachedLoopsRequestSchema,
    create(ReplaceCachedLoopsRequestSchema, { loops: list.map(cacheToProtoLoop) }));
}

export function runsBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as LoopRunData[];
  return toBinary(ReplaceCachedRunsRequestSchema,
    create(ReplaceCachedRunsRequestSchema, { runs: list.map(cacheToProtoLoopRun) }));
}

export function currentLoopBytes(currentJson: string | null): Uint8Array {
  if (!currentJson) return new Uint8Array();
  return toBinary(SetCurrentLoopRequestSchema,
    create(SetCurrentLoopRequestSchema, { loop: cacheToProtoLoop(JSON.parse(currentJson) as LoopData) }));
}
