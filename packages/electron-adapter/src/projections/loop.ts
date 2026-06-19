import type { Loop as ProtoLoop, LoopRun as ProtoLoopRun } from "@agentsmesh/proto/loop/v1/loop_pb";
import type { LoopData, LoopRunData } from "@agentsmesh/service-interface";
import { parseJSONObject } from "./parse-json";

// Single source of truth for the proto.loop.v1 → LoopData projection, shared by
// web (fromProtoLoop in loopConnect.ts) and desktop (ElectronLoopService). One
// copy means a new proto field cannot silently drop on one side — the drift
// that crashed the desktop loop-detail ConfigPanel.

export function loopToCache(p: ProtoLoop): LoopData {
  return {
    id: Number(p.id), organization_id: 0, name: p.name, slug: p.slug,
    description: p.description || undefined,
    agent_slug: p.agentSlug || undefined,
    permission_mode: p.permissionMode,
    prompt_template: p.promptTemplate,
    prompt_variables: parseJSONObject(p.promptVariablesJson),
    repository_id: p.repositoryId != null ? Number(p.repositoryId) : undefined,
    runner_id: p.runnerId != null ? Number(p.runnerId) : undefined,
    branch_name: p.branchName || undefined,
    ticket_id: p.ticketId != null ? Number(p.ticketId) : undefined,
    credential_profile_id: p.credentialProfileId != null ? Number(p.credentialProfileId) : undefined,
    used_env_bundles: p.usedEnvBundles ?? [],
    config_overrides: parseJSONObject(p.configOverridesJson),
    execution_mode: p.executionMode as LoopData["execution_mode"],
    cron_expression: p.cronExpression || undefined,
    callback_url: p.callbackUrl || undefined,
    autopilot_config: parseJSONObject(p.autopilotConfigJson) ?? {},
    status: p.status as LoopData["status"],
    sandbox_strategy: p.sandboxStrategy as LoopData["sandbox_strategy"],
    session_persistence: p.sessionPersistence,
    concurrency_policy: p.concurrencyPolicy as LoopData["concurrency_policy"],
    max_concurrent_runs: p.maxConcurrentRuns,
    max_retained_runs: p.maxRetainedRuns,
    timeout_minutes: p.timeoutMinutes,
    created_by_id: 0,
    total_runs: Number(p.totalRuns), successful_runs: Number(p.successfulRuns),
    failed_runs: Number(p.failedRuns), active_run_count: Number(p.activeRunCount),
    avg_duration_sec: p.avgDurationSec ?? undefined,
    last_run_at: p.lastRunAt, created_at: p.createdAt, updated_at: p.updatedAt,
  };
}

export function loopRunToCache(p: ProtoLoopRun): LoopRunData {
  return {
    id: Number(p.id), organization_id: 0, loop_id: Number(p.loopId),
    run_number: Number(p.runNumber), status: p.status as LoopRunData["status"],
    pod_key: p.podKey, trigger_type: "", started_at: p.startedAt,
    // proto LoopRun.completed_at maps to the viewModel's finished_at.
    finished_at: p.completedAt,
    error_message: p.errorMessage, created_at: p.createdAt, updated_at: p.createdAt,
  };
}
