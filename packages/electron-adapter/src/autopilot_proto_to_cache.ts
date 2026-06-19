// wire(proto.autopilot.v1) → renderer cache shape, the desktop mirror of the
// web fetch path (Rust wire_controller_to_state ∘ controllerSnapshotToCache).
// Produces the SAME cache the wasm read side yields: wire-only id / user_takeover
// / started_at / last_iteration_at drop, circuit_breaker stays nested, so the
// shared selectors decode desktop and web identically.
import type {
  AutopilotController as WireController,
  AutopilotIteration as WireIteration,
} from "@agentsmesh/proto/autopilot/v1/autopilot_pb";

export function controllerToCache(c: WireController): Record<string, unknown> {
  return {
    id: 0,
    autopilot_controller_key: c.autopilotControllerKey,
    pod_key: c.podKey,
    phase: c.phase,
    current_iteration: c.currentIteration,
    max_iterations: c.maxIterations,
    circuit_breaker: {
      state: c.circuitBreaker?.state || "closed",
      reason: c.circuitBreaker?.reason || undefined,
    },
    user_takeover: false,
    prompt: c.prompt || undefined,
    created_at: c.createdAt,
  };
}

// controller_key comes from the caller's fetch key (set by the service), not the
// wire field — same as Rust wire_iteration_to_state.
export function iterationToCache(i: WireIteration): Record<string, unknown> {
  return {
    id: Number(i.id),
    autopilot_controller_id: 0,
    iteration: Number(i.iterationNumber),
    phase: i.status,
    summary: i.result || undefined,
    created_at: i.startedAt ?? "",
  };
}
