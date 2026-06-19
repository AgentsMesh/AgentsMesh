// Connect-RPC adapter for proto.loop.v1.LoopService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out — conventions §2.5), decodes
// responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing LoopData / LoopRunData shapes (viewModels/loop.ts)
// so call-sites in the loop store don't need to flip off camelCase + BigInt.

import {
  CancelRunRequestSchema,
  CancelRunResponseSchema,
  CreateLoopRequestSchema,
  DeleteLoopRequestSchema,
  DeleteLoopResponseSchema,
  EnvBundleListSchema,
  GetLoopRequestSchema,
  ListLoopsRequestSchema,
  ListLoopsResponseSchema,
  ListRunsRequestSchema,
  ListRunsResponseSchema,
  LoopActionRequestSchema,
  LoopSchema,
  TriggerLoopRequestSchema,
  TriggerLoopResponseSchema,
  UpdateLoopRequestSchema,
} from "@proto/loop/v1/loop_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
// Shared proto→LoopData projection — single SSOT, also consumed by the desktop
// electron-adapter. Aliased to the historical fromProto* names used below.
import { loopToCache as fromProtoLoop, loopRunToCache as fromProtoLoopRun } from "@agentsmesh/electron-adapter/projections";
import { getLoopService } from "@/lib/wasm-core";
import type {
  CreateLoopRequest,
  LoopData,
  LoopRunData,
  UpdateLoopRequest,
} from "@/lib/viewModels/loop";

interface ListFilters {
  status?: string;
  executionMode?: string;
  cronEnabled?: boolean;
  query?: string;
  limit?: number;
  offset?: number;
}

export async function listLoops(
  orgSlug: string,
  filters?: ListFilters,
): Promise<{ items: LoopData[]; total: number }> {
  const req = create(ListLoopsRequestSchema, {
    orgSlug,
    status: filters?.status ?? "",
    executionMode: filters?.executionMode ?? "",
    cronEnabled: filters?.cronEnabled,
    query: filters?.query ?? "",
    offset: filters?.offset,
    limit: filters?.limit,
  });
  const bytes = toBinary(ListLoopsRequestSchema, req);
  const respBytes = await getLoopService().listLoopsConnect(bytes);
  const resp = fromBinary(ListLoopsResponseSchema, new Uint8Array(respBytes));
  return { items: resp.items.map(fromProtoLoop), total: Number(resp.total) };
}

// Raw wire bytes for the fetch→state path: response → apply_fetched_loops
// (Rust set_loops via loop_from_proto), no TS fromProtoLoop/loopToProtoLoop.
export async function listLoopsRaw(orgSlug: string, filters?: ListFilters): Promise<Uint8Array> {
  const req = create(ListLoopsRequestSchema, {
    orgSlug,
    status: filters?.status ?? "",
    executionMode: filters?.executionMode ?? "",
    cronEnabled: filters?.cronEnabled,
    query: filters?.query ?? "",
    offset: filters?.offset,
    limit: filters?.limit,
  });
  return new Uint8Array(await getLoopService().listLoopsConnect(toBinary(ListLoopsRequestSchema, req)));
}

export async function getLoop(orgSlug: string, loopSlug: string): Promise<LoopData> {
  const req = create(GetLoopRequestSchema, { orgSlug, loopSlug });
  const bytes = toBinary(GetLoopRequestSchema, req);
  const respBytes = await getLoopService().getLoopConnect(bytes);
  return fromProtoLoop(fromBinary(LoopSchema, new Uint8Array(respBytes)));
}

// Raw wire bytes for the fetch→state path: response (Loop) →
// apply_fetched_current_loop (Rust set_current_loop). The full wire Loop carries
// the proto-only fields the lossy loopToProtoLoop round-trip dropped.
export async function getLoopRaw(orgSlug: string, loopSlug: string): Promise<Uint8Array> {
  const req = create(GetLoopRequestSchema, { orgSlug, loopSlug });
  return new Uint8Array(
    await getLoopService().getLoopConnect(toBinary(GetLoopRequestSchema, req)),
  );
}

function toJsonString(v: unknown): string {
  if (v === undefined || v === null) return "";
  if (typeof v === "string") return v;
  return JSON.stringify(v);
}

export async function createLoop(orgSlug: string, data: CreateLoopRequest): Promise<LoopData> {
  const req = create(CreateLoopRequestSchema, {
    orgSlug,
    name: data.name,
    slug: data.slug ?? "",
    description: data.description ?? "",
    agentSlug: data.agent_slug ?? "",
    permissionMode: data.permission_mode ?? "",
    promptTemplate: data.prompt_template,
    promptVariablesJson: toJsonString(data.prompt_variables),
    configOverridesJson: toJsonString(data.config_overrides),
    autopilotConfigJson: toJsonString(data.autopilot_config),
    repositoryId: data.repository_id != null ? BigInt(data.repository_id) : undefined,
    runnerId: data.runner_id != null ? BigInt(data.runner_id) : undefined,
    branchName: data.branch_name ?? "",
    ticketId: data.ticket_id != null ? BigInt(data.ticket_id) : undefined,
    credentialProfileId: data.credential_profile_id != null ? BigInt(data.credential_profile_id) : undefined,
    executionMode: data.execution_mode ?? "",
    cronExpression: data.cron_expression ?? "",
    callbackUrl: data.callback_url ?? "",
    sandboxStrategy: data.sandbox_strategy ?? "",
    sessionPersistence: data.session_persistence,
    concurrencyPolicy: data.concurrency_policy ?? "",
    maxConcurrentRuns: data.max_concurrent_runs,
    maxRetainedRuns: data.max_retained_runs,
    timeoutMinutes: data.timeout_minutes,
    usedEnvBundles: data.used_env_bundles ?? [],
  });
  const bytes = toBinary(CreateLoopRequestSchema, req);
  const respBytes = await getLoopService().createLoopConnect(bytes);
  return fromProtoLoop(fromBinary(LoopSchema, new Uint8Array(respBytes)));
}

export async function updateLoop(
  orgSlug: string,
  loopSlug: string,
  data: UpdateLoopRequest,
): Promise<LoopData> {
  const req = create(UpdateLoopRequestSchema, {
    orgSlug,
    loopSlug,
    name: data.name,
    description: data.description,
    agentSlug: data.agent_slug ?? "",
    permissionMode: data.permission_mode,
    promptTemplate: data.prompt_template,
    promptVariablesJson: toJsonString(data.prompt_variables),
    configOverridesJson: toJsonString(data.config_overrides),
    autopilotConfigJson: toJsonString(data.autopilot_config),
    repositoryId: data.repository_id != null ? BigInt(data.repository_id) : undefined,
    runnerId: data.runner_id != null ? BigInt(data.runner_id) : undefined,
    branchName: data.branch_name,
    ticketId: data.ticket_id != null ? BigInt(data.ticket_id) : undefined,
    credentialProfileId: data.credential_profile_id != null ? BigInt(data.credential_profile_id) : undefined,
    executionMode: data.execution_mode,
    cronExpression: data.cron_expression,
    callbackUrl: data.callback_url,
    sandboxStrategy: data.sandbox_strategy,
    sessionPersistence: data.session_persistence,
    concurrencyPolicy: data.concurrency_policy,
    maxConcurrentRuns: data.max_concurrent_runs,
    maxRetainedRuns: data.max_retained_runs,
    timeoutMinutes: data.timeout_minutes,
    usedEnvBundles:
      data.used_env_bundles !== undefined
        ? create(EnvBundleListSchema, { names: data.used_env_bundles ?? [] })
        : undefined,
  });
  const bytes = toBinary(UpdateLoopRequestSchema, req);
  const respBytes = await getLoopService().updateLoopConnect(bytes);
  return fromProtoLoop(fromBinary(LoopSchema, new Uint8Array(respBytes)));
}

export async function deleteLoop(orgSlug: string, loopSlug: string): Promise<void> {
  const req = create(DeleteLoopRequestSchema, { orgSlug, loopSlug });
  const bytes = toBinary(DeleteLoopRequestSchema, req);
  const respBytes = await getLoopService().deleteLoopConnect(bytes);
  fromBinary(DeleteLoopResponseSchema, new Uint8Array(respBytes));
}

async function loopAction(
  caller: (b: Uint8Array) => Promise<Uint8Array>,
  orgSlug: string,
  loopSlug: string,
): Promise<LoopData> {
  const req = create(LoopActionRequestSchema, { orgSlug, loopSlug });
  const bytes = toBinary(LoopActionRequestSchema, req);
  const respBytes = await caller(bytes);
  return fromProtoLoop(fromBinary(LoopSchema, new Uint8Array(respBytes)));
}

export async function enableLoop(orgSlug: string, loopSlug: string): Promise<LoopData> {
  return loopAction((b) => getLoopService().enableLoopConnect(b), orgSlug, loopSlug);
}

export async function disableLoop(orgSlug: string, loopSlug: string): Promise<LoopData> {
  return loopAction((b) => getLoopService().disableLoopConnect(b), orgSlug, loopSlug);
}

export interface TriggerLoopResult {
  run?: LoopRunData;
  skipped?: boolean;
  reason?: string;
}

export async function triggerLoop(
  orgSlug: string,
  loopSlug: string,
  variables?: Record<string, unknown>,
): Promise<TriggerLoopResult> {
  const req = create(TriggerLoopRequestSchema, {
    orgSlug,
    loopSlug,
    variablesJson: variables ? JSON.stringify(variables) : "",
  });
  const bytes = toBinary(TriggerLoopRequestSchema, req);
  const respBytes = await getLoopService().triggerLoopConnect(bytes);
  const resp = fromBinary(TriggerLoopResponseSchema, new Uint8Array(respBytes));
  if (resp.skipped) {
    return { skipped: true, reason: resp.reason };
  }
  return resp.run ? { run: fromProtoLoopRun(resp.run) } : {};
}

export async function listLoopRuns(
  orgSlug: string,
  loopSlug: string,
  filters?: { status?: string; limit?: number; offset?: number },
): Promise<{ items: LoopRunData[]; total: number }> {
  const req = create(ListRunsRequestSchema, {
    orgSlug,
    loopSlug,
    status: filters?.status ?? "",
    offset: filters?.offset,
    limit: filters?.limit,
  });
  const bytes = toBinary(ListRunsRequestSchema, req);
  const respBytes = await getLoopService().listRunsConnect(bytes);
  const resp = fromBinary(ListRunsResponseSchema, new Uint8Array(respBytes));
  return { items: resp.items.map(fromProtoLoopRun), total: Number(resp.total) };
}

// Raw wire bytes for the runs fetch→state path: response → apply_fetched_runs /
// apply_appended_runs (Rust set_runs/append_runs via run_from_proto).
export async function listRunsRaw(
  orgSlug: string,
  loopSlug: string,
  filters?: { status?: string; limit?: number; offset?: number },
): Promise<Uint8Array> {
  const req = create(ListRunsRequestSchema, {
    orgSlug, loopSlug, status: filters?.status ?? "", offset: filters?.offset, limit: filters?.limit,
  });
  return new Uint8Array(await getLoopService().listRunsConnect(toBinary(ListRunsRequestSchema, req)));
}

export async function cancelLoopRun(
  orgSlug: string,
  loopSlug: string,
  runId: number,
): Promise<void> {
  const req = create(CancelRunRequestSchema, { orgSlug, loopSlug, runId: BigInt(runId) });
  const bytes = toBinary(CancelRunRequestSchema, req);
  const respBytes = await getLoopService().cancelRunConnect(bytes);
  fromBinary(CancelRunResponseSchema, new Uint8Array(respBytes));
}
