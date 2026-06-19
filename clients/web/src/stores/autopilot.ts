import { create } from "zustand";
import { create as protoCreate, toBinary, fromBinary } from "@bufbuild/protobuf";
import type {
  AutopilotControllerData, AutopilotIterationData,
  CreateAutopilotControllerRequest, ApproveRequest,
} from "@/lib/viewModels/autopilot";
import { AutopilotThinkingData } from "@/lib/realtime/types";
import { reconnectRegistry } from "@/lib/realtime";
import { getErrorMessage } from "@/lib/utils";
import { getAutopilotState, parseWasmAny } from "@/lib/wasm-core";
import { readCurrentOrg } from "@/stores/auth";
import {
  listAutopilotsRaw,
  getAutopilotRaw,
  createAutopilot as createAutopilotConnect,
  pauseAutopilot as pauseAutopilotConnect,
  resumeAutopilot as resumeAutopilotConnect,
  stopAutopilot as stopAutopilotConnect,
  approveAutopilot as approveAutopilotConnect,
  takeoverAutopilot as takeoverAutopilotConnect,
  handbackAutopilot as handbackAutopilotConnect,
  getAutopilotIterationsRaw,
  type AutopilotControllerWire,
} from "@/lib/api/facade/autopilotConnect";
import {
  SetCurrentControllerRequestSchema,
  InsertControllerRequestSchema,
  PatchControllerRequestSchema,
  AppendIterationRequestSchema,
  UpdateThinkingRequestSchema,
  RemoveControllerRequestSchema,
  ReplaceCachedControllersRequestSchema,
} from "@proto/autopilot_state/v1/autopilot_state_pb";
import { controllerToProto, iterationToProto } from "@/lib/api/autopilotProtoMap";
import { controllerSnapshotToCache } from "./autopilotSnapshotToCache";

export type AutopilotController = AutopilotControllerData;
export type AutopilotIteration = AutopilotIterationData;
export type AutopilotThinking = AutopilotThinkingData;
export type { CreateAutopilotControllerRequest, ApproveRequest };
export {
  useAutopilotControllers, useCurrentAutopilotController, useAutopilotControllerByPodKey,
  useAutopilotIterations, useAutopilotThinking, useAutopilotThinkingHistory,
} from "./autopilotSelectors";

type Ctrl = AutopilotController;
export const ACTIVE = ["initializing", "running", "paused", "user_takeover", "waiting_approval"];
// Autopilot state SSOT is the shared AppState (runtime.state) via
// getAutopilotState — the SAME state the EventBus dispatch + desktop snapshot
// mirror write, so realtime controller/iteration/thinking changes flow without
// a JS pure-patch. Connect-RPC stays on the autopilotConnect facade.
const svc = () => getAutopilotState();
const bump = () => useAutopilotStore.setState((s) => ({ _tick: s._tick + 1 }));
const slug = () => readCurrentOrg()?.slug ?? "";

const fromWireCtrl = (w: AutopilotControllerWire): Ctrl => w as unknown as Ctrl;

function dispatchSetCurrentController(c: Ctrl | null) {
  const req = protoCreate(SetCurrentControllerRequestSchema, {
    controller: c ? controllerToProto(c) : undefined,
  });
  svc().set_current_controller_proto(toBinary(SetCurrentControllerRequestSchema, req));
}

function dispatchInsertController(c: Ctrl) {
  const req = protoCreate(InsertControllerRequestSchema, { controller: controllerToProto(c) });
  svc().insert_controller(toBinary(InsertControllerRequestSchema, req));
}

function dispatchPatchController(key: string, c: Ctrl) {
  const req = protoCreate(PatchControllerRequestSchema, {
    autopilotControllerKey: key,
    controller: controllerToProto(c),
  });
  svc().patch_controller(toBinary(PatchControllerRequestSchema, req));
}

function dispatchAppendIteration(key: string, it: AutopilotIteration) {
  const req = protoCreate(AppendIterationRequestSchema, {
    autopilotControllerKey: key,
    iteration: iterationToProto(it),
  });
  svc().append_iteration(toBinary(AppendIterationRequestSchema, req));
}

function dispatchUpdateThinking(key: string, thinking: AutopilotThinking) {
  const req = protoCreate(UpdateThinkingRequestSchema, {
    autopilotControllerKey: key,
    thinkingJson: JSON.stringify(thinking),
  });
  svc().update_thinking_proto(toBinary(UpdateThinkingRequestSchema, req));
}

function dispatchRemoveController(key: string) {
  const req = protoCreate(RemoveControllerRequestSchema, { autopilotControllerKey: key });
  svc().remove_controller_proto(toBinary(RemoveControllerRequestSchema, req));
}

function patchCtrl(key: string, patch: Partial<Ctrl> | ((c: Ctrl) => Partial<Ctrl>)) {
  // Read the full snapshot via bytes (controllers_json is a lossy flat serde
  // that drops nested circuit_breaker) so the optimistic patch round-trip keeps
  // nested fields intact.
  const ctrls = fromBinary(ReplaceCachedControllersRequestSchema, svc().controllers_bytes())
    .controllers.map(controllerSnapshotToCache);
  const target = ctrls.find((c) => c.autopilot_controller_key === key);
  if (!target) return;
  const next = { ...target, ...(typeof patch === "function" ? patch(target) : patch) };
  dispatchPatchController(key, next);
}

interface AutopilotState { _tick: number; loading: boolean; error: string | null; }
interface AutopilotActions {
  fetchAutopilotControllers: () => Promise<void>;
  fetchAutopilotController: (key: string) => Promise<void>;
  createAutopilotController: (data: CreateAutopilotControllerRequest) => Promise<Ctrl>;
  pauseAutopilotController: (key: string) => Promise<void>;
  resumeAutopilotController: (key: string) => Promise<void>;
  stopAutopilotController: (key: string) => Promise<void>;
  approveAutopilotController: (key: string, data?: ApproveRequest) => Promise<void>;
  takeoverAutopilotController: (key: string) => Promise<void>;
  handbackAutopilotController: (key: string) => Promise<void>;
  fetchIterations: (key: string) => Promise<void>;
  updateAutopilotControllerStatus: (key: string, phase: string, cur: number, max: number, cbState: string, cbReason?: string) => void;
  addIteration: (key: string, iteration: AutopilotIteration) => void;
  updateThinking: (key: string, thinking: AutopilotThinking) => void;
  setCurrentAutopilotController: (c: Ctrl | null) => void;
  removeAutopilotController: (key: string) => void;
  clearError: () => void;
  getAutopilotControllerByPodKey: (podKey: string) => Ctrl | undefined;
}

export const useAutopilotStore = create<AutopilotState & AutopilotActions>((set, get) => {
  const action = (
    fn: (s: string, k: string) => Promise<unknown>, msg: string,
    patch: Partial<Ctrl> | ((c: Ctrl) => Partial<Ctrl>),
  ) => async (key: string) => {
    try { await fn(slug(), key); patchCtrl(key, patch); bump(); }
    catch (e) { set({ error: getErrorMessage(e, msg) }); }
  };

  return {
  _tick: 0, loading: false, error: null,

  fetchAutopilotControllers: async () => {
    set({ loading: true, error: null });
    try {
      svc().apply_fetched_controllers(await listAutopilotsRaw(slug()));
      set({ loading: false, _tick: get()._tick + 1 });
    } catch (e) { set({ error: getErrorMessage(e, "Failed to fetch controllers"), loading: false }); }
  },

  fetchAutopilotController: async (key) => {
    try {
      svc().apply_fetched_current_controller(await getAutopilotRaw(slug(), key));
      bump();
    } catch (e) { set({ error: getErrorMessage(e, "Failed to fetch controller") }); }
  },

  createAutopilotController: async (data) => {
    try {
      const ctrl = fromWireCtrl(await createAutopilotConnect({
        orgSlug: slug(), podKey: data.pod_key, prompt: data.prompt,
        maxIterations: data.max_iterations, iterationTimeoutSec: data.iteration_timeout_sec,
        noProgressThreshold: data.no_progress_threshold, sameErrorThreshold: data.same_error_threshold,
        approvalTimeoutMin: data.approval_timeout_min, controlAgentSlug: data.control_agent_slug,
        controlPromptTemplate: data.control_prompt_template, mcpConfigJson: data.mcp_config_json,
      }));
      dispatchInsertController(ctrl);
      dispatchSetCurrentController(ctrl);
      bump();
      return ctrl;
    } catch (e) { set({ error: getErrorMessage(e, "Failed to create") }); throw e; }
  },

  pauseAutopilotController: action(pauseAutopilotConnect, "Failed to pause", { phase: "paused" }),
  resumeAutopilotController: action(resumeAutopilotConnect, "Failed to resume", { phase: "running" }),
  stopAutopilotController: action(stopAutopilotConnect, "Failed to stop", { phase: "stopped" }),
  takeoverAutopilotController: action(takeoverAutopilotConnect, "Failed to takeover", { phase: "user_takeover", user_takeover: true }),
  handbackAutopilotController: action(handbackAutopilotConnect, "Failed to handback", { phase: "running", user_takeover: false }),

  approveAutopilotController: async (key, data) => {
    try {
      await approveAutopilotConnect(slug(), key, data?.continue_execution, data?.additional_iterations);
      patchCtrl(key, (c) => ({
        phase: data?.continue_execution === false ? "stopped" : "running",
        max_iterations: data?.additional_iterations ? c.max_iterations + data.additional_iterations : c.max_iterations,
      }));
      bump();
    } catch (e) { set({ error: getErrorMessage(e, "Failed to approve") }); }
  },

  fetchIterations: async (key) => {
    try {
      svc().apply_fetched_iterations(key, await getAutopilotIterationsRaw(slug(), key));
      bump();
    } catch (e) { set({ error: getErrorMessage(e, "Failed to fetch iterations") }); }
  },

  updateAutopilotControllerStatus: (key, phase, cur, max, cbState, cbReason) => {
    patchCtrl(key, {
      phase: phase as Ctrl["phase"], current_iteration: cur, max_iterations: max,
      circuit_breaker: { state: cbState as Ctrl["circuit_breaker"]["state"], reason: cbReason },
    });
    bump();
  },

  addIteration: (key, iter) => { dispatchAppendIteration(key, iter); bump(); },
  updateThinking: (key, t) => { dispatchUpdateThinking(key, t); bump(); },
  setCurrentAutopilotController: (c) => { dispatchSetCurrentController(c); bump(); },
  removeAutopilotController: (key) => { dispatchRemoveController(key); bump(); },
  clearError: () => set({ error: null }),

  getAutopilotControllerByPodKey: (podKey) => {
    const bytes = svc().controller_by_pod_key_bytes(podKey);
    if (!bytes.length) return undefined;
    const c = fromBinary(SetCurrentControllerRequestSchema, bytes).controller;
    const val = c ? controllerSnapshotToCache(c) : undefined;
    return val && ACTIVE.includes(val.phase) ? val : undefined;
  },
  };
});

reconnectRegistry.register({
  name: "autopilot:controllers",
  fn: () => useAutopilotStore.getState().fetchAutopilotControllers?.(),
  priority: "low",
});
