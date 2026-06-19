import { invoke } from "./invoke";
import type { IPodService, PodData } from "@agentsmesh/service-interface";
import { fromBinary } from "@bufbuild/protobuf";
import {
  InsertCreatedPodRequestSchema,
  MarkPodTerminatedRequestSchema,
  PatchPodPerpetualRequestSchema,
  ApplyPodStatusEventRequestSchema,
  ApplyPodTitleEventRequestSchema,
  ApplyPodAliasEventRequestSchema,
  ApplyAgentStatusEventRequestSchema,
} from "@agentsmesh/proto/pod_state/v1/pod_state_pb";
import { PodSchema, ListPodsResponseSchema } from "@agentsmesh/proto/pod/v1/pod_pb";
import { podToCache } from "./projections/pod";
import { podsBytes, podBytes, currentPodBytes } from "./pod_cache_to_bytes";

// Apply scalar status patch on a single cached pod. Used by the proto-bytes
// status/title/alias/agent-status mutators below; not exposed publicly.
function patchPodInCache(
  cache: string,
  podKey: string,
  patch: Record<string, unknown>,
): string {
  const pods = JSON.parse(cache) as Array<Record<string, unknown>>;
  const p = pods.find((x) => x.pod_key === podKey);
  if (p) Object.assign(p, patch);
  return JSON.stringify(pods);
}

export class ElectronPodService implements IPodService {
  private _podsCache = "[]";
  private _currentPodCache: string | null = null;

  // ── Read selectors ──

  pods_json(): string { return this._podsCache; }
  current_pod_json(): unknown { return this._currentPodCache; }
  get_pod_json(pod_key: string): unknown {
    const pods = JSON.parse(this._podsCache) as { pod_key: string }[];
    const p = pods.find(x => x.pod_key === pod_key);
    return p ? JSON.stringify(p) : null;
  }

  // Read side (B, zero-JSON): re-encode the renderer cache into state proto
  // bytes so the shared selectors decode desktop and web identically.
  pods_bytes(): Uint8Array { return podsBytes(this._podsCache); }
  current_pod_bytes(): Uint8Array { return currentPodBytes(this._currentPodCache); }
  get_pod_bytes(pod_key: string): Uint8Array { return podBytes(this._podsCache, pod_key); }

  // ── Proto-bytes mutators (mirror clients/core/crates/wasm WasmPodState) ──
  // Decode locally so `pods_json()` reflects the mutation synchronously, AND
  // fire-and-forget the same bytes to the `app_pod_*` commands so the
  // main-process `runtime.state.pods` (the realtime dispatch + snapshot SSOT)
  // gets the same fetch/user-action baseline. Not awaited — IPC latency would
  // defeat the sync-cache invariant the renderer's _tick reactivity assumes.

  // Fetch→state (B): decode wire ListPodsResponse → renderer cache (sync, for
  // reactivity) + fire the SAME wire bytes to main so runtime.state (the SSOT
  // the realtime snapshot reads) folds the identical baseline. Wire Pod IS the
  // cache Pod, so this mirrors the wasm apply_fetched_pods identity exactly.
  apply_fetched_pods(respBytes: Uint8Array): void {
    const resp = fromBinary(ListPodsResponseSchema, respBytes);
    this._podsCache = JSON.stringify(resp.items.map(podToCache));
    void invoke<void>("appPodApplyFetchedPods", Array.from(respBytes)).catch(() => undefined);
  }

  apply_appended_pods(respBytes: Uint8Array): void {
    const resp = fromBinary(ListPodsResponseSchema, respBytes);
    const existing = JSON.parse(this._podsCache) as { pod_key: string }[];
    const seen = new Set(existing.map((p) => p.pod_key));
    for (const p of resp.items) {
      const c = podToCache(p);
      if (!seen.has(c.pod_key as string)) existing.push(c as { pod_key: string });
    }
    this._podsCache = JSON.stringify(existing);
    void invoke<void>("appPodApplyAppendedPods", Array.from(respBytes)).catch(() => undefined);
  }

  insert_created_pod(reqBytes: Uint8Array): void {
    const req = fromBinary(InsertCreatedPodRequestSchema, reqBytes);
    if (!req.pod) return;
    const cache = JSON.parse(this._podsCache) as { pod_key: string }[];
    const c = podToCache(req.pod);
    const idx = cache.findIndex((p) => p.pod_key === c.pod_key);
    if (idx >= 0) cache[idx] = { ...cache[idx], ...c };
    else cache.unshift(c as { pod_key: string });
    this._podsCache = JSON.stringify(cache);
    void invoke<void>("appPodInsertCreated", Array.from(reqBytes)).catch(() => undefined);
  }

  mark_pod_terminated(reqBytes: Uint8Array): void {
    const req = fromBinary(MarkPodTerminatedRequestSchema, reqBytes);
    this._podsCache = patchPodInCache(this._podsCache, req.podKey, { status: "terminated" });
    void invoke<void>("appPodMarkTerminated", Array.from(reqBytes)).catch(() => undefined);
  }

  patch_pod_perpetual(reqBytes: Uint8Array): void {
    const req = fromBinary(PatchPodPerpetualRequestSchema, reqBytes);
    this._podsCache = patchPodInCache(this._podsCache, req.podKey, { perpetual: req.perpetual });
    void invoke<void>("appPodPatchPerpetual", Array.from(reqBytes)).catch(() => undefined);
  }

  // Surgical realtime mirror: the main-pushed snapshot carries one Rust-computed
  // pod. Update it in place ONLY if already cached — the pod sidebar is a
  // FILTERED set, so adding an absent pod would corrupt the filtered view (a
  // brand-new pod is added by the handler's fetchPod refetch instead).
  apply_pod_snapshot(podBytes: Uint8Array): void {
    if (!podBytes.length) return;
    let pod: PodData;
    try {
      pod = podToCache(fromBinary(PodSchema, podBytes));
    } catch {
      return;
    }
    if (!pod.pod_key) return;
    const cache = JSON.parse(this._podsCache) as Array<Record<string, unknown>>;
    const idx = cache.findIndex((p) => p.pod_key === pod.pod_key);
    if (idx >= 0) {
      cache[idx] = { ...cache[idx], ...pod };
      this._podsCache = JSON.stringify(cache);
    }
  }

  apply_pod_status_event(reqBytes: Uint8Array): void {
    const req = fromBinary(ApplyPodStatusEventRequestSchema, reqBytes);
    const patch: Record<string, unknown> = {};
    if (req.status) {
      patch.status = req.status;
      if (req.errorCode !== undefined) patch.error_code = req.errorCode ?? undefined;
      if (req.errorMessage !== undefined) patch.error_message = req.errorMessage ?? undefined;
    }
    if (req.agentStatus !== undefined) patch.agent_status = req.agentStatus ?? undefined;
    this._podsCache = patchPodInCache(this._podsCache, req.podKey, patch);
  }

  apply_pod_title_event(reqBytes: Uint8Array): void {
    const req = fromBinary(ApplyPodTitleEventRequestSchema, reqBytes);
    this._podsCache = patchPodInCache(this._podsCache, req.podKey, { title: req.title });
  }

  apply_pod_alias_event(reqBytes: Uint8Array): void {
    const req = fromBinary(ApplyPodAliasEventRequestSchema, reqBytes);
    this._podsCache = patchPodInCache(this._podsCache, req.podKey, { alias: req.alias ?? "" });
  }

  apply_agent_status_event(reqBytes: Uint8Array): void {
    const req = fromBinary(ApplyAgentStatusEventRequestSchema, reqBytes);
    this._podsCache = patchPodInCache(this._podsCache, req.podKey, { agent_status: req.agentStatus });
  }
}
