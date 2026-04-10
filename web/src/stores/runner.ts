import { create } from "zustand";
import { runnerApi, RunnerData, GRPCRegistrationToken } from "@/lib/api";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";

export type RunnerStatus = "online" | "offline" | "maintenance" | "busy";

// Re-export RunnerData as Runner for cleaner component API
export type Runner = RunnerData;

interface RunnerState {
  // State
  runners: Runner[];
  availableRunners: Runner[];
  currentRunner: Runner | null;
  tokens: GRPCRegistrationToken[];
  loading: boolean;
  error: string | null;

  // Actions
  fetchRunners: (status?: RunnerStatus) => Promise<void>;
  fetchAvailableRunners: () => Promise<void>;
  fetchRunner: (id: number) => Promise<void>;
  updateRunner: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean }) => Promise<Runner>;
  deleteRunner: (id: number) => Promise<void>;
  // Token management (gRPC registration tokens)
  createToken: (data?: { name?: string; labels?: string[]; max_uses?: number; expires_in_days?: number }) => Promise<string>;
  fetchTokens: () => Promise<void>;
  deleteToken: (id: number) => Promise<void>;
  setCurrentRunner: (runner: Runner | null) => void;
  updateRunnerStatus: (runnerId: number, status: RunnerStatus) => void;
  clearError: () => void;
}

export const useRunnerStore = create<RunnerState>((set) => ({
  runners: [],
  availableRunners: [],
  currentRunner: null,
  tokens: [],
  loading: false,
  error: null,

  fetchRunners: async (status) => {
    set({ loading: true, error: null });
    try {
      const response = await runnerApi.list(status);
      set({ runners: response.runners || [], loading: false });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch runners"),
        loading: false,
      });
    }
  },

  fetchAvailableRunners: async () => {
    try {
      const response = await runnerApi.listAvailable();
      set({ availableRunners: response.runners || [] });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch available runners"),
      });
    }
  },

  fetchRunner: async (id) => {
    try {
      const response = await runnerApi.get(id);
      set({ currentRunner: response.runner });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch runner"),
      });
    }
  },

  updateRunner: async (id, data) => {
    try {
      const response = await runnerApi.update(id, data);
      set((state) => ({
        runners: state.runners.map((r) => (r.id === id ? response.runner : r)),
        availableRunners: state.availableRunners.map((r) => (r.id === id ? response.runner : r)),
        currentRunner: state.currentRunner?.id === id ? response.runner : state.currentRunner,
      }));
      return response.runner;
    } catch (error: unknown) {
      const message = getErrorMessage(error, "Failed to update runner");
      set({ error: message });
      throw error;
    }
  },

  deleteRunner: async (id) => {
    try {
      await runnerApi.delete(id);
      set((state) => ({
        runners: state.runners.filter((r) => r.id !== id),
        availableRunners: state.availableRunners.filter((r) => r.id !== id),
        currentRunner:
          state.currentRunner?.id === id ? null : state.currentRunner,
      }));
    } catch (error: unknown) {
      const message = getErrorMessage(error, "Failed to delete runner");
      set({ error: message });
      throw error;
    }
  },

  createToken: async (data) => {
    try {
      const response = await runnerApi.createToken(data);
      return response.token;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to create token") });
      throw error;
    }
  },

  fetchTokens: async () => {
    try {
      const response = await runnerApi.listTokens();
      set({ tokens: response.tokens || [] });
    } catch (error: unknown) {
      const message = getErrorMessage(error, "Failed to fetch tokens");
      set({ error: message });
    }
  },

  deleteToken: async (id) => {
    try {
      await runnerApi.deleteToken(id);
      set((state) => ({
        tokens: state.tokens.filter((t) => t.id !== id),
      }));
    } catch (error: unknown) {
      const message = getErrorMessage(error, "Failed to delete token");
      set({ error: message });
      throw error;
    }
  },

  setCurrentRunner: (runner) => {
    set({ currentRunner: runner });
  },

  updateRunnerStatus: (runnerId, status) => {
    set((state) => {
      const updatedRunner = state.runners.find((r) => r.id === runnerId);
      const runnerWithStatus = updatedRunner ? { ...updatedRunner, status } : undefined;

      let availableRunners: Runner[];
      if (status === "online" && runnerWithStatus) {
        const alreadyAvailable = state.availableRunners.some((r) => r.id === runnerId);
        availableRunners = alreadyAvailable
          ? state.availableRunners.map((r) => (r.id === runnerId ? runnerWithStatus : r))
          : [...state.availableRunners, runnerWithStatus];
      } else {
        availableRunners = state.availableRunners.filter((r) => r.id !== runnerId);
      }

      return {
        runners: state.runners.map((r) => (r.id === runnerId ? { ...r, status } : r)),
        availableRunners,
        currentRunner: state.currentRunner?.id === runnerId
          ? { ...state.currentRunner, status }
          : state.currentRunner,
      };
    });
  },

  clearError: () => {
    set({ error: null });
  },
}));

export { getRunnerStatusInfo, canAcceptPods, formatHostInfo } from "./runner-display-info";

reconnectRegistry.register({
  name: "runner:list",
  fn: () => useRunnerStore.getState().fetchRunners?.(),
  priority: "immediate",
});
