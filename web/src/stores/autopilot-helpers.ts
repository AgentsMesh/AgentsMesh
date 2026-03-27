import type { AutopilotController } from "./autopilot";

/**
 * Helper: update a controller by key in the list and current.
 * Accepts a partial update or a full updater function.
 */
export function updateControllerInState(
  state: {
    autopilotControllers: AutopilotController[];
    currentAutopilotController: AutopilotController | null;
  },
  key: string,
  updater: Partial<AutopilotController> | ((c: AutopilotController) => AutopilotController)
) {
  const applyUpdate = (c: AutopilotController) =>
    typeof updater === "function" ? updater(c) : { ...c, ...updater };

  return {
    autopilotControllers: state.autopilotControllers.map((c) =>
      c.autopilot_controller_key === key ? applyUpdate(c) : c
    ),
    currentAutopilotController:
      state.currentAutopilotController?.autopilot_controller_key === key
        ? applyUpdate(state.currentAutopilotController)
        : state.currentAutopilotController,
  };
}
