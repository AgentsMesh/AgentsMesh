import { useMemo } from "react";
import { fromBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedControllersRequestSchema,
  ReplaceCachedIterationsRequestSchema,
  SetCurrentControllerRequestSchema,
} from "@proto/autopilot_state/v1/autopilot_state_pb";
import type { AutopilotControllerData, AutopilotIterationData } from "@/lib/viewModels/autopilot";
import type { AutopilotThinkingData } from "@/lib/realtime/types";
import { getAutopilotState, parseWasmAny } from "@/lib/wasm-core";
import { controllerSnapshotToCache, iterationSnapshotToCache } from "./autopilotSnapshotToCache";
import { useAutopilotStore, ACTIVE } from "./autopilot";

type Ctrl = AutopilotControllerData;
const svc = () => getAutopilotState();

// Read side, zero-JSON: decode state bytes → controllerSnapshotToCache. Replaces
// controllers_json (flat state struct serde that dropped the nested
// circuit_breaker the UI reads).
export function useAutopilotControllers(): Ctrl[] {
  const tick = useAutopilotStore((s) => s._tick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(
    () => fromBinary(ReplaceCachedControllersRequestSchema, svc().controllers_bytes())
      .controllers.map(controllerSnapshotToCache),
    [tick],
  );
}

export function useCurrentAutopilotController(): Ctrl | null {
  const tick = useAutopilotStore((s) => s._tick);
  return useMemo(() => {
    const bytes = svc().current_controller_bytes();
    if (!bytes.length) return null;
    const c = fromBinary(SetCurrentControllerRequestSchema, bytes).controller;
    return c ? controllerSnapshotToCache(c) : null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick]);
}

export function useAutopilotControllerByPodKey(podKey: string): Ctrl | undefined {
  const tick = useAutopilotStore((s) => s._tick);
  return useMemo(() => {
    const bytes = svc().controller_by_pod_key_bytes(podKey);
    if (!bytes.length) return undefined;
    const c = fromBinary(SetCurrentControllerRequestSchema, bytes).controller;
    const ctrl = c ? controllerSnapshotToCache(c) : undefined;
    // Only surface a controller for the pod when it's in an active phase —
    // a terminated controller must not light up the autopilot overlay.
    return ctrl && ACTIVE.includes(ctrl.phase) ? ctrl : undefined;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, podKey]);
}

export function useAutopilotIterations(key: string | null | undefined): AutopilotIterationData[] {
  const tick = useAutopilotStore((s) => s._tick);
  return useMemo(() => {
    if (!key) return [];
    const bytes = svc().iterations_bytes(key);
    if (!bytes.length) return [];
    return fromBinary(ReplaceCachedIterationsRequestSchema, bytes).iterations.map(iterationSnapshotToCache);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, key]);
}

export function useAutopilotThinking(key: string | null | undefined): AutopilotThinkingData | null {
  const tick = useAutopilotStore((s) => s._tick);
  return useMemo(() => {
    if (!key) return null;
    return parseWasmAny<AutopilotThinkingData>(svc().get_thinking_json(key)) ?? null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, key]);
}

export function useAutopilotThinkingHistory(key: string | null | undefined): AutopilotThinkingData[] {
  const tick = useAutopilotStore((s) => s._tick);
  return useMemo(() => {
    if (!key) return [];
    return parseWasmAny<AutopilotThinkingData[]>(svc().get_thinking_history_json(key)) ?? [];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, key]);
}
