import { create } from "zustand";
import {
  AutopilotControllerData,
  AutopilotIterationData,
  CreateAutopilotControllerRequest,
  ApproveRequest,
} from "@/lib/api/autopilot";
import { AutopilotThinkingData } from "@/lib/realtime/types";
import { reconnectRegistry } from "@/lib/realtime";
import { createApiActions } from "./autopilot-actions";
import { updateControllerInState } from "./autopilot-helpers";

// Re-export types for component use
export type AutopilotController = AutopilotControllerData;
export type AutopilotIteration = AutopilotIterationData;
export type AutopilotThinking = AutopilotThinkingData;

interface AutopilotState {
  autopilotControllers: AutopilotController[];
  currentAutopilotController: AutopilotController | null;
  iterations: Record<string, AutopilotIteration[]>;
  thinking: Record<string, AutopilotThinking | null>;
  thinkingHistory: Record<string, AutopilotThinking[]>;
  loading: boolean;
  error: string | null;

  // API actions
  fetchAutopilotControllers: () => Promise<void>;
  fetchAutopilotController: (key: string) => Promise<void>;
  createAutopilotController: (data: CreateAutopilotControllerRequest) => Promise<AutopilotController>;
  pauseAutopilotController: (key: string) => Promise<void>;
  resumeAutopilotController: (key: string) => Promise<void>;
  stopAutopilotController: (key: string) => Promise<void>;
  approveAutopilotController: (key: string, data?: ApproveRequest) => Promise<void>;
  takeoverAutopilotController: (key: string) => Promise<void>;
  handbackAutopilotController: (key: string) => Promise<void>;
  fetchIterations: (key: string) => Promise<void>;

  // Real-time updates
  updateAutopilotControllerStatus: (
    key: string, phase: string, currentIteration: number,
    maxIterations: number, circuitBreakerState: string, circuitBreakerReason?: string
  ) => void;
  addIteration: (key: string, iteration: AutopilotIteration) => void;
  updateThinking: (key: string, thinking: AutopilotThinking) => void;
  setCurrentAutopilotController: (controller: AutopilotController | null) => void;
  removeAutopilotController: (key: string) => void;

  clearError: () => void;
  getAutopilotControllerByPodKey: (podKey: string) => AutopilotController | undefined;
  getThinking: (key: string) => AutopilotThinking | null;
  getThinkingHistory: (key: string) => AutopilotThinking[];
}

export const useAutopilotStore = create<AutopilotState>((set, get) => ({
  autopilotControllers: [],
  currentAutopilotController: null,
  iterations: {},
  thinking: {},
  thinkingHistory: {},
  loading: false,
  error: null,

  ...createApiActions(set as Parameters<typeof createApiActions>[0]),

  updateAutopilotControllerStatus: (key, phase, currentIteration, maxIterations, circuitBreakerState, circuitBreakerReason) => {
    set((state) =>
      updateControllerInState(state, key, {
        phase: phase as AutopilotController["phase"],
        current_iteration: currentIteration,
        max_iterations: maxIterations,
        circuit_breaker: {
          state: circuitBreakerState as AutopilotController["circuit_breaker"]["state"],
          reason: circuitBreakerReason,
        },
      })
    );
  },

  addIteration: (key, iteration) => {
    set((state) => {
      const prev = state.iterations[key] || [];
      const MAX_ITERATIONS = 200;
      const updated = prev.length >= MAX_ITERATIONS
        ? [...prev.slice(prev.length - MAX_ITERATIONS + 1), iteration] : [...prev, iteration];
      return { iterations: { ...state.iterations, [key]: updated } };
    });
  },

  updateThinking: (key, thinking) => {
    set((state) => {
      const prev = state.thinkingHistory[key] || [];
      const MAX = 100;
      const updated = prev.length >= MAX ? [...prev.slice(prev.length - MAX + 1), thinking] : [...prev, thinking];
      return {
        thinking: { ...state.thinking, [key]: thinking },
        thinkingHistory: { ...state.thinkingHistory, [key]: updated },
      };
    });
  },

  setCurrentAutopilotController: (controller) => set({ currentAutopilotController: controller }),

  removeAutopilotController: (key) => {
    set((state) => ({
      autopilotControllers: state.autopilotControllers.filter((c) => c.autopilot_controller_key !== key),
      currentAutopilotController:
        state.currentAutopilotController?.autopilot_controller_key === key ? null : state.currentAutopilotController,
    }));
  },

  clearError: () => set({ error: null }),

  getAutopilotControllerByPodKey: (podKey) => {
    return get().autopilotControllers.find(
      (c) => c.pod_key === podKey &&
        ["initializing", "running", "paused", "user_takeover", "waiting_approval"].includes(c.phase)
    );
  },

  getThinking: (key) => get().thinking[key] || null,
  getThinkingHistory: (key) => get().thinkingHistory[key] || [],
}));

reconnectRegistry.register({
  name: "autopilot:controllers",
  fn: () => useAutopilotStore.getState().fetchAutopilotControllers?.(),
  priority: "low",
});
