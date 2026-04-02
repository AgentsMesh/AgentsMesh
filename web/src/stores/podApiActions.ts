import { podApi, ApiError } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import { getErrorMessage } from "@/lib/utils";
import type { Pod, PodState } from "./podTypes";
import { SIDEBAR_STATUS_MAP, SIDEBAR_PAGE_SIZE, fetchPodInflight, upsertPod } from "./podTypes";

type SetState = (partial: Partial<PodState> | ((state: PodState) => Partial<PodState> | PodState)) => void;
type GetState = () => PodState;

export const createPodApiActions = (set: SetState, get: GetState) => ({
  fetchPods: async (filters?: { status?: string; runnerId?: number }) => {
    const fetchStartTs = Date.now();
    set({ error: null });
    try {
      const response = await podApi.list(filters);
      const apiPods = response.pods || [];
      set((state) => {
        const apiKeys = new Set(apiPods.map((p) => p.pod_key));
        const mergedPods = apiPods.map((apiPod) => {
          const localTs = state.podTimestamps[apiPod.pod_key];
          if (localTs && localTs > fetchStartTs) {
            const local = state.pods.find((p) => p.pod_key === apiPod.pod_key);
            if (local) return local;
          }
          return apiPod;
        });
        const localOnlyPods = state.pods.filter((p) => {
          if (apiKeys.has(p.pod_key)) return false;
          const localTs = state.podTimestamps[p.pod_key];
          return localTs !== undefined && localTs > fetchStartTs;
        });
        return { pods: [...localOnlyPods, ...mergedPods] };
      });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to fetch pods") });
    }
  },

  fetchPod: async (podKey: string) => {
    const inflight = fetchPodInflight.get(podKey);
    if (inflight) return inflight;

    const promise = (async () => {
      const fetchStartTs = Date.now();
      try {
        const response = await podApi.get(podKey);
        set((state) => upsertPod(state, podKey, () => response.pod, fetchStartTs) ?? state);
      } catch (error: unknown) {
        console.warn("[PodStore] fetchPod failed for", podKey, error);
        throw error;
      } finally {
        fetchPodInflight.delete(podKey);
      }
    })();

    fetchPodInflight.set(podKey, promise);
    return promise;
  },

  fetchSidebarPods: async (statusFilter: string) => {
    const fetchStartTs = Date.now();
    set({ loading: true, error: null, currentSidebarFilter: statusFilter });
    try {
      const statusParam = SIDEBAR_STATUS_MAP[statusFilter] ?? "";
      const createdById = statusFilter === "mine" ? useAuthStore.getState().user?.id : undefined;
      const response = await podApi.list({
        status: statusParam || undefined,
        createdById,
        limit: SIDEBAR_PAGE_SIZE,
        offset: 0,
      });
      const apiPods = response.pods || [];
      set((state) => {
        const mergedPods = apiPods.map((apiPod) => {
          const localTs = state.podTimestamps[apiPod.pod_key];
          if (localTs && localTs > fetchStartTs) {
            const local = state.pods.find((p) => p.pod_key === apiPod.pod_key);
            if (local) return local;
          }
          return apiPod;
        });
        return {
          pods: mergedPods,
          podTotal: response.total,
          podHasMore: mergedPods.length < response.total,
          loading: false,
        };
      });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to fetch pods"), loading: false });
    }
  },

  loadMorePods: async () => {
    const { pods, podHasMore, loadingMore, currentSidebarFilter } = get();
    if (!podHasMore || loadingMore) return;
    set({ loadingMore: true });
    try {
      const statusParam = SIDEBAR_STATUS_MAP[currentSidebarFilter] ?? "";
      const createdById = currentSidebarFilter === "mine" ? useAuthStore.getState().user?.id : undefined;
      const response = await podApi.list({
        status: statusParam || undefined,
        createdById,
        limit: SIDEBAR_PAGE_SIZE,
        offset: pods.length,
      });
      const newPods = response.pods || [];
      set((state) => {
        if (state.currentSidebarFilter !== currentSidebarFilter) {
          return { loadingMore: false };
        }
        const existingKeys = new Set(state.pods.map((p) => p.pod_key));
        const uniqueNewPods = newPods.filter((p) => !existingKeys.has(p.pod_key));
        const merged = [...state.pods, ...uniqueNewPods];
        return {
          pods: merged,
          podTotal: response.total,
          podHasMore: merged.length < response.total,
          loadingMore: false,
        };
      });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to load more pods"), loadingMore: false });
    }
  },

  terminatePod: async (podKey: string) => {
    try {
      await podApi.terminate(podKey);
    } catch (error: unknown) {
      const isNotFound = error instanceof ApiError && error.status === 404;
      if (!isNotFound) {
        set({ error: getErrorMessage(error, "Failed to terminate pod") });
        throw error;
      }
    }
    set((state) => {
      const result = upsertPod(state, podKey, (existing) =>
        existing ? { ...existing, status: "terminated" as const } : undefined
      );
      return result ?? state;
    });
  },

  updatePodAlias: async (podKey: string, alias: string | null) => {
    set((state) => {
      const result = upsertPod(state, podKey, (existing) =>
        existing ? { ...existing, alias: alias ?? undefined } : undefined,
      );
      return result ?? state;
    });
    try {
      await podApi.updateAlias(podKey, alias);
    } catch (error: unknown) {
      console.warn("[PodStore] updatePodAlias failed, reverting", error);
      try {
        const response = await podApi.get(podKey);
        set((state) => upsertPod(state, podKey, () => response.pod) ?? state);
      } catch {
        // Best-effort revert failed; leave as-is
      }
      throw error;
    }
  },
});
