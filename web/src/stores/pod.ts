import { create } from "zustand";
import type { PodState } from "./podTypes";
import { upsertPod } from "./podTypes";
import { createPodApiActions } from "./podApiActions";

// Re-export types for consumer convenience
export type { Pod } from "./podTypes";
export { SIDEBAR_STATUS_MAP } from "./podTypes";

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

  // API actions (fetch, create, terminate, alias)
  ...createPodApiActions(set, get),

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

  updatePodAliasFromEvent: (podKey, alias) => {
    set((state) => {
      const result = upsertPod(state, podKey, (existing) =>
        existing ? { ...existing, alias: alias ?? undefined } : undefined,
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
