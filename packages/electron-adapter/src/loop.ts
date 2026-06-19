import type { ILoopService } from "@agentsmesh/service-interface";
import { fromBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedLoopsRequestSchema,
  SetCurrentLoopRequestSchema,
  ClearCurrentLoopRequestSchema,
  PatchLoopFromActionRequestSchema,
  InsertLoopRunRequestSchema,
  ReplaceCachedRunsRequestSchema,
  AppendCachedRunsRequestSchema,
  PatchLoopRunStatusRequestSchema,
  ClearLoopRunsRequestSchema,
} from "@agentsmesh/proto/loop_state/v1/loop_state_pb";
import { LoopSchema, ListLoopsResponseSchema, ListRunsResponseSchema } from "@agentsmesh/proto/loop/v1/loop_pb";
import { loopToCache, loopRunToCache } from "./projections/loop";
import { loopsBytes, runsBytes, currentLoopBytes } from "./loop_cache_to_bytes";

// Renderer-local loop cache. Unlike pod/channel/mesh there is NO fire-and-forget
// to the main process: the main-side Rust loop state had no reader, no snapshot
// push-back, and no persistence (a write-only dead mirror), so the app_loop_*
// NAPI commands were removed. Loop realtime rides the store refetch path.
export class ElectronLoopService implements ILoopService {
  private _loopsCache = "[]";
  private _runsCache = "[]";
  private _currentLoopCache: string | null = null;

  loops_json(): string { return this._loopsCache; }
  runs_json(): string { return this._runsCache; }
  current_loop_json(): unknown { return this._currentLoopCache; }

  get_loop_by_slug_json(slug: string): unknown {
    const loops = JSON.parse(this._loopsCache) as { slug: string }[];
    const l = loops.find(x => x.slug === slug);
    return l ? JSON.stringify(l) : null;
  }

  // Read side (B, zero-JSON): re-encode renderer cache into state proto bytes.
  loops_bytes(): Uint8Array { return loopsBytes(this._loopsCache); }
  runs_bytes(): Uint8Array { return runsBytes(this._runsCache); }
  current_loop_bytes(): Uint8Array { return currentLoopBytes(this._currentLoopCache); }

  // Fetch→state (B): decode wire ListLoops/ListRuns response → renderer cache.
  // Renderer-local only — loop realtime rides the store refetch path, no NAPI.
  apply_fetched_loops(respBytes: Uint8Array): void {
    const resp = fromBinary(ListLoopsResponseSchema, respBytes);
    this._loopsCache = JSON.stringify(resp.items.map(loopToCache));
  }

  // Single-object fetch (B): decode the full wire GetLoop response (Loop) →
  // current cache. Renderer-local only (no NAPI).
  apply_fetched_current_loop(respBytes: Uint8Array): void {
    this._currentLoopCache = JSON.stringify(loopToCache(fromBinary(LoopSchema, respBytes)));
  }

  apply_fetched_runs(respBytes: Uint8Array): void {
    const resp = fromBinary(ListRunsResponseSchema, respBytes);
    this._runsCache = JSON.stringify(resp.items.map(loopRunToCache));
  }

  apply_appended_runs(respBytes: Uint8Array): void {
    const resp = fromBinary(ListRunsResponseSchema, respBytes);
    const existing = JSON.parse(this._runsCache) as unknown[];
    this._runsCache = JSON.stringify([...existing, ...resp.items.map(loopRunToCache)]);
  }

  set_current_loop(reqBytes: Uint8Array): void {
    const req = fromBinary(SetCurrentLoopRequestSchema, reqBytes);
    this._currentLoopCache = req.loop ? JSON.stringify(loopToCache(req.loop)) : null;
  }

  clear_current_loop(reqBytes: Uint8Array): void {
    fromBinary(ClearCurrentLoopRequestSchema, reqBytes);
    this._currentLoopCache = null;
  }

  patch_loop_from_action(reqBytes: Uint8Array): void {
    const req = fromBinary(PatchLoopFromActionRequestSchema, reqBytes);
    if (req.loop) {
      const patch = loopToCache(req.loop);
      const list = JSON.parse(this._loopsCache) as Array<{ slug: string }>;
      const idx = list.findIndex(x => x.slug === req.slug);
      if (idx >= 0) list[idx] = { ...list[idx], ...patch } as { slug: string };
      this._loopsCache = JSON.stringify(list);
    }
  }

  insert_loop_run(reqBytes: Uint8Array): void {
    const req = fromBinary(InsertLoopRunRequestSchema, reqBytes);
    if (req.run) {
      const runs = JSON.parse(this._runsCache) as unknown[];
      runs.push(loopRunToCache(req.run));
      this._runsCache = JSON.stringify(runs);
    }
  }

  patch_loop_run_status(reqBytes: Uint8Array): void {
    const req = fromBinary(PatchLoopRunStatusRequestSchema, reqBytes);
    const runs = JSON.parse(this._runsCache) as Array<{ id: number; status?: string }>;
    const r = runs.find(x => x.id === Number(req.runId));
    if (r) r.status = req.status;
    this._runsCache = JSON.stringify(runs);
  }

  clear_loop_runs(reqBytes: Uint8Array): void {
    fromBinary(ClearLoopRunsRequestSchema, reqBytes);
    this._runsCache = "[]";
  }
}
