import { create } from "zustand";
import { useMemo } from "react";
import { create as protoCreate, toBinary } from "@bufbuild/protobuf";
import { getAcpManager } from "@/lib/wasm-core";
import type {
  AcpToolCall, AcpPlanStep, AcpPermissionRequest, AcpSessionState, AcpThinking, AcpLog, AcpConfiguration,
} from "./acpSessionTypes";
import { EMPTY_SESSION, sessionFromWasm, toolCallToWasm, permReqToWasm, wasmFromSession } from "./acpSessionTypes";
import {
  UpdateToolCallRequestSchema,
  UpdatePlanRequestSchema,
  AddPermissionRequestRequestSchema,
  UpdateConfigurationRequestSchema,
} from "@proto/acp_state/v1/acp_state_pb";

export type { AcpToolCall, AcpPlanStep, AcpPermissionRequest, AcpSessionState, AcpThinking, AcpLog, AcpConfiguration };

interface AcpSessionStore {
  _tick: number;
  cache: Record<string, AcpSessionState>;
  addContentChunk: (podKey: string, sessionId: string, text: string, role: string) => void;
  markLastMessageComplete: (podKey: string) => void;
  updateToolCall: (podKey: string, sessionId: string, toolCall: AcpToolCall) => void;
  setToolCallResult: (podKey: string, sessionId: string, toolCallId: string, success: boolean, resultText: string, errorMessage: string) => void;
  updatePlan: (podKey: string, sessionId: string, steps: AcpPlanStep[]) => void;
  addThinking: (podKey: string, sessionId: string, text: string) => void;
  addPermissionRequest: (podKey: string, req: AcpPermissionRequest) => void;
  removePermissionRequest: (podKey: string, requestId: string) => void;
  updateSessionState: (podKey: string, sessionId: string, state: string) => void;
  addLog: (podKey: string, level: string, message: string) => void;
  updateConfiguration: (podKey: string, configuration: Partial<AcpConfiguration>) => void;
  clearSession: (podKey: string) => void;
}

const mgr = () => getAcpManager();

function readSessionFromWasm(podKey: string): AcpSessionState | null {
  try {
    const raw = mgr().get_session_json(podKey);
    if (!raw) return null;
    return sessionFromWasm(typeof raw === "string" ? JSON.parse(raw) : raw);
  } catch (err) {
    console.error("[acpSession] wasm read failed", { podKey, err });
    return null;
  }
}

function refreshCache(podKey: string): void {
  const session = readSessionFromWasm(podKey);
  useAcpSessionStore.setState((s) => {
    const next = { ...s.cache };
    if (session) next[podKey] = session;
    else delete next[podKey];
    return { _tick: s._tick + 1, cache: next };
  });
}

function dropFromCache(podKey: string): void {
  useAcpSessionStore.setState((s) => {
    if (!(podKey in s.cache)) return { _tick: s._tick + 1 };
    const next = { ...s.cache };
    delete next[podKey];
    return { _tick: s._tick + 1, cache: next };
  });
}

export const useAcpSessionStore = create<AcpSessionStore>(() => ({
  _tick: 0,
  cache: {},

  addContentChunk: (podKey, _sid, text, role) => {
    mgr().add_content_chunk(podKey, text, role);
    refreshCache(podKey);
  },

  markLastMessageComplete: (podKey) => {
    mgr().mark_last_message_complete(podKey);
    refreshCache(podKey);
  },

  updateToolCall: (podKey, _sid, tc) => {
    const req = protoCreate(UpdateToolCallRequestSchema, {
      podKey,
      toolCallJson: toolCallToWasm(tc),
    });
    mgr().update_tool_call(toBinary(UpdateToolCallRequestSchema, req));
    refreshCache(podKey);
  },

  setToolCallResult: (podKey, _sid, id, success, resultText, errorMessage) => {
    mgr().set_tool_call_result(
      podKey, id, success,
      resultText || undefined,
      errorMessage || undefined,
    );
    refreshCache(podKey);
  },

  updatePlan: (podKey, _sid, steps) => {
    const req = protoCreate(UpdatePlanRequestSchema, {
      podKey,
      stepsJson: JSON.stringify(steps),
    });
    mgr().update_plan(toBinary(UpdatePlanRequestSchema, req));
    refreshCache(podKey);
  },

  addThinking: (podKey, _sid, text) => {
    mgr().add_thinking(podKey, text);
    refreshCache(podKey);
  },

  addPermissionRequest: (podKey, req) => {
    const protoReq = protoCreate(AddPermissionRequestRequestSchema, {
      podKey,
      requestJson: permReqToWasm(req),
    });
    mgr().add_permission_request(toBinary(AddPermissionRequestRequestSchema, protoReq));
    refreshCache(podKey);
  },

  removePermissionRequest: (podKey, requestId) => {
    mgr().remove_permission_request(podKey, requestId);
    refreshCache(podKey);
  },

  updateSessionState: (podKey, _sid, state) => {
    mgr().update_session_state(podKey, state);
    refreshCache(podKey);
  },

  addLog: (podKey, level, message) => {
    mgr().add_log(podKey, level, message);
    refreshCache(podKey);
  },

  updateConfiguration: (podKey, configuration) => {
    const req = protoCreate(UpdateConfigurationRequestSchema, {
      podKey,
      configJson: JSON.stringify({
        permission_mode: configuration.permissionMode ?? "",
        model: configuration.model ?? "",
        // Capability flows via snapshot only; configChanged omits it so core's
        // merge guard preserves the seeded value (empty = "unchanged").
        ...(configuration.supportedPermissionModes !== undefined
          ? { supported_permission_modes: configuration.supportedPermissionModes }
          : {}),
      }),
    });
    mgr().update_configuration(toBinary(UpdateConfigurationRequestSchema, req));
    refreshCache(podKey);
  },

  clearSession: (podKey) => {
    mgr().clear_session(podKey);
    dropFromCache(podKey);
  },
}));

export function readAcpSession(podKey: string): AcpSessionState | null {
  return useAcpSessionStore.getState().cache[podKey] ?? null;
}

export function useAcpSession(podKey: string | null | undefined): AcpSessionState | null {
  const tick = useAcpSessionStore((s) => s._tick);
  return useMemo(() => {
    if (!podKey) return null;
    return useAcpSessionStore.getState().cache[podKey] ?? null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, podKey]);
}

export function useAcpSessionField<T>(
  podKey: string | null | undefined,
  pick: (s: AcpSessionState) => T,
): T {
  const tick = useAcpSessionStore((s) => s._tick);
  return useMemo(() => {
    if (!podKey) return pick(EMPTY_SESSION);
    const session = useAcpSessionStore.getState().cache[podKey];
    return pick(session ?? EMPTY_SESSION);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, podKey]);
}

export function __seedAcpSessionForTests(podKey: string, session: AcpSessionState): void {
  const mock = mgr() as unknown as { _seed: (k: string, s: unknown) => void };
  if (typeof mock._seed !== "function") {
    throw new Error("AcpManager mock does not support _seed — test-only API");
  }
  mock._seed(podKey, wasmFromSession(session));
  refreshCache(podKey);
}

export function __resetAcpSessionsForTests(): void {
  const mock = mgr() as unknown as { _reset: () => void };
  if (typeof mock._reset !== "function") {
    throw new Error("AcpManager mock does not support _reset — test-only API");
  }
  mock._reset();
  useAcpSessionStore.setState({ _tick: 0, cache: {} });
}
