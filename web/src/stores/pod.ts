import { create } from "zustand";
import { useMemo } from "react";
import { ApiError } from "@/lib/api/api-types";
import { reconnectRegistry } from "@/lib/realtime";
import { readCurrentUser, useAuthStore } from "@/stores/auth";
import { getErrorMessage } from "@/lib/utils";
import { initWasmCore, getPodService, getApiClient } from "@/lib/wasm-core";
import type { PodState, Pod } from "./podTypes";

export type { Pod } from "./podTypes";
export { SIDEBAR_STATUS_MAP } from "./podTypes";

export function usePods(): Pod[] {
  const tick = usePodStore((s) => s._tick);
  return useMemo(() => JSON.parse(getPodService().pods_json()) as Pod[], [tick]);
}

export function usePod(podKey: string | undefined): Pod | undefined {
  const tick = usePodStore((s) => s._tick);
  return useMemo(() => {
    if (!podKey) return undefined;
    const json = getPodService().get_pod_json(podKey);
    if (!json) return undefined;
    return JSON.parse(json as string) as Pod;
  }, [tick, podKey]);
}

export function useCurrentPod(): Pod | null {
  const tick = usePodStore((s) => s._tick);
  return useMemo(() => {
    const json = getPodService().current_pod_json();
    if (!json) return null;
    return JSON.parse(json as string) as Pod;
  }, [tick]);
}

const fetchPodInflight = new Map<string, Promise<void>>();
const bump = () => usePodStore.setState((s) => ({ _tick: s._tick + 1 }));

export const usePodStore = create<PodState>((set, get) => ({
  _tick: 0, loading: false, error: null, initProgress: {},
  podTotal: 0, podHasMore: false, loadingMore: false, currentSidebarFilter: "mine",

  fetchPods: async (filters) => {
    await initWasmCore();
    set({ error: null });
    try {
      await getPodService().fetch_pods(
        filters?.status ?? null,
        filters?.runnerId != null ? BigInt(filters.runnerId) : null,
        null, null, null,
      );
      bump();
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to fetch pods") });
    }
  },

  fetchPod: async (podKey) => {
    const inflight = fetchPodInflight.get(podKey);
    if (inflight) return inflight;
    const promise = (async () => {
      await initWasmCore();
      try {
        await getPodService().fetch_pod(podKey);
        bump();
      } catch (error: unknown) {
        console.warn("[PodStore] fetchPod failed for", podKey, error);
        throw error;
      } finally { fetchPodInflight.delete(podKey); }
    })();
    fetchPodInflight.set(podKey, promise);
    return promise;
  },

  fetchSidebarPods: async (statusFilter) => {
    await initWasmCore();
    set({ loading: true, error: null, currentSidebarFilter: statusFilter });
    try {
      const uid = statusFilter === "mine" ? readCurrentUser()?.id ?? null : null;
      const userId = uid != null ? BigInt(uid) : null;
      const json = await getPodService().fetch_sidebar_pods(statusFilter, userId);
      const { total, hasMore } = JSON.parse(json);
      set({ podTotal: total, podHasMore: hasMore, loading: false, _tick: get()._tick + 1 });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to fetch pods"), loading: false });
    }
  },

  loadMorePods: async () => {
    const { podHasMore, loadingMore, currentSidebarFilter } = get();
    if (!podHasMore || loadingMore) return;
    await initWasmCore();
    set({ loadingMore: true });
    try {
      const uid = currentSidebarFilter === "mine" ? readCurrentUser()?.id ?? null : null;
      const userId = uid != null ? BigInt(uid) : null;
      const pods: Pod[] = JSON.parse(getPodService().pods_json());
      const json = await getPodService().load_more_pods(currentSidebarFilter, userId, BigInt(pods.length));
      const { total, hasMore } = JSON.parse(json);
      set((state) => {
        if (state.currentSidebarFilter !== currentSidebarFilter) return { loadingMore: false };
        return { podTotal: total, podHasMore: hasMore, loadingMore: false, _tick: state._tick + 1 };
      });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to load more pods"), loadingMore: false });
    }
  },

  terminatePod: async (podKey) => {
    try {
      await getPodService().terminate_pod(podKey);
    } catch (error: unknown) {
      const msg = error instanceof Error ? error.message : String(error);
      const isNotFound = (error instanceof ApiError && error.status === 404) || msg.includes("404");
      if (!isNotFound) { set({ error: getErrorMessage(error, "Failed to terminate pod") }); throw error; }
    }
    bump();
  },

  upsertPod: (pod) => {
    getPodService().upsert_pod(JSON.stringify(pod));
    bump();
  },

  setCurrentPod: (pod) => {
    getPodService().set_current_pod(pod ? JSON.stringify(pod) : "");
    bump();
  },

  updatePodStatus: (podKey, status, agentStatus, errorCode, errorMessage) => {
    getPodService().update_pod_status(podKey, status, agentStatus ?? undefined, errorCode ?? undefined, errorMessage ?? undefined);
    bump();
  },

  updateAgentStatus: (podKey, agentStatus) => {
    getPodService().update_agent_status(podKey, agentStatus);
    bump();
  },

  updatePodTitle: (podKey, title) => {
    getPodService().update_pod_title(podKey, title);
    bump();
  },

  updatePodAliasFromEvent: (podKey, alias) => {
    getPodService().update_pod_alias(podKey, alias ?? "");
    bump();
  },

  updatePodAlias: async (podKey, alias) => {
    await initWasmCore();
    try {
      await getPodService().update_pod_alias_api(podKey, alias);
      bump();
    } catch (error: unknown) {
      console.warn("[PodStore] updatePodAlias failed, reverting", error);
      bump();
      throw error;
    }
  },

  updatePodPerpetualFromEvent: (podKey, perpetual) => {
    const raw = getPodService().get_pod_json(podKey);
    if (!raw) return;
    const pod = JSON.parse(String(raw)) as Pod;
    getPodService().upsert_pod(JSON.stringify({ ...pod, perpetual }));
    bump();
  },

  updatePodPerpetual: async (podKey, perpetual) => {
    await initWasmCore();
    try {
      await getApiClient().patch(
        getApiClient().org_path(`/pods/${podKey}/perpetual`),
        JSON.stringify({ perpetual }),
      );
      const raw = getPodService().get_pod_json(podKey);
      if (raw) {
        const pod = JSON.parse(String(raw)) as Pod;
        getPodService().upsert_pod(JSON.stringify({ ...pod, perpetual }));
      }
      bump();
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to update perpetual") });
      throw error;
    }
  },

  updatePodInitProgress: (podKey, phase, progress, message) => {
    set((state) => ({ initProgress: { ...state.initProgress, [podKey]: { phase, progress, message } } }));
  },

  clearInitProgress: (podKey) => {
    set((state) => {
      const { [podKey]: _removed, ...rest } = state.initProgress;
      return { initProgress: rest };
    });
  },

  clearError: () => set({ error: null }),
}));

reconnectRegistry.register({
  name: "pod:sidebar",
  fn: () => {
    const s = usePodStore.getState();
    s.fetchSidebarPods?.(s.currentSidebarFilter);
  },
  priority: "immediate",
});
