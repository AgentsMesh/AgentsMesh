import { create } from "zustand";
import { loopApi, LoopData, LoopRunData, RunStatus } from "@/lib/api/loop";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";

export type { LoopData, LoopRunData, RunStatus };

interface LoopState {
  // State
  loops: LoopData[];
  currentLoop: LoopData | null;
  runs: LoopRunData[];
  loading: boolean;
  loopLoading: boolean;
  runsLoading: boolean;
  error: string | null;
  totalCount: number;
  runsTotalCount: number;
  runsOffset: number;

  // Actions
  fetchLoops: (filters?: { query?: string; status?: string }) => Promise<void>;
  fetchLoop: (slug: string) => Promise<void>;
  createLoop: (data: Parameters<typeof loopApi.create>[0]) => Promise<{ loop: LoopData }>;
  updateLoop: (slug: string, data: Parameters<typeof loopApi.update>[1]) => Promise<LoopData>;
  deleteLoop: (slug: string) => Promise<void>;
  enableLoop: (slug: string) => Promise<void>;
  disableLoop: (slug: string) => Promise<void>;
  triggerLoop: (slug: string) => Promise<{ run?: LoopRunData; skipped?: boolean; reason?: string }>;
  fetchRuns: (slug: string, filters?: { status?: string; limit?: number; offset?: number }) => Promise<void>;
  loadMoreRuns: (slug: string) => Promise<void>;
  cancelRun: (slug: string, runId: number) => Promise<void>;
  setCurrentLoop: (loop: LoopData | null) => void;
  clearError: () => void;
}

export const useLoopStore = create<LoopState>((set, get) => ({
  loops: [],
  currentLoop: null,
  runs: [],
  loading: false,
  loopLoading: false,
  runsLoading: false,
  error: null,
  totalCount: 0,
  runsTotalCount: 0,
  runsOffset: 0,

  fetchLoops: async (filters) => {
    set({ loading: true, error: null });
    try {
      const res = await loopApi.list(filters);
      set({ loops: res.loops || [], totalCount: res.total, loading: false });
    } catch (err) {
      set({ error: getErrorMessage(err, "An error occurred"), loading: false });
    }
  },

  fetchLoop: async (slug) => {
    // Clear runs when switching to a different loop to prevent stale data flash
    const currentSlug = get().currentLoop?.slug;
    if (currentSlug && currentSlug !== slug) {
      set({ runs: [], runsTotalCount: 0, runsOffset: 0 });
    }
    set({ loopLoading: true, error: null });
    try {
      const res = await loopApi.get(slug);
      set({ currentLoop: res.loop, loopLoading: false });
    } catch (err) {
      set({ error: getErrorMessage(err, "An error occurred"), loopLoading: false });
    }
  },

  createLoop: async (data) => {
    const res = await loopApi.create(data);
    // Refresh list
    get().fetchLoops();
    return res;
  },

  updateLoop: async (slug, data) => {
    const res = await loopApi.update(slug, data);
    set({ currentLoop: res.loop });
    get().fetchLoops();
    return res.loop;
  },

  deleteLoop: async (slug) => {
    await loopApi.delete(slug);
    set({ currentLoop: null });
    get().fetchLoops();
  },

  enableLoop: async (slug) => {
    const res = await loopApi.enable(slug);
    set({ currentLoop: res.loop });
    // Update in list
    set((state) => ({
      loops: state.loops.map((l) => (l.slug === slug ? res.loop : l)),
    }));
  },

  disableLoop: async (slug) => {
    const res = await loopApi.disable(slug);
    set({ currentLoop: res.loop });
    set((state) => ({
      loops: state.loops.map((l) => (l.slug === slug ? res.loop : l)),
    }));
  },

  triggerLoop: async (slug) => {
    const res = await loopApi.trigger(slug);
    if (res.skipped) {
      return { skipped: true, reason: res.reason };
    }
    // Refresh runs from offset 0 and reset offset state
    set({ runsOffset: 0 });
    get().fetchRuns(slug, { limit: 20, offset: 0 });
    get().fetchLoop(slug);
    return { run: res.run };
  },

  fetchRuns: async (slug, filters) => {
    set({ runsLoading: true });
    try {
      const res = await loopApi.listRuns(slug, filters);
      const newRuns = res.runs || [];
      const offset = filters?.offset ?? 0;
      // Append when loading more (offset > 0), replace on fresh load
      set((state) => ({
        runs: offset > 0 ? [...state.runs, ...newRuns] : newRuns,
        runsTotalCount: res.total,
        runsLoading: false,
      }));
    } catch (err) {
      set({ error: getErrorMessage(err, "An error occurred"), runsLoading: false });
    }
  },

  loadMoreRuns: async (slug) => {
    // Guard against double-click race condition — skip if already loading
    if (get().runsLoading) return;
    const prevOffset = get().runsOffset;
    const newOffset = prevOffset + 20;
    set({ runsOffset: newOffset });
    try {
      await get().fetchRuns(slug, { limit: 20, offset: newOffset });
    } catch {
      // Rollback offset on failure so the next "Load more" retries the same page
      set({ runsOffset: prevOffset });
    }
  },

  cancelRun: async (slug, runId) => {
    await loopApi.cancelRun(slug, runId);
    // Reset offset and refresh from first page
    set({ runsOffset: 0 });
    get().fetchRuns(slug, { limit: 20, offset: 0 });
    get().fetchLoop(slug);
  },

  setCurrentLoop: (loop) => set({ currentLoop: loop }),
  clearError: () => set({ error: null }),
}));

reconnectRegistry.register({
  name: "loop:list",
  fn: () => useLoopStore.getState().fetchLoops?.(),
  priority: "low",
});
