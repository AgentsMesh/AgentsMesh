import {
  autopilotApi,
  CreateAutopilotControllerRequest,
  ApproveRequest,
} from "@/lib/api/autopilot";
import { getErrorMessage } from "@/lib/utils";
import type { AutopilotController, AutopilotIteration } from "./autopilot";
import { updateControllerInState } from "./autopilot-helpers";

type SetFn = (
  updater:
    | Partial<ReturnType<typeof getInitialState>>
    | ((state: ReturnType<typeof getInitialState>) => Partial<ReturnType<typeof getInitialState>>)
) => void;
type GetFn = () => ReturnType<typeof getInitialState>;

// Stub type for referencing the shape; actual state lives in autopilot.ts
function getInitialState() {
  return {
    autopilotControllers: [] as AutopilotController[],
    currentAutopilotController: null as AutopilotController | null,
    iterations: {} as Record<string, AutopilotIteration[]>,
    loading: false,
    error: null as string | null,
  };
}

export function createApiActions(set: SetFn, _get: GetFn) {
  return {
    fetchAutopilotControllers: async () => {
      set({ loading: true, error: null });
      try {
        const controllers = await autopilotApi.list();
        set({ autopilotControllers: controllers || [], loading: false });
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to fetch AutopilotControllers"), loading: false });
      }
    },

    fetchAutopilotController: async (key: string) => {
      set({ error: null });
      try {
        const controller = await autopilotApi.get(key);
        set((state) => {
          const exists = state.autopilotControllers.some((c) => c.autopilot_controller_key === key);
          return {
            autopilotControllers: exists
              ? state.autopilotControllers.map((c) => (c.autopilot_controller_key === key ? controller : c))
              : [...state.autopilotControllers, controller],
            currentAutopilotController: controller,
          };
        });
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to fetch AutopilotController") });
      }
    },

    createAutopilotController: async (data: CreateAutopilotControllerRequest) => {
      set({ error: null });
      try {
        const controller = await autopilotApi.create(data);
        set((state) => ({
          autopilotControllers: [...state.autopilotControllers, controller],
          currentAutopilotController: controller,
        }));
        return controller;
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to create AutopilotController") });
        throw error;
      }
    },

    pauseAutopilotController: async (key: string) => {
      try {
        await autopilotApi.pause(key);
        set((state) => updateControllerInState(state, key, { phase: "paused" as const }));
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to pause AutopilotController") });
      }
    },

    resumeAutopilotController: async (key: string) => {
      try {
        await autopilotApi.resume(key);
        set((state) => updateControllerInState(state, key, { phase: "running" as const }));
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to resume AutopilotController") });
      }
    },

    stopAutopilotController: async (key: string) => {
      try {
        await autopilotApi.stop(key);
        set((state) => updateControllerInState(state, key, { phase: "stopped" as const }));
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to stop AutopilotController") });
      }
    },

    approveAutopilotController: async (key: string, data?: ApproveRequest) => {
      try {
        await autopilotApi.approve(key, data);
        set((state) =>
          updateControllerInState(state, key, (c) => ({
            ...c,
            phase: data?.continue_execution === false ? ("stopped" as const) : ("running" as const),
            max_iterations: data?.additional_iterations
              ? c.max_iterations + data.additional_iterations
              : c.max_iterations,
          }))
        );
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to approve AutopilotController") });
      }
    },

    takeoverAutopilotController: async (key: string) => {
      try {
        await autopilotApi.takeover(key);
        set((state) =>
          updateControllerInState(state, key, { phase: "user_takeover" as const, user_takeover: true })
        );
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to takeover AutopilotController") });
      }
    },

    handbackAutopilotController: async (key: string) => {
      try {
        await autopilotApi.handback(key);
        set((state) =>
          updateControllerInState(state, key, { phase: "running" as const, user_takeover: false })
        );
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to handback AutopilotController") });
      }
    },

    fetchIterations: async (key: string) => {
      try {
        const iterations = await autopilotApi.getIterations(key);
        set((state) => ({ iterations: { ...state.iterations, [key]: iterations || [] } }));
      } catch (error: unknown) {
        set({ error: getErrorMessage(error, "Failed to fetch iterations") });
      }
    },
  };
}
