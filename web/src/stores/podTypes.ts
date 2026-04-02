import type { PodData } from "@/lib/api";

// Re-export PodData as Pod for cleaner component API
export type Pod = PodData;

// Sidebar status filter → API status query parameter mapping
export const SIDEBAR_STATUS_MAP: Record<string, string> = {
  mine: "running,initializing",
  org: "running,initializing",
  completed: "terminated,failed,paused,completed,error",
};
export const SIDEBAR_PAGE_SIZE = 20;

// Pod initialization progress state
export interface PodInitProgress {
  phase: string;
  progress: number;
  message: string;
}

export interface PodState {
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
  terminatePod: (podKey: string) => Promise<void>;
  setCurrentPod: (pod: Pod | null) => void;
  updatePodStatus: (podKey: string, status: Pod["status"], agentStatus?: string, errorCode?: string, errorMessage?: string, timestamp?: number) => void;
  updateAgentStatus: (podKey: string, agentStatus: string, timestamp?: number) => void;
  updatePodTitle: (podKey: string, title: string, timestamp?: number) => void;
  updatePodAlias: (podKey: string, alias: string | null) => Promise<void>;
  updatePodAliasFromEvent: (podKey: string, alias: string | null) => void;
  updatePodInitProgress: (podKey: string, phase: string, progress: number, message: string) => void;
  clearInitProgress: (podKey: string) => void;
  clearError: () => void;
}

// Track in-flight fetchPod requests to deduplicate concurrent calls.
export const fetchPodInflight = new Map<string, Promise<void>>();

/**
 * Timestamp guard: only allow updates when the incoming timestamp is newer than
 * the last recorded one for this pod.
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
 */
export function upsertPod(
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
