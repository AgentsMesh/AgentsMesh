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
import {
  ListRunnersResponseSchema,
  ListAvailableRunnersResponseSchema,
  GetRunnerResponseSchema,
} from "@agentsmesh/proto/runner_api/v1/runner_pb";
import { runnerToCache } from "./projections/runner";
import { runnersBytes, availableRunnersBytes, currentRunnerBytes } from "./runner_cache_to_bytes";

export class ElectronRunnerService implements IRunnerService {
  private _runnersCache = "[]";
  private _availableRunnersCache = "[]";
  private _currentRunnerCache: string | null = null;

  runners_json(): string { return this._runnersCache; }
  available_runners_json(): string { return this._availableRunnersCache; }
  current_runner_json(): unknown { return this._currentRunnerCache; }

  // Fetch→state (B): wire Runner == cache Runner; decode wire response →
  // renderer cache (sync) + fan SAME wire bytes to main (runtime.state SSOT).
  apply_fetched_runners(respBytes: Uint8Array): void {
    const resp = fromBinary(ListRunnersResponseSchema, respBytes);
    this._runnersCache = JSON.stringify(resp.items.map(runnerToCache));
    void invoke<void>("appRunnerApplyFetched", Array.from(respBytes)).catch(() => undefined);
  }

  apply_fetched_available_runners(respBytes: Uint8Array): void {
    const resp = fromBinary(ListAvailableRunnersResponseSchema, respBytes);
    this._availableRunnersCache = JSON.stringify(resp.items.map(runnerToCache));
    void invoke<void>("appRunnerApplyFetchedAvailable", Array.from(respBytes)).catch(() => undefined);
  }

  // Single-object fetch (B): decode wire GetRunnerResponse → current cache +
  // fan the SAME wire bytes to main (runtime.state SSOT).
  apply_fetched_current_runner(respBytes: Uint8Array): void {
    const resp = fromBinary(GetRunnerResponseSchema, respBytes);
    this._currentRunnerCache = resp.runner ? JSON.stringify(runnerToCache(resp.runner)) : null;
    void invoke<void>("appRunnerApplyFetchedCurrent", Array.from(respBytes)).catch(() => undefined);
  }

  // Read side (B, zero-JSON): re-encode renderer cache into state proto bytes.
  runners_bytes(): Uint8Array { return runnersBytes(this._runnersCache); }
  available_runners_bytes(): Uint8Array { return availableRunnersBytes(this._availableRunnersCache); }
  current_runner_bytes(): Uint8Array { return currentRunnerBytes(this._currentRunnerCache); }

  get_runner_json(id: bigint): unknown {
    const runners = JSON.parse(this._runnersCache) as { id: number }[];
    const r = runners.find(x => x.id === Number(id));
    return r ? JSON.stringify(r) : null;
  }

  // Proto-bytes mutators — decode locally + update JS cache synchronously,
  // then fire-and-forget NAPI sync. Mirrors ElectronChannelService /
  // ElectronPodService pattern (renderer reads via runners_json() etc.
  // immediately after dispatch; IPC roundtrip would defeat that invariant).

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

  // Surgical realtime mirror: the main-pushed snapshot carries the Rust-encoded
  // runner lists as proto *Request bytes. Decode via fromBinary + runnerToCache
  // (the fetch projection) so all three caches share RunnerData shape with fetch.
  // Always decode — empty bytes mean an empty list, which must clear the cache.
  apply_runners_snapshot(runnersBytes: Uint8Array, availableBytes: Uint8Array, currentBytes: Uint8Array): void {
    const rReq = fromBinary(ReplaceCachedRunnersRequestSchema, runnersBytes);
    this._runnersCache = JSON.stringify(rReq.runners.map(runnerToCache));
    const aReq = fromBinary(ReplaceAvailableRunnersRequestSchema, availableBytes);
    this._availableRunnersCache = JSON.stringify(aReq.runners.map(runnerToCache));
    const cReq = fromBinary(SetCurrentRunnerRequestSchema, currentBytes);
    this._currentRunnerCache = cReq.runner ? JSON.stringify(runnerToCache(cReq.runner)) : null;
  }

  update_runner_status(id: bigint, status: string): void {
    const runners = JSON.parse(this._runnersCache) as { id: number; status?: string }[];
    const r = runners.find(x => x.id === Number(id));
    if (r) r.status = status;
    this._runnersCache = JSON.stringify(runners);
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
