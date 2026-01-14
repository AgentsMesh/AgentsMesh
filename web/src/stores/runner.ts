import { create } from "zustand";
import { runnerApi, RunnerData } from "@/lib/api/client";
import { getErrorMessage } from "@/lib/utils";

export type RunnerStatus = "online" | "offline" | "maintenance" | "busy";

// Re-export types for backward compatibility
export type Runner = RunnerData;

interface RunnerState {
  // State
  runners: Runner[];
  availableRunners: Runner[];
  currentRunner: Runner | null;
  loading: boolean;
  error: string | null;

  // Actions
  fetchRunners: (status?: RunnerStatus) => Promise<void>;
  fetchAvailableRunners: () => Promise<void>;
  fetchRunner: (id: number) => Promise<void>;
  updateRunner: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean }) => Promise<Runner>;
  deleteRunner: (id: number) => Promise<void>;
  regenerateAuthToken: (id: number) => Promise<string>;
  // Token management - simplified to one-time token generation
  createToken: () => Promise<string>;
  setCurrentRunner: (runner: Runner | null) => void;
  updateRunnerStatus: (runnerId: number, status: RunnerStatus) => void;
  clearError: () => void;
}

export const useRunnerStore = create<RunnerState>((set) => ({
  runners: [],
  availableRunners: [],
  currentRunner: null,
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
    set({ loading: true, error: null });
    try {
      const response = await runnerApi.listAvailable();
      set({ availableRunners: response.runners || [], loading: false });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch available runners"),
        loading: false,
      });
    }
  },

  fetchRunner: async (id) => {
    set({ loading: true, error: null });
    try {
      const response = await runnerApi.get(id);
      set({ currentRunner: response.runner, loading: false });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch runner"),
        loading: false,
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

  regenerateAuthToken: async (id) => {
    try {
      const response = await runnerApi.regenerateAuthToken(id);
      return response.auth_token;
    } catch (error: unknown) {
      const message = getErrorMessage(error, "Failed to regenerate auth token");
      set({ error: message });
      throw error;
    }
  },

  createToken: async () => {
    try {
      const response = await runnerApi.createToken();
      return response.token;
    } catch (error: unknown) {
      const message = getErrorMessage(error, "Failed to create token");
      set({ error: message });
      throw error;
    }
  },

  setCurrentRunner: (runner) => {
    set({ currentRunner: runner });
  },

  updateRunnerStatus: (runnerId, status) => {
    set((state) => ({
      runners: state.runners.map((r) =>
        r.id === runnerId ? { ...r, status } : r
      ),
      availableRunners:
        status === "online"
          ? state.availableRunners
          : state.availableRunners.filter((r) => r.id !== runnerId),
      currentRunner:
        state.currentRunner?.id === runnerId
          ? { ...state.currentRunner, status }
          : state.currentRunner,
    }));
  },

  clearError: () => {
    set({ error: null });
  },
}));

// Helper function to get status display info
export const getRunnerStatusInfo = (status: RunnerStatus) => {
  const statusMap: Record<
    RunnerStatus,
    { label: string; color: string; dotColor: string }
  > = {
    online: {
      label: "Online",
      color: "text-green-600",
      dotColor: "bg-green-500",
    },
    offline: {
      label: "Offline",
      color: "text-gray-500",
      dotColor: "bg-gray-400",
    },
    maintenance: {
      label: "Maintenance",
      color: "text-yellow-600",
      dotColor: "bg-yellow-500",
    },
    busy: {
      label: "Busy",
      color: "text-orange-600",
      dotColor: "bg-orange-500",
    },
  };
  return statusMap[status];
};

// Helper function to check if runner can accept new pods
export const canAcceptPods = (runner: Runner): boolean => {
  return (
    runner.status === "online" &&
    runner.current_pods < runner.max_concurrent_pods
  );
};

// Helper function to format host info
export const formatHostInfo = (hostInfo?: Runner["host_info"]) => {
  if (!hostInfo) return "Unknown";

  const parts: string[] = [];
  if (hostInfo.os) parts.push(hostInfo.os);
  if (hostInfo.arch) parts.push(hostInfo.arch);
  if (hostInfo.cpu_cores) parts.push(`${hostInfo.cpu_cores} cores`);
  if (hostInfo.memory) {
    const memoryGB = (hostInfo.memory / 1024 / 1024 / 1024).toFixed(1);
    parts.push(`${memoryGB}GB RAM`);
  }

  return parts.length > 0 ? parts.join(" / ") : "Unknown";
};
