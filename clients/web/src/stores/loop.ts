import { create } from "zustand";
import { useMemo } from "react";
import type { LoopData, LoopRunData, RunStatus, CreateLoopRequest, UpdateLoopRequest } from "@/lib/api/loopTypes";
import { getLoopService } from "@/lib/wasm-core";
import { readCurrentOrg } from "@/stores/auth";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import {
  listLoops as listLoopsConnect,
  getLoop as getLoopConnect,
  createLoop as createLoopConnect,
  updateLoop as updateLoopConnect,
  deleteLoop as deleteLoopConnect,
  enableLoop as enableLoopConnect,
  disableLoop as disableLoopConnect,
  triggerLoop as triggerLoopConnect,
  listLoopRuns as listLoopRunsConnect,
  cancelLoopRun as cancelLoopRunConnect,
} from "@/lib/api/loopConnect";

export type { LoopData, LoopRunData, RunStatus };

const svc = () => getLoopService();
const bump = () => useLoopStore.setState((s) => ({ _tick: s._tick + 1 }));

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

export function useLoops(): LoopData[] {
  const tick = useLoopStore((s) => s._tick);
  return useMemo(() => JSON.parse(svc().loops_json()), [tick]);
}

export function useCurrentLoop(): LoopData | null {
  const tick = useLoopStore((s) => s._tick);
  return useMemo(() => {
    const v = svc().current_loop_json();
    return v ? (typeof v === "string" ? JSON.parse(v) : v) : null;
  }, [tick]);
}

export function useLoopRuns(): LoopRunData[] {
  const tick = useLoopStore((s) => s._tick);
  return useMemo(() => JSON.parse(svc().runs_json()), [tick]);
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
      const { items, total } = await listLoopsConnect(orgSlug(), {
        status: filters?.status,
        query: filters?.query,
        limit: 500,
      });
      svc().set_loops(JSON.stringify(items));
      set({ totalCount: total, loading: false, _tick: get()._tick + 1 });
    } catch (err) { set({ error: getErrorMessage(err, "An error occurred"), loading: false }); }
  },

  fetchLoop: async (slug) => {
    const curJson = svc().current_loop_json();
    const curSlug = curJson ? (typeof curJson === "string" ? JSON.parse(curJson) : curJson)?.slug : null;
    if (curSlug !== slug) {
      svc().clear_runs();
      set({ runsTotalCount: 0, _tick: get()._tick + 1 });
    }
    set({ loopLoading: true, error: null });
    try {
      const loop = await getLoopConnect(orgSlug(), slug);
      svc().set_current_loop(JSON.stringify(loop));
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
    svc().set_current_loop("");
    bump();
    get().fetchLoops();
  },

  enableLoop: async (slug) => {
    const loop = await enableLoopConnect(orgSlug(), slug);
    svc().update_loop_local(slug, JSON.stringify(loop));
    bump();
  },
  disableLoop: async (slug) => {
    const loop = await disableLoopConnect(orgSlug(), slug);
    svc().update_loop_local(slug, JSON.stringify(loop));
    bump();
  },

  triggerLoop: async (slug) => {
    try {
      const result = await triggerLoopConnect(orgSlug(), slug);
      if (result.run) {
        svc().add_run(JSON.stringify(result.run));
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
      const { items, total } = await listLoopRunsConnect(orgSlug(), slug, {
        status: filters?.status,
        limit: filters?.limit,
        offset: filters?.offset,
      });
      if ((filters?.offset ?? 0) > 0) {
        svc().append_runs(JSON.stringify(items));
      } else {
        svc().set_runs(JSON.stringify(items));
      }
      set({ runsTotalCount: total, runsLoading: false, _tick: get()._tick + 1 });
    } catch (err) { set({ error: getErrorMessage(err, "An error occurred"), runsLoading: false }); }
  },

  loadMoreRuns: async (slug) => {
    if (get().runsLoading) return;
    const runs: LoopRunData[] = JSON.parse(svc().runs_json());
    await get().fetchRuns(slug, { limit: 20, offset: runs.length });
  },

  cancelRun: async (slug, runId) => {
    await cancelLoopRunConnect(orgSlug(), slug, runId);
    svc().update_run_status(BigInt(runId), "cancelled");
    bump();
    get().fetchRuns(slug, { limit: 20, offset: 0 });
    get().fetchLoop(slug);
  },

  setCurrentLoop: (loop) => {
    svc().set_current_loop(loop ? JSON.stringify(loop) : "");
    bump();
  },

  getLoopBySlug: (slug) => {
    const val = svc().get_loop_by_slug_json(slug);
    return val ? (typeof val === "string" ? JSON.parse(val) : val) : undefined;
  },

  clearError: () => set({ error: null }),
}));

reconnectRegistry.register({
  name: "loop:list",
  fn: () => useLoopStore.getState().fetchLoops?.(),
  priority: "low",
});
