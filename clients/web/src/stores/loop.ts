import { create } from "zustand";
import { useMemo } from "react";
import { create as protoCreate, toBinary, fromBinary } from "@bufbuild/protobuf";
import type { LoopData, LoopRunData, RunStatus, CreateLoopRequest, UpdateLoopRequest } from "@/lib/viewModels/loop";
import { getLoopState } from "@/lib/wasm-core";
import { readCurrentOrg } from "@/stores/auth";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import {
  listLoopsRaw as listLoopsRawConnect,
  getLoopRaw as getLoopRawConnect,
  createLoop as createLoopConnect,
  updateLoop as updateLoopConnect,
  deleteLoop as deleteLoopConnect,
  enableLoop as enableLoopConnect,
  disableLoop as disableLoopConnect,
  triggerLoop as triggerLoopConnect,
  listRunsRaw as listRunsRawConnect,
  cancelLoopRun as cancelLoopRunConnect,
} from "@/lib/api/facade/loopConnect";
import {
  ClearCurrentLoopRequestSchema,
  ClearLoopRunsRequestSchema, InsertLoopRunRequestSchema,
  PatchLoopFromActionRequestSchema, PatchLoopRunStatusRequestSchema,
  ReplaceCachedLoopsRequestSchema, ReplaceCachedRunsRequestSchema,
  SetCurrentLoopRequestSchema,
} from "@proto/loop_state/v1/loop_state_pb";
import { ListLoopsResponseSchema, ListRunsResponseSchema } from "@proto/loop/v1/loop_pb";
import { loopToProtoLoop, loopRunToProtoLoopRun } from "@/lib/api/loopProtoMap";
import { loopToCache, loopRunToCache } from "@agentsmesh/electron-adapter/projections";

export type { LoopData, LoopRunData, RunStatus };

const svc = () => getLoopState();
const bump = () => useLoopStore.setState((s) => ({ _tick: s._tick + 1 }));

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

// Read side (B, zero-JSON): UI is a projection of state proto bytes decoded via
// fromBinary + loopToCache/loopRunToCache (shared projection). LoopData is a
// lossy subset of proto.loop.v1.Loop on the Rust side — the same fields the old
// loops_json read path dropped, so no UI regression.
export function useLoops(): LoopData[] {
  const tick = useLoopStore((s) => s._tick);
  return useMemo(
    () => fromBinary(ReplaceCachedLoopsRequestSchema, svc().loops_bytes()).loops.map(loopToCache),
    [tick],
  );
}

export function useCurrentLoop(): LoopData | null {
  const tick = useLoopStore((s) => s._tick);
  return useMemo(() => {
    const bytes = svc().current_loop_bytes();
    if (bytes.length === 0) return null;
    const loop = fromBinary(SetCurrentLoopRequestSchema, bytes).loop;
    return loop ? loopToCache(loop) : null;
  }, [tick]);
}

export function useLoopRuns(): LoopRunData[] {
  const tick = useLoopStore((s) => s._tick);
  return useMemo(
    () => fromBinary(ReplaceCachedRunsRequestSchema, svc().runs_bytes()).runs.map(loopRunToCache),
    [tick],
  );
}

function setCurrentLoop(loop: LoopData): void {
  const req = protoCreate(SetCurrentLoopRequestSchema, { loop: loopToProtoLoop(loop) });
  svc().set_current_loop(toBinary(SetCurrentLoopRequestSchema, req));
}

function clearCurrentLoop(): void {
  const req = protoCreate(ClearCurrentLoopRequestSchema, {});
  svc().clear_current_loop(toBinary(ClearCurrentLoopRequestSchema, req));
}

function patchLoopFromAction(slug: string, loop: LoopData): void {
  const req = protoCreate(PatchLoopFromActionRequestSchema, {
    slug, loop: loopToProtoLoop(loop),
  });
  svc().patch_loop_from_action(toBinary(PatchLoopFromActionRequestSchema, req));
}

function insertLoopRun(run: LoopRunData): void {
  const req = protoCreate(InsertLoopRunRequestSchema, { run: loopRunToProtoLoopRun(run) });
  svc().insert_loop_run(toBinary(InsertLoopRunRequestSchema, req));
}

function patchLoopRunStatus(runId: number, status: string): void {
  const req = protoCreate(PatchLoopRunStatusRequestSchema, {
    runId: BigInt(runId), status,
  });
  svc().patch_loop_run_status(toBinary(PatchLoopRunStatusRequestSchema, req));
}

function clearLoopRuns(): void {
  const req = protoCreate(ClearLoopRunsRequestSchema, {});
  svc().clear_loop_runs(toBinary(ClearLoopRunsRequestSchema, req));
}

interface LoopStoreState {
  _tick: number;
  loading: boolean; loopLoading: boolean; runsLoading: boolean;
  error: string | null; totalCount: number; runsTotalCount: number;
  fetchLoops: (filters?: { query?: string; status?: string }) => Promise<void>;
  fetchLoop: (slug: string) => Promise<void>;
  createLoop: (data: CreateLoopRequest) => Promise<{ loop: LoopData }>;
  updateLoop: (slug: string, data: UpdateLoopRequest) => Promise<LoopData>;
  deleteLoop: (slug: string) => Promise<void>;
  enableLoop: (slug: string) => Promise<void>;
  disableLoop: (slug: string) => Promise<void>;
  triggerLoop: (slug: string) => Promise<{ run?: LoopRunData; skipped?: boolean; reason?: string }>;
  fetchRuns: (slug: string, filters?: { status?: string; limit?: number; offset?: number }) => Promise<void>;
  loadMoreRuns: (slug: string) => Promise<void>;
  cancelRun: (slug: string, runId: number) => Promise<void>;
  setCurrentLoop: (loop: LoopData | null) => void;
  getLoopBySlug: (slug: string) => LoopData | undefined;
  clearError: () => void;
}

export const useLoopStore = create<LoopStoreState>((set, get) => ({
  _tick: 0,
  loading: false, loopLoading: false, runsLoading: false,
  error: null, totalCount: 0, runsTotalCount: 0,

  fetchLoops: async (filters) => {
    set({ loading: true, error: null });
    try {
      const respBytes = await listLoopsRawConnect(orgSlug(), {
        status: filters?.status, query: filters?.query, limit: 500,
      });
      svc().apply_fetched_loops(respBytes);
      set({ loading: false, _tick: get()._tick + 1 });
    } catch (err) { set({ error: getErrorMessage(err, "An error occurred"), loading: false }); }
  },

  fetchLoop: async (slug) => {
    const curBytes = svc().current_loop_bytes();
    const curLoop = curBytes.length === 0 ? null : fromBinary(SetCurrentLoopRequestSchema, curBytes).loop;
    if ((curLoop?.slug ?? null) !== slug) {
      clearLoopRuns();
      set({ runsTotalCount: 0, _tick: get()._tick + 1 });
    }
    set({ loopLoading: true, error: null });
    try {
      const respBytes = await getLoopRawConnect(orgSlug(), slug);
      svc().apply_fetched_current_loop(respBytes);
      set({ loopLoading: false, _tick: get()._tick + 1 });
    } catch (err) { set({ error: getErrorMessage(err, "An error occurred"), loopLoading: false }); }
  },

  createLoop: async (data) => {
    const loop = await createLoopConnect(orgSlug(), data);
    get().fetchLoops();
    return { loop };
  },

  updateLoop: async (slug, data) => {
    const loop = await updateLoopConnect(orgSlug(), slug, data);
    bump();
    get().fetchLoops();
    return loop;
  },

  deleteLoop: async (slug) => {
    await deleteLoopConnect(orgSlug(), slug);
    clearCurrentLoop();
    bump();
    get().fetchLoops();
  },

  enableLoop: async (slug) => {
    const loop = await enableLoopConnect(orgSlug(), slug);
    patchLoopFromAction(slug, loop);
    bump();
  },
  disableLoop: async (slug) => {
    const loop = await disableLoopConnect(orgSlug(), slug);
    patchLoopFromAction(slug, loop);
    bump();
  },

  triggerLoop: async (slug) => {
    try {
      const result = await triggerLoopConnect(orgSlug(), slug);
      if (result.run) {
        insertLoopRun(result.run);
      }
      bump();
      get().fetchRuns(slug, { limit: 20, offset: 0 });
      get().fetchLoop(slug);
      return result;
    } catch { return { skipped: true, reason: "trigger skipped or failed" }; }
  },

  fetchRuns: async (slug, filters) => {
    set({ runsLoading: true });
    try {
      const respBytes = await listRunsRawConnect(orgSlug(), slug, {
        status: filters?.status, limit: filters?.limit, offset: filters?.offset,
      });
      // total is pagination metadata (not state) — decode it off the same wire
      // bytes that feed apply_*_runs, so no second RPC and no JSON state read.
      const total = Number(fromBinary(ListRunsResponseSchema, respBytes).total);
      if ((filters?.offset ?? 0) > 0) svc().apply_appended_runs(respBytes);
      else svc().apply_fetched_runs(respBytes);
      set({ runsTotalCount: total, runsLoading: false, _tick: get()._tick + 1 });
    } catch (err) { set({ error: getErrorMessage(err, "An error occurred"), runsLoading: false }); }
  },

  loadMoreRuns: async (slug) => {
    if (get().runsLoading) return;
    const loaded = fromBinary(ReplaceCachedRunsRequestSchema, svc().runs_bytes()).runs.length;
    await get().fetchRuns(slug, { limit: 20, offset: loaded });
  },

  cancelRun: async (slug, runId) => {
    await cancelLoopRunConnect(orgSlug(), slug, runId);
    patchLoopRunStatus(runId, "cancelled");
    bump();
    get().fetchRuns(slug, { limit: 20, offset: 0 });
    get().fetchLoop(slug);
  },

  setCurrentLoop: (loop) => {
    if (loop) setCurrentLoop(loop);
    else clearCurrentLoop();
    bump();
  },

  getLoopBySlug: (slug) => {
    const loops = fromBinary(ReplaceCachedLoopsRequestSchema, svc().loops_bytes()).loops;
    const found = loops.find((l) => l.slug === slug);
    return found ? loopToCache(found) : undefined;
  },

  clearError: () => set({ error: null }),
}));

reconnectRegistry.register({
  name: "loop:list",
  fn: () => useLoopStore.getState().fetchLoops?.(),
  priority: "low",
});
