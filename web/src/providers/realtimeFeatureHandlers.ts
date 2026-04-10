import { useAutopilotStore } from "@/stores/autopilot";
import { useLoopStore } from "@/stores/loop";
import type { DebounceRef } from "./realtimeEventHandlers";
import type {
  RealtimeEvent,
  AutopilotStatusChangedData, AutopilotIterationData,
  AutopilotTerminatedData, AutopilotThinkingData,
  LoopRunEventData, LoopRunWarningData,
} from "@/lib/realtime";

export function handleAutopilotEvent(event: RealtimeEvent) {
  const store = useAutopilotStore.getState();
  switch (event.type) {
    case "autopilot:status_changed": {
      const data = event.data as AutopilotStatusChangedData;
      store.updateAutopilotControllerStatus(data.autopilot_controller_key, data.phase, data.current_iteration, data.max_iterations, data.circuit_breaker_state, data.circuit_breaker_reason);
      break;
    }
    case "autopilot:iteration": {
      const data = event.data as AutopilotIterationData;
      store.addIteration(data.autopilot_controller_key, {
        id: 0, autopilot_controller_id: 0, iteration: data.iteration, phase: data.phase,
        summary: data.summary, files_changed: data.files_changed, duration_ms: data.duration_ms,
        created_at: new Date().toISOString(),
      });
      break;
    }
    case "autopilot:created": {
      store.fetchAutopilotControllers?.();
      break;
    }
    case "autopilot:terminated": {
      const data = event.data as AutopilotTerminatedData;
      store.removeAutopilotController(data.autopilot_controller_key);
      break;
    }
    case "autopilot:thinking": {
      const data = event.data as AutopilotThinkingData;
      store.updateThinking(data.autopilot_controller_key, data);
      break;
    }
  }
}

export function handleLoopEvent(
  event: RealtimeEvent,
  debounceRef: DebounceRef | undefined,
  t: (key: string, params?: Record<string, string | number>) => string,
  showWarning: (title: string, description: string) => void
) {
  switch (event.type) {
    case "loop_run:started":
    case "loop_run:completed":
    case "loop_run:failed": {
      if (!debounceRef) return;
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        debounceRef.current = null;
        const s = useLoopStore.getState();
        s.fetchLoops?.();
        if (s.currentLoop?.id === (event.data as LoopRunEventData).loop_id) {
          s.fetchLoop?.(s.currentLoop.slug);
          useLoopStore.setState({ runsOffset: 0 });
          s.fetchRuns?.(s.currentLoop.slug, { limit: 20, offset: 0 });
        }
      }, 500);
      break;
    }
    case "loop_run:warning": {
      const data = event.data as LoopRunWarningData;
      showWarning(t("loops.runWarningTitle", { runNumber: data.run_number }), data.detail || data.warning);
      break;
    }
  }
}
