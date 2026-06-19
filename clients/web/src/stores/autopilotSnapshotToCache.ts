// proto.autopilot_state.v1 snapshot → renderer view-model (AutopilotControllerData
// / AutopilotIterationData). The read-side mirror of the desktop adapter's
// snapshotToController/Iteration (electron-adapter/src/autopilot.ts) so web and
// desktop decode the SAME state bytes into the SAME cache shape — notably
// re-folding the flat circuit_breaker_state/reason into the nested
// circuit_breaker the UI reads.
import type {
  AutopilotControllerSnapshot, AutopilotIterationSnapshot,
} from "@proto/autopilot_state/v1/autopilot_state_pb";
import type {
  AutopilotControllerData, AutopilotIterationData,
  AutopilotPhase, CircuitBreakerState,
} from "@/lib/viewModels/autopilot";

const optNum = (v: bigint | undefined): number | undefined =>
  v !== undefined ? Number(v) : undefined;

export function controllerSnapshotToCache(s: AutopilotControllerSnapshot): AutopilotControllerData {
  return {
    id: 0,
    autopilot_controller_key: s.autopilotControllerKey,
    pod_key: s.podKey,
    phase: (s.phase ?? "") as AutopilotPhase,
    current_iteration: optNum(s.currentIteration) ?? 0,
    max_iterations: optNum(s.maxIterations) ?? 0,
    circuit_breaker: {
      state: (s.circuitBreakerState ?? "closed") as CircuitBreakerState,
      reason: s.circuitBreakerReason,
    },
    user_takeover: false,
    prompt: s.prompt,
    created_at: s.createdAt ?? "",
  };
}

export function iterationSnapshotToCache(s: AutopilotIterationSnapshot): AutopilotIterationData {
  return {
    id: Number(s.id),
    autopilot_controller_id: 0,
    iteration: optNum(s.iterationNumber) ?? 0,
    phase: s.status ?? "",
    summary: s.result,
    created_at: s.startedAt ?? "",
  };
}
