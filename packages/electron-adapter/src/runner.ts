import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { IRunnerService } from "@agentsmesh/service-interface";
import { fromBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedRunnersRequestSchema,
  ReplaceAvailableRunnersRequestSchema,
  SetCurrentRunnerRequestSchema,
  PatchCachedRunnerRequestSchema,
  RemoveCachedRunnerRequestSchema,
} from "@agentsmesh/proto/runner_state/v1/runner_state_pb";
import type { Runner as ProtoRunner } from "@agentsmesh/proto/runner_api/v1/runner_pb";

// ProtoRunner (camelCase + BigInt) → JS-cache shape (snake_case + number).
// Inlined to keep electron-adapter independent of the web/src tree.
function runnerToCache(r: ProtoRunner): Record<string, unknown> {
  const ZERO = BigInt(0);
  const host: Record<string, unknown> | undefined = r.hostInfoJson
    ? (() => { try { return JSON.parse(r.hostInfoJson); } catch { return undefined; } })()
    : undefined;
  return {
    id: Number(r.id),
    node_id: r.nodeId,
    description: r.description || undefined,
    status: r.status,
    last_heartbeat: r.lastHeartbeat,
    current_pods: r.currentPods,
    max_concurrent_pods: r.maxConcurrentPods,
    runner_version: r.runnerVersion,
    is_enabled: r.isEnabled,
    visibility: r.visibility,
    registered_by_user_id: r.registeredByUserId !== undefined && r.registeredByUserId !== ZERO
      ? Number(r.registeredByUserId) : undefined,
    host_info: host,
    available_agents: r.availableAgents?.length ? r.availableAgents : undefined,
    tags: r.tags?.length ? r.tags : undefined,
    created_at: r.createdAt,
    updated_at: r.updatedAt,
  };
}

export class ElectronRunnerService implements IRunnerService {
  private _runnersCache = "[]";
  private _availableRunnersCache = "[]";
  private _currentRunnerCache: string | null = null;

  runners_json(): string { return this._runnersCache; }
  available_runners_json(): string { return this._availableRunnersCache; }
  current_runner_json(): unknown { return this._currentRunnerCache; }

  get_runner_json(id: bigint): unknown {
    const runners = JSON.parse(this._runnersCache) as { id: number }[];
    const r = runners.find(x => x.id === Number(id));
    return r ? JSON.stringify(r) : null;
  }

  // Proto-bytes mutators — decode locally + update JS cache synchronously,
  // then fire-and-forget NAPI sync. Mirrors ElectronChannelService /
  // ElectronPodService pattern (renderer reads via runners_json() etc.
  // immediately after dispatch; IPC roundtrip would defeat that invariant).

  replace_cached_runners(reqBytes: Uint8Array): void {
    const req = fromBinary(ReplaceCachedRunnersRequestSchema, reqBytes);
    this._runnersCache = JSON.stringify(req.runners.map(runnerToCache));
    void invoke<void>("appRunnerReplaceCached", Array.from(reqBytes)).catch(() => undefined);
  }

  replace_available_runners(reqBytes: Uint8Array): void {
    const req = fromBinary(ReplaceAvailableRunnersRequestSchema, reqBytes);
    this._availableRunnersCache = JSON.stringify(req.runners.map(runnerToCache));
    void invoke<void>("appRunnerReplaceAvailable", Array.from(reqBytes)).catch(() => undefined);
  }

  set_current_runner_proto(reqBytes: Uint8Array): void {
    const req = fromBinary(SetCurrentRunnerRequestSchema, reqBytes);
    this._currentRunnerCache = req.runner ? JSON.stringify(runnerToCache(req.runner)) : null;
    void invoke<void>("appRunnerSetCurrent", Array.from(reqBytes)).catch(() => undefined);
  }

  patch_cached_runner(reqBytes: Uint8Array): void {
    const req = fromBinary(PatchCachedRunnerRequestSchema, reqBytes);
    if (req.runner) {
      const patch = runnerToCache(req.runner);
      const list = JSON.parse(this._runnersCache) as { id: number }[];
      const idx = list.findIndex((x) => x.id === patch.id);
      if (idx >= 0) list[idx] = { ...list[idx], ...patch };
      else list.push(patch as { id: number });
      this._runnersCache = JSON.stringify(list);
    }
    void invoke<void>("appRunnerPatch", Array.from(reqBytes)).catch(() => undefined);
  }

  remove_cached_runner(reqBytes: Uint8Array): void {
    const req = fromBinary(RemoveCachedRunnerRequestSchema, reqBytes);
    const list = JSON.parse(this._runnersCache) as { id: number }[];
    this._runnersCache = JSON.stringify(list.filter((x) => x.id !== Number(req.runnerId)));
    void invoke<void>("appRunnerRemove", Array.from(reqBytes)).catch(() => undefined);
  }

  // Surgical realtime mirror: the main-pushed snapshot carries the Rust-computed
  // runner lists (runners + available + current). Replace all three local caches
  // so runners_json()/available_runners_json()/current_runner_json() reflect the
  // SSOT. Empty string for current clears it.
  apply_runners_snapshot(runnersJson: string, availableJson: string, currentJson: string): void {
    if (runnersJson) this._runnersCache = runnersJson;
    if (availableJson) this._availableRunnersCache = availableJson;
    this._currentRunnerCache = currentJson ? currentJson : null;
  }

  update_runner_status(id: bigint, status: string): void {
    const runners = JSON.parse(this._runnersCache) as { id: number; status?: string }[];
    const r = runners.find(x => x.id === Number(id));
    if (r) r.status = status;
    this._runnersCache = JSON.stringify(runners);
  }

  async fetch_runners(status?: string | null): Promise<string> {
    const result = await invoke<string>("runnerFetchRunners", status);
    const parsed = JSON.parse(result);
    this._runnersCache = JSON.stringify(parsed.runners ?? []);
    return result;
  }

  async fetch_runner(id: bigint): Promise<string> {
    const result = await invoke<string>("runnerFetchRunner", Number(id));
    this._currentRunnerCache = result;
    return result;
  }

  async fetch_available_runners(): Promise<string> {
    const result = await invoke<string>("runnerFetchAvailableRunners");
    const parsed = JSON.parse(result);
    this._availableRunnersCache = JSON.stringify(parsed.runners ?? []);
    return result;
  }

  async create_token(json: string): Promise<string> {
    return invoke<string>("runnerCreateToken", json);
  }

  async fetch_tokens(): Promise<string> {
    return invoke<string>("runnerFetchTokens");
  }

  async delete_token(id: bigint): Promise<void> {
    await invoke<void>("runnerDeleteToken", Number(id));
  }

  async delete_runner(id: bigint): Promise<void> {
    await invoke<void>("runnerDeleteRunner", Number(id));
    // Note: caller is the store; store also dispatches RemoveCachedRunnerRequest separately
  }

  async upgrade_runner(id: bigint, json: string): Promise<string> {
    return invoke<string>("runnerUpgradeRunner", Number(id), json);
  }

  async authorize_runner(reqBytes: Uint8Array): Promise<Uint8Array> {
    const result = await invoke<number[] | Uint8Array>(
      "runnerAuthorizeRunner",
      Array.from(reqBytes),
    );
    return coerceConnectResponse(result);
  }

  async get_auth_status(reqBytes: Uint8Array): Promise<Uint8Array> {
    const result = await invoke<number[] | Uint8Array>(
      "runnerGetAuthStatus",
      Array.from(reqBytes),
    );
    return coerceConnectResponse(result);
  }

  async list_runner_logs(id: bigint): Promise<string> {
    return invoke<string>("runnerListRunnerLogs", Number(id));
  }

  async query_runner_sandboxes(id: bigint, json: string): Promise<string> {
    return invoke<string>("runnerQueryRunnerSandboxes", Number(id), json);
  }

  async request_log_upload(id: bigint): Promise<void> {
    await invoke<void>("runnerRequestLogUpload", Number(id));
  }
}
