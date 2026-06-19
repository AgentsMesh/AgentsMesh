import { invoke } from "./invoke";
import type { IAutopilotService } from "@agentsmesh/service-interface";
import { fromBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedControllersRequestSchema,
  SetCurrentControllerRequestSchema,
  InsertControllerRequestSchema,
  PatchControllerRequestSchema,
  ReplaceCachedIterationsRequestSchema,
  AppendIterationRequestSchema,
  UpdateThinkingRequestSchema,
  RemoveControllerRequestSchema,
  type AutopilotControllerSnapshot,
  type AutopilotIterationSnapshot,
} from "@agentsmesh/proto/autopilot_state/v1/autopilot_state_pb";
import {
  AutopilotControllerSchema,
  ListAutopilotControllersResponseSchema,
  GetIterationsResponseSchema,
} from "@agentsmesh/proto/autopilot/v1/autopilot_pb";
import { controllerToCache, iterationToCache } from "./autopilot_proto_to_cache";
import {
  controllersBytes, currentControllerBytes, controllerByPodKeyBytes, iterationsBytes,
} from "./autopilot_cache_to_bytes";

function snapshotToController(s: AutopilotControllerSnapshot): Record<string, unknown> {
  return {
    autopilot_controller_key: s.autopilotControllerKey,
    pod_key: s.podKey,
    status: s.status,
    phase: s.phase,
    prompt: s.prompt,
    max_iterations: s.maxIterations !== undefined ? Number(s.maxIterations) : undefined,
    iteration_timeout_sec: s.iterationTimeoutSec !== undefined ? Number(s.iterationTimeoutSec) : undefined,
    no_progress_threshold: s.noProgressThreshold !== undefined ? Number(s.noProgressThreshold) : undefined,
    same_error_threshold: s.sameErrorThreshold !== undefined ? Number(s.sameErrorThreshold) : undefined,
    approval_timeout_min: s.approvalTimeoutMin !== undefined ? Number(s.approvalTimeoutMin) : undefined,
    current_iteration: s.currentIteration !== undefined ? Number(s.currentIteration) : undefined,
    control_agent_slug: s.controlAgentSlug,
    circuit_breaker: (s.circuitBreakerState || s.circuitBreakerReason) ? {
      state: s.circuitBreakerState, reason: s.circuitBreakerReason,
    } : undefined,
    created_at: s.createdAt,
    updated_at: s.updatedAt,
  };
}

function snapshotToIteration(s: AutopilotIterationSnapshot): Record<string, unknown> {
  return {
    id: Number(s.id),
    controller_key: s.controllerKey,
    iteration_number: s.iterationNumber !== undefined ? Number(s.iterationNumber) : undefined,
    status: s.status,
    result: s.result,
    started_at: s.startedAt,
    completed_at: s.completedAt,
  };
}

export class ElectronAutopilotService implements IAutopilotService {
  private _controllersCache = "[]";
  private _currentControllerCache: string | null = null;
  private _iterationsCache = new Map<string, string>();
  private _thinkingCache = new Map<string, string>();
  private _thinkingHistoryCache = new Map<string, string>();

  controllers_json(): string { return this._controllersCache; }
  current_controller_json(): unknown { return this._currentControllerCache; }

  get_controller_by_pod_key_json(podKey: string): unknown {
    const ctrls = JSON.parse(this._controllersCache) as { pod_key: string }[];
    const c = ctrls.find(x => x.pod_key === podKey);
    return c ? JSON.stringify(c) : null;
  }

  get_iterations_json(key: string): unknown {
    return this._iterationsCache.get(key) ?? null;
  }

  get_thinking_json(key: string): unknown {
    return this._thinkingCache.get(key) ?? null;
  }

  get_thinking_history_json(key: string): unknown {
    return this._thinkingHistoryCache.get(key) ?? null;
  }

  // Proto-bytes mutators — decode locally + JS-cache sync, NAPI fire-and-forget.

  set_current_controller_proto(reqBytes: Uint8Array): void {
    const req = fromBinary(SetCurrentControllerRequestSchema, reqBytes);
    this._currentControllerCache = req.controller ? JSON.stringify(snapshotToController(req.controller)) : null;
    void invoke<void>("appAutopilotSetCurrentControllerProto", Array.from(reqBytes)).catch(() => undefined);
  }

  insert_controller(reqBytes: Uint8Array): void {
    const req = fromBinary(InsertControllerRequestSchema, reqBytes);
    if (req.controller) {
      const ctrls = JSON.parse(this._controllersCache) as { autopilot_controller_key?: string }[];
      const c = snapshotToController(req.controller);
      const idx = ctrls.findIndex(x => x.autopilot_controller_key === c.autopilot_controller_key);
      if (idx >= 0) ctrls[idx] = { ...ctrls[idx], ...c };
      else ctrls.push(c as { autopilot_controller_key: string });
      this._controllersCache = JSON.stringify(ctrls);
    }
    void invoke<void>("appAutopilotInsertController", Array.from(reqBytes)).catch(() => undefined);
  }

  patch_controller(reqBytes: Uint8Array): void {
    const req = fromBinary(PatchControllerRequestSchema, reqBytes);
    if (req.controller) {
      const ctrls = JSON.parse(this._controllersCache) as { autopilot_controller_key?: string }[];
      const idx = ctrls.findIndex(x => x.autopilot_controller_key === req.autopilotControllerKey);
      const c = snapshotToController(req.controller);
      if (idx >= 0) ctrls[idx] = { ...ctrls[idx], ...c };
      this._controllersCache = JSON.stringify(ctrls);
    }
    void invoke<void>("appAutopilotPatchController", Array.from(reqBytes)).catch(() => undefined);
  }

  remove_controller_proto(reqBytes: Uint8Array): void {
    const req = fromBinary(RemoveControllerRequestSchema, reqBytes);
    const ctrls = JSON.parse(this._controllersCache) as { autopilot_controller_key?: string }[];
    this._controllersCache = JSON.stringify(
      ctrls.filter(x => x.autopilot_controller_key !== req.autopilotControllerKey),
    );
    void invoke<void>("appAutopilotRemoveControllerProto", Array.from(reqBytes)).catch(() => undefined);
  }

  append_iteration(reqBytes: Uint8Array): void {
    const req = fromBinary(AppendIterationRequestSchema, reqBytes);
    if (req.iteration) {
      const iters = JSON.parse(this._iterationsCache.get(req.autopilotControllerKey) ?? "[]") as unknown[];
      iters.push(snapshotToIteration(req.iteration));
      this._iterationsCache.set(req.autopilotControllerKey, JSON.stringify(iters));
    }
    void invoke<void>("appAutopilotAppendIteration", Array.from(reqBytes)).catch(() => undefined);
  }

  update_thinking_proto(reqBytes: Uint8Array): void {
    const req = fromBinary(UpdateThinkingRequestSchema, reqBytes);
    this._thinkingCache.set(req.autopilotControllerKey, req.thinkingJson);
    void invoke<void>("appAutopilotUpdateThinkingProto", Array.from(reqBytes)).catch(() => undefined);
  }

  // Fetch→state (B): decode wire ListX response → renderer cache (sync, for
  // reactivity) + fire the SAME wire bytes to main so runtime.state (the SSOT
  // the realtime snapshot reads) folds the identical baseline. Mirrors the wasm
  // apply_fetched_* methods.
  apply_fetched_controllers(respBytes: Uint8Array): void {
    const resp = fromBinary(ListAutopilotControllersResponseSchema, respBytes);
    this._controllersCache = JSON.stringify(resp.items.map(controllerToCache));
    void invoke<void>("appAutopilotApplyFetchedControllers", Array.from(respBytes)).catch(() => undefined);
  }

  apply_fetched_iterations(key: string, respBytes: Uint8Array): void {
    const resp = fromBinary(GetIterationsResponseSchema, respBytes);
    this._iterationsCache.set(
      key,
      JSON.stringify(resp.items.map((it) => ({ ...iterationToCache(it), controller_key: key }))),
    );
    void invoke<void>("appAutopilotApplyFetchedIterations", key, Array.from(respBytes)).catch(() => undefined);
  }

  // Single-object fetch (B): decode wire GetAutopilotController response + upsert
  // into the cache + set current, fan the SAME wire bytes to main.
  apply_fetched_current_controller(respBytes: Uint8Array): void {
    const ctrl = controllerToCache(fromBinary(AutopilotControllerSchema, respBytes));
    const key = ctrl.autopilot_controller_key as string;
    const list = JSON.parse(this._controllersCache) as { autopilot_controller_key: string }[];
    const idx = list.findIndex((c) => c.autopilot_controller_key === key);
    if (idx >= 0) list[idx] = { ...list[idx], ...ctrl };
    else list.push(ctrl as { autopilot_controller_key: string });
    this._controllersCache = JSON.stringify(list);
    this._currentControllerCache = JSON.stringify(ctrl);
    void invoke<void>("appAutopilotApplyFetchedCurrentController", Array.from(respBytes)).catch(() => undefined);
  }

  // Read side (B, zero-JSON): re-encode the renderer cache into state proto bytes
  // so the shared selectors decode desktop and web identically.
  controllers_bytes(): Uint8Array { return controllersBytes(this._controllersCache); }
  current_controller_bytes(): Uint8Array { return currentControllerBytes(this._currentControllerCache); }
  controller_by_pod_key_bytes(podKey: string): Uint8Array {
    return controllerByPodKeyBytes(this._controllersCache, podKey);
  }
  iterations_bytes(key: string): Uint8Array {
    return iterationsBytes(key, this._iterationsCache.get(key) ?? "[]");
  }

  async fetch_controllers(): Promise<string> {
    const result = await invoke<string>("autopilotFetchControllers");
    const parsed = JSON.parse(result);
    this._controllersCache = JSON.stringify(parsed.controllers ?? parsed);
    return result;
  }

  async fetch_controller(key: string): Promise<string> {
    const result = await invoke<string>("autopilotFetchController", key);
    this._currentControllerCache = result;
    return result;
  }

  async fetch_iterations(key: string): Promise<string> {
    const result = await invoke<string>("autopilotFetchIterations", key);
    const parsed = JSON.parse(result);
    this._iterationsCache.set(key, JSON.stringify(parsed.iterations ?? parsed));
    return result;
  }

  async create_controller(json: string): Promise<string> {
    const result = await invoke<string>("autopilotCreateController", json);
    this._currentControllerCache = result;
    return result;
  }

  async approve_controller(key: string, json: string): Promise<void> {
    await invoke<void>("autopilotApproveController", key, json);
  }

  async pause_controller(key: string): Promise<void> {
    await invoke<void>("autopilotPauseController", key);
  }

  async resume_controller(key: string): Promise<void> {
    await invoke<void>("autopilotResumeController", key);
  }

  async stop_controller(key: string): Promise<void> {
    await invoke<void>("autopilotStopController", key);
  }

  async takeover_controller(key: string): Promise<void> {
    await invoke<void>("autopilotTakeoverController", key);
  }

  async handback_controller(key: string): Promise<void> {
    await invoke<void>("autopilotHandbackController", key);
  }

  // Realtime mirror: main pushes the controller list + affected key's iterations
  // as proto bytes, decoded via snapshotToController/Iteration for shape parity —
  // fixes the snapshot's flat circuit_breaker_state drift vs the mutators' nested
  // circuit_breaker. thinking/history stay JSON passthrough (no projection).
  apply_autopilot_snapshot(
    controllersBytes: Uint8Array,
    key: string,
    iterationsBytes: Uint8Array,
    thinkingJson: string,
    thinkingHistoryJson: string,
  ): void {
    const cReq = fromBinary(ReplaceCachedControllersRequestSchema, controllersBytes);
    this._controllersCache = JSON.stringify(cReq.controllers.map(snapshotToController));
    if (key && iterationsBytes.length) {
      const iReq = fromBinary(ReplaceCachedIterationsRequestSchema, iterationsBytes);
      this._iterationsCache.set(key, JSON.stringify(iReq.iterations.map(snapshotToIteration)));
    }
    if (key && thinkingJson) this._thinkingCache.set(key, thinkingJson);
    if (key && thinkingHistoryJson) this._thinkingHistoryCache.set(key, thinkingHistoryJson);
  }
}
