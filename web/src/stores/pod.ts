import { create } from "zustand";
import { podApi, PodData, ApiError } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";

// Re-export PodData as Pod for cleaner component API
export type Pod = PodData;

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

  // Actions
  fetchPods: (filters?: {
    status?: string;
    runnerId?: number;
  }) => Promise<void>;
  fetchPod: (podKey: string) => Promise<void>;
  createPod: (data: {
    runnerId: number;
    agentTypeId?: number;
    repositoryId?: number;
    ticketId?: number;
    initialPrompt?: string;
    branchName?: string;
  }) => Promise<Pod>;
  terminatePod: (podKey: string) => Promise<void>;
  setCurrentPod: (pod: Pod | null) => void;
  updatePodStatus: (podKey: string, status: Pod["status"], agentStatus?: string, errorCode?: string, errorMessage?: string) => void;
  updateAgentStatus: (podKey: string, agentStatus: string) => void;
  updatePodTitle: (podKey: string, title: string) => void;
  updatePodInitProgress: (podKey: string, phase: string, progress: number, message: string) => void;
  clearInitProgress: (podKey: string) => void;
  clearError: () => void;
}

export const usePodStore = create<PodState>((set) => ({
  pods: [],
  currentPod: null,
  loading: false,
  error: null,
  initProgress: {},

  fetchPods: async (filters) => {
    set({ loading: true, error: null });
    try {
      const response = await podApi.list(filters);
      set({ pods: response.pods || [], loading: false });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch pods"),
        loading: false,
      });
    }
  },

  fetchPod: async (podKey) => {
    set({ loading: true, error: null });
    try {
      const response = await podApi.get(podKey);
      const fetchedPod = response.pod;
      set((state) => {
        // Update or add to pods array
        const existingIndex = state.pods.findIndex((p) => p.pod_key === podKey);
        const updatedPods = existingIndex >= 0
          ? state.pods.map((p) => (p.pod_key === podKey ? fetchedPod : p))
          : [...state.pods, fetchedPod];
        return {
          pods: updatedPods,
          currentPod: fetchedPod,
          loading: false,
        };
      });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch pod"),
        loading: false,
      });
    }
  },

  createPod: async (data) => {
    set({ loading: true, error: null });
    try {
      // Convert camelCase to snake_case for API
      const apiData = {
        agent_type_id: data.agentTypeId ?? 0,
        runner_id: data.runnerId,
        repository_id: data.repositoryId,
        ticket_id: data.ticketId,
        initial_prompt: data.initialPrompt,
        branch_name: data.branchName,
      };
      const response = await podApi.create(apiData);
      set((state) => ({
        pods: [response.pod, ...state.pods],
        currentPod: response.pod,
        loading: false,
      }));
      return response.pod;
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to create pod"),
        loading: false,
      });
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
    set((state) => ({
      pods: state.pods.map((p) =>
        p.pod_key === podKey ? { ...p, status: "terminated" as const } : p
      ),
      currentPod:
        state.currentPod?.pod_key === podKey
          ? { ...state.currentPod, status: "terminated" as const }
          : state.currentPod,
    }));
  },

  setCurrentPod: (pod) => {
    set({ currentPod: pod });
  },

  updatePodStatus: (podKey, status, agentStatus, errorCode, errorMessage) => {
    set((state) => ({
      pods: state.pods.map((p) =>
        p.pod_key === podKey
          ? {
              ...p,
              status,
              ...(agentStatus && { agent_status: agentStatus }),
              ...(errorCode !== undefined && { error_code: errorCode }),
              ...(errorMessage !== undefined && { error_message: errorMessage }),
            }
          : p
      ),
      currentPod:
        state.currentPod?.pod_key === podKey
          ? {
              ...state.currentPod,
              status,
              ...(agentStatus && { agent_status: agentStatus }),
              ...(errorCode !== undefined && { error_code: errorCode }),
              ...(errorMessage !== undefined && { error_message: errorMessage }),
            }
          : state.currentPod,
    }));
  },

  updateAgentStatus: (podKey, agentStatus) => {
    set((state) => ({
      pods: state.pods.map((p) =>
        p.pod_key === podKey ? { ...p, agent_status: agentStatus } : p
      ),
      currentPod:
        state.currentPod?.pod_key === podKey
          ? { ...state.currentPod, agent_status: agentStatus }
          : state.currentPod,
    }));
  },

  updatePodTitle: (podKey, title) => {
    set((state) => ({
      pods: state.pods.map((p) =>
        p.pod_key === podKey ? { ...p, title } : p
      ),
      currentPod:
        state.currentPod?.pod_key === podKey
          ? { ...state.currentPod, title }
          : state.currentPod,
    }));
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
