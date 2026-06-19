// Facade re-export of the autopilot Connect-RPC adapter. Business code
// imports from here (or from the `@/lib/api` barrel) so the wire-shape
// layer stays internal to the facade boundary. Tests mock this path.

export {
  listAutopilots,
  listAutopilotsRaw,
  getAutopilot,
  getAutopilotRaw,
  createAutopilot,
  pauseAutopilot,
  resumeAutopilot,
  stopAutopilot,
  takeoverAutopilot,
  handbackAutopilot,
  approveAutopilot,
  getAutopilotIterations,
  getAutopilotIterationsRaw,
  type AutopilotControllerWire,
  type AutopilotIterationWire,
  type CreateAutopilotParams,
} from "../connect/autopilotConnect";
