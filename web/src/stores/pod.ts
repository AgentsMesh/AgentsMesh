import { create } from "zustand";
import { podApi, PodData, ApiError } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import { getErrorMessage } from "@/lib/utils";

// Re-export PodData as Pod for cleaner component API
export type Pod = PodData;

// Sidebar status filter → API status query parameter mapping
export const SIDEBAR_STATUS_MAP: Record<string, string> = {
  mine: "running,initializing,orphaned",
  org: "running,initializing,orphaned",
  completed: "terminated,failed,paused,completed,error",
};
const SIDEBAR_PAGE_SIZE = 20;

// Pod initialization progress state
interface PodInitProgress {
  phase: string;
  progress: number;
  message: string;
}

interface PodState {
  // State
  pods: Pod[];
  currentPod: Pod | null;
  loading: boolean;
  error: string | null;
  // Pod initialization progress (keyed by pod_key)
  initProgress: Record<string, PodInitProgress>;
  // Timestamp guards — track last-known update time per pod to prevent
  // stale API responses from overwriting newer WebSocket event data.
  podTimestamps: Record<string, number>;
  // Sidebar pagination state
  podTotal: number;
  podHasMore: boolean;
  loadingMore: boolean;
  currentSidebarFilter: string;

  // Actions
  fetchPods: (filters?: {
    status?: string;
    runnerId?: number;
  }) => Promise<void>;
  fetchPod: (podKey: string) => Promise<void>;
  fetchSidebarPods: (statusFilter: string) => Promise<void>;
  loadMorePods: () => Promise<void>;
  createPod: (data: {
    runnerId: number;
    agentTypeId?: number;
    repositoryId?: number;
    ticketSlug?: string;
    initialPrompt?: string;
    branchName?: string;
  }) => Promise<Pod>;
  terminatePod: (podKey: string) => Promise<void>;
  setCurrentPod: (pod: Pod | null) => void;
  updatePodStatus: (podKey: string, status: Pod["status"], agentStatus?: string, errorCode?: string, errorMessage?: string, timestamp?: number) => void;
  updateAgentStatus: (podKey: string, agentStatus: string, timestamp?: number) => void;
  updatePodTitle: (podKey: string, title: string, timestamp?: number) => void;
  updatePodInitProgress: (podKey: string, phase: string, progress: number, message: string) => void;
  clearInitProgress: (podKey: string) => void;
  clearError: () => void;
}

// Track in-flight fetchPod requests to deduplicate concurrent calls.
// When multiple WebSocket events arrive for the same pod_key before the first
// API response returns, subsequent calls reuse the existing promise instead of
// issuing redundant requests.
const fetchPodInflight = new Map<string, Promise<void>>();

/**
 * Timestamp guard: only allow updates when the incoming timestamp is newer than
 * the last recorded one for this pod. Returns false if the update should be
 * skipped (stale data). When no timestamp is provided, the update is always
 * allowed (backwards-compatible).
 */
function shouldUpdate(
  podTimestamps: Record<string, number>,
  podKey: string,
  timestamp?: number,
): boolean {
  if (timestamp === undefined) return true;
  const existing = podTimestamps[podKey];
  return !existing || timestamp >= existing;
}

/** Record a timestamp for a pod, returning updated map (immutable). */
function recordTimestamp(
  podTimestamps: Record<string, number>,
  podKey: string,
  timestamp?: number,
): Record<string, number> {
  if (timestamp === undefined) return podTimestamps;
  return { ...podTimestamps, [podKey]: timestamp };
}

/**
 * Unified pod upsert — single write path for all pod data mutations.
 * Handles deduplication, currentPod sync, and timestamp guards.
 *
 * @param state     Current store state
 * @param podKey    Target pod key
 * @param merger    Produces the new Pod given the existing one (or undefined for new).
 *                  Return undefined to skip the update (e.g. partial update on missing pod).
 * @param timestamp Optional timestamp for staleness guard
 * @param options.prepend  If true, new pods are prepended (default: append)
 * @returns Partial state update, or null if skipped (stale or no-op)
 */
function upsertPod(
  state: PodState,
  podKey: string,
  merger: (existing: Pod | undefined) => Pod | undefined,
  timestamp?: number,
  options?: { prepend?: boolean },
): Partial<PodState> | null {
  if (!shouldUpdate(state.podTimestamps, podKey, timestamp)) {
    return null;
  }

  const existingIndex = state.pods.findIndex((p) => p.pod_key === podKey);
  const existing = existingIndex >= 0 ? state.pods[existingIndex] : undefined;
  const merged = merger(existing);

  // Merger returned undefined → partial update on non-existing pod, skip
  if (!merged) return null;

  let updatedPods: Pod[];
  if (existingIndex >= 0) {
    updatedPods = state.pods.map((p) => (p.pod_key === podKey ? merged : p));
  } else if (options?.prepend) {
    updatedPods = [merged, ...state.pods];
  } else {
    updatedPods = [...state.pods, merged];
  }

  return {
    pods: updatedPods,
    currentPod: state.currentPod?.pod_key === podKey ? merged : state.currentPod,
    podTimestamps: recordTimestamp(state.podTimestamps, podKey, timestamp),
  };
}

export const usePodStore = create<PodState>((set, get) => ({
  pods: [],
  currentPod: null,
  loading: false,
  error: null,
  initProgress: {},
  podTimestamps: {},
  podTotal: 0,
  podHasMore: false,
  loadingMore: false,
  currentSidebarFilter: "mine",

  fetchPods: async (filters) => {
    const fetchStartTs = Date.now();
    // NOTE: Do NOT set shared `loading` here — fetchPods is called fire-and-forget
    // from multiple places (IDEShell, MobileSidebar, ChannelPodManager) and should
    // not interfere with fetchSidebarPods which owns the sidebar loading state.
    set({ error: null });
    try {
      const response = await podApi.list(filters);
      const apiPods = response.pods || [];
      set((state) => {
        // Merge strategy: preserve locally-modified pods (WS events after API request)
        const apiKeys = new Set(apiPods.map((p) => p.pod_key));
        const mergedPods = apiPods.map((apiPod) => {
          const localTs = state.podTimestamps[apiPod.pod_key];
          if (localTs && localTs > fetchStartTs) {
            const local = state.pods.find((p) => p.pod_key === apiPod.pod_key);
            if (local) return local;
          }
          return apiPod;
        });
        // Preserve locally-known pods not in API response (e.g., just-created pods
        // that haven't been indexed yet or don't match the filter)
        const localOnlyPods = state.pods.filter((p) => {
          if (apiKeys.has(p.pod_key)) return false;
          const localTs = state.podTimestamps[p.pod_key];
          return localTs !== undefined && localTs > fetchStartTs;
        });
        return {
          pods: [...localOnlyPods, ...mergedPods],
        };
      });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch pods"),
      });
    }
  },

  fetchPod: async (podKey) => {
    // Deduplicate: if a fetchPod for this podKey is already in-flight, reuse it.
    const inflight = fetchPodInflight.get(podKey);
    if (inflight) return inflight;

    const promise = (async () => {
      // NOTE: Do NOT set global loading here — this is a background refresh for a
      // single pod (typically triggered by a WebSocket event).  Setting loading:true
      // would cause the sidebar to flash a loading spinner on every realtime event.

      // Capture start time BEFORE the API call. If a WebSocket event updates this
      // pod while the request is in flight, its timestamp will be > fetchStartTs,
      // and the stale API response will be correctly skipped by the guard.
      const fetchStartTs = Date.now();
      try {
        const response = await podApi.get(podKey);
        set((state) => upsertPod(state, podKey, () => response.pod, fetchStartTs) ?? state);
      } catch (error: unknown) {
        // Don't set global error state — callers (e.g., usePodStatus) handle errors individually.
        console.warn("[PodStore] fetchPod failed for", podKey, error);
        throw error;
      } finally {
        fetchPodInflight.delete(podKey);
      }
    })();

    fetchPodInflight.set(podKey, promise);
    return promise;
  },

  fetchSidebarPods: async (statusFilter) => {
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
        // Reconnection recovery: preserve local pod data when a WebSocket event
        // updated it after the API request was initiated (timestamp > fetchStartTs).
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
      set({
        error: getErrorMessage(error, "Failed to fetch pods"),
        loading: false,
      });
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
        // Deduplicate: realtime events may have already added some of these pods
        const existingKeys = new Set(state.pods.map((p) => p.pod_key));
        const uniqueNewPods = newPods.filter(
          (p) => !existingKeys.has(p.pod_key)
        );
        const merged = [...state.pods, ...uniqueNewPods];
        return {
          pods: merged,
          podTotal: response.total,
          podHasMore: merged.length < response.total,
          loadingMore: false,
        };
      });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to load more pods"),
        loadingMore: false,
      });
    }
  },

  createPod: async (data) => {
    // NOTE: Do NOT set shared `loading` here — createPod is an independent
    // operation that should not interfere with fetchPods/fetchSidebarPods.
    set({ error: null });
    try {
      // Convert camelCase to snake_case for API
      const apiData = {
        agent_type_id: data.agentTypeId ?? 0,
        runner_id: data.runnerId,
        repository_id: data.repositoryId,
        ticket_slug: data.ticketSlug,
        initial_prompt: data.initialPrompt,
        branch_name: data.branchName,
      };
      const response = await podApi.create(apiData);
      const createTs = Date.now();
      set((state) => {
        const result = upsertPod(state, response.pod.pod_key, () => response.pod, createTs, { prepend: true });
        return {
          ...(result ?? {}),
          currentPod: response.pod,
        };
      });
      return response.pod;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to create pod") });
      throw error;
    }
  },

  terminatePod: async (podKey) => {
    try {
      await podApi.terminate(podKey);
    } catch (error: unknown) {
      // If pod is not found (404), treat it as already terminated
      // This can happen when the pod was already terminated or deleted
      const isNotFound = error instanceof ApiError && error.status === 404;
      if (!isNotFound) {
        set({ error: getErrorMessage(error, "Failed to terminate pod") });
        throw error;
      }
      // Pod doesn't exist (404), treat as terminated - continue to update local state
    }
    // Always update local state to mark pod as terminated
    set((state) => {
      const result = upsertPod(state, podKey, (existing) =>
        existing ? { ...existing, status: "terminated" as const } : undefined
      );
      return result ?? state;
    });
  },

  setCurrentPod: (pod) => {
    set({ currentPod: pod });
  },

  updatePodStatus: (podKey, status, agentStatus, errorCode, errorMessage, timestamp) => {
    set((state) => {
      const result = upsertPod(state, podKey, (existing) => {
        if (!existing) return undefined;
        return {
          ...existing,
          status,
          ...(agentStatus !== undefined && { agent_status: agentStatus }),
          // When errorCode is provided, use it. Otherwise, clear stale error fields
          // when transitioning away from "error" status (prevents InfoTabContent from
          // showing leftover error info on a running pod).
          error_code: errorCode !== undefined ? errorCode : (status === "error" ? existing.error_code : undefined),
          error_message: errorMessage !== undefined ? errorMessage : (status === "error" ? existing.error_message : undefined),
        };
      }, timestamp);
      return result ?? state;
    });
  },

  updateAgentStatus: (podKey, agentStatus, timestamp) => {
    set((state) => {
      const result = upsertPod(state, podKey, (existing) =>
        existing ? { ...existing, agent_status: agentStatus } : undefined,
        timestamp,
      );
      return result ?? state;
    });
  },

  updatePodTitle: (podKey, title, timestamp) => {
    set((state) => {
      const result = upsertPod(state, podKey, (existing) =>
        existing ? { ...existing, title } : undefined,
        timestamp,
      );
      return result ?? state;
    });
  },

  updatePodInitProgress: (podKey, phase, progress, message) => {
    set((state) => ({
      initProgress: {
        ...state.initProgress,
        [podKey]: { phase, progress, message },
      },
    }));
  },

  clearInitProgress: (podKey) => {
    set((state) => {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { [podKey]: _removed, ...rest } = state.initProgress;
      return { initProgress: rest };
    });
  },

  clearError: () => {
    set({ error: null });
  },
}));
