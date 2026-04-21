import { create } from "zustand";
import { useMemo } from "react";
import { getAcpManager } from "@/lib/wasm-core";
import type {
  AcpToolCall, AcpPlanStep, AcpPermissionRequest, AcpSessionState, AcpThinking, AcpLog,
} from "./acpSessionTypes";
import { EMPTY_SESSION, sessionFromWasm, toolCallToWasm, permReqToWasm, wasmFromSession } from "./acpSessionTypes";

export type { AcpToolCall, AcpPlanStep, AcpPermissionRequest, AcpSessionState, AcpThinking, AcpLog };

interface AcpSessionStore {
  _tick: number;
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
  clearSession: (podKey: string) => void;
}

const mgr = () => getAcpManager();
const bump = () => useAcpSessionStore.setState((s) => ({ _tick: s._tick + 1 }));

export const useAcpSessionStore = create<AcpSessionStore>(() => ({
  _tick: 0,

  addContentChunk: (podKey, _sid, text, role) => {
    mgr().add_content_chunk(podKey, text, role);
    bump();
  },

  markLastMessageComplete: (podKey) => {
    mgr().mark_last_message_complete(podKey);
    bump();
  },

  updateToolCall: (podKey, _sid, tc) => {
    mgr().update_tool_call(podKey, toolCallToWasm(tc));
    bump();
  },

  setToolCallResult: (podKey, _sid, id, success, resultText, errorMessage) => {
    mgr().set_tool_call_result(
      podKey, id, success,
      resultText || undefined,
      errorMessage || undefined,
    );
    bump();
  },

  updatePlan: (podKey, _sid, steps) => {
    mgr().update_plan(podKey, JSON.stringify(steps));
    bump();
  },

  addThinking: (podKey, _sid, text) => {
    mgr().add_thinking(podKey, text);
    bump();
  },

  addPermissionRequest: (podKey, req) => {
    mgr().add_permission_request(podKey, permReqToWasm(req));
    bump();
  },

  removePermissionRequest: (podKey, requestId) => {
    mgr().remove_permission_request(podKey, requestId);
    bump();
  },

  updateSessionState: (podKey, _sid, state) => {
    mgr().update_session_state(podKey, state);
    bump();
  },

  addLog: (podKey, level, message) => {
    mgr().add_log(podKey, level, message);
    bump();
  },

  clearSession: (podKey) => {
    mgr().clear_session(podKey);
    bump();
  },
}));

export function readAcpSession(podKey: string): AcpSessionState | null {
  const raw = mgr().get_session_json(podKey);
  if (!raw) return null;
  return sessionFromWasm(typeof raw === "string" ? JSON.parse(raw) : raw);
}

export function useAcpSession(podKey: string | null | undefined): AcpSessionState | null {
  const tick = useAcpSessionStore((s) => s._tick);
  return useMemo(() => {
    if (!podKey) return null;
    return readAcpSession(podKey);
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
    const s = readAcpSession(podKey);
    return pick(s ?? EMPTY_SESSION);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, podKey]);
}

/** Test-only helper: seed the WASM-backed session directly, bumping tick so
 *  subscribed components re-render. Use inside beforeEach/each test setup. */
export function __seedAcpSessionForTests(podKey: string, session: AcpSessionState): void {
  const mock = mgr() as unknown as { _seed: (k: string, s: unknown) => void };
  if (typeof mock._seed !== "function") {
    throw new Error("AcpManager mock does not support _seed — test-only API");
  }
  mock._seed(podKey, wasmFromSession(session));
  useAcpSessionStore.setState((s) => ({ _tick: s._tick + 1 }));
}

/** Test-only helper: reset the WASM-backed session store and tick. */
export function __resetAcpSessionsForTests(): void {
  const mock = mgr() as unknown as { _reset: () => void };
  if (typeof mock._reset !== "function") {
    throw new Error("AcpManager mock does not support _reset — test-only API");
  }
  mock._reset();
  useAcpSessionStore.setState({ _tick: 0 });
}
