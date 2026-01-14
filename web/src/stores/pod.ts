import { create } from "zustand";
import { podApi, PodData } from "@/lib/api/client";
import { getErrorMessage } from "@/lib/utils";

// Re-export PodData as Pod for backward compatibility
export type Pod = PodData;

interface PodState {
  // State
  pods: Pod[];
  currentPod: Pod | null;
  loading: boolean;
  error: string | null;

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
  updatePodStatus: (podKey: string, status: Pod["status"], agentStatus?: string) => void;
  updateAgentStatus: (podKey: string, agentStatus: string) => void;
  clearError: () => void;
}

export const usePodStore = create<PodState>((set) => ({
  pods: [],
  currentPod: null,
  loading: false,
  error: null,

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
      set({ currentPod: response.pod, loading: false });
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
      set((state) => ({
        pods: state.pods.map((p) =>
          p.pod_key === podKey ? { ...p, status: "terminated" as const } : p
        ),
        currentPod:
          state.currentPod?.pod_key === podKey
            ? { ...state.currentPod, status: "terminated" as const }
            : state.currentPod,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to terminate pod") });
      throw error;
    }
  },

  setCurrentPod: (pod) => {
    set({ currentPod: pod });
  },

  updatePodStatus: (podKey, status, agentStatus) => {
    set((state) => ({
      pods: state.pods.map((p) =>
        p.pod_key === podKey
          ? { ...p, status, ...(agentStatus && { agent_status: agentStatus }) }
          : p
      ),
      currentPod:
        state.currentPod?.pod_key === podKey
          ? {
              ...state.currentPod,
              status,
              ...(agentStatus && { agent_status: agentStatus }),
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

  clearError: () => {
    set({ error: null });
  },
}));
