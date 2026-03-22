/**
 * ACP event dispatcher — routes relay messages to the acpSession store.
 *
 * Extracted from AgentPanel so that:
 * 1. Store-layer logic doesn't live in a UI component
 * 2. Other consumers (e.g. RealtimeProvider) can reuse the same dispatcher
 */

import { useAcpSessionStore } from "@/stores/acpSession";
import { MsgType } from "@/stores/relayProtocol";

/**
 * Dispatch an ACP relay event to the acpSessionStore.
 * Safe to call from any context — wraps all operations in try-catch.
 */
export function dispatchAcpRelayEvent(podKey: string, msgType: number, payload: unknown): void {
  try {
    const data = payload as Record<string, unknown>;
    const store = useAcpSessionStore.getState();
    const sessionId = (data.session_id as string) || "";

    if (msgType === MsgType.AcpEvent) {
      dispatchEvent(store, podKey, sessionId, data);
    } else if (msgType === MsgType.AcpSnapshot) {
      dispatchSnapshot(store, podKey, sessionId, data);
    }
  } catch (err) {
    console.error("[ACP] Failed to dispatch relay event", { podKey, msgType, err });
  }
}

type AcpStore = ReturnType<typeof useAcpSessionStore.getState>;

/** Route a single ACP event to the appropriate store mutation. */
function dispatchEvent(
  store: AcpStore,
  podKey: string,
  sessionId: string,
  data: Record<string, unknown>,
): void {
  const eventType = data.type as string;

  switch (eventType) {
    case "content_chunk":
      store.addContentChunk(podKey, sessionId, data.text as string, data.role as string);
      break;
    case "tool_call_update":
      store.updateToolCall(podKey, sessionId, data as unknown as Parameters<AcpStore["updateToolCall"]>[2]);
      break;
    case "tool_call_result":
      store.setToolCallResult(
        podKey, sessionId,
        data.tool_call_id as string,
        data.success as boolean,
        data.result_text as string,
        data.error_message as string,
      );
      break;
    case "plan_update":
      store.updatePlan(podKey, sessionId, data.steps as Parameters<AcpStore["updatePlan"]>[2]);
      break;
    case "thinking_update":
      store.addThinking(podKey, sessionId, data.text as string);
      break;
    case "permission_request":
      store.addPermissionRequest(podKey, {
        request_id: data.request_id as string,
        tool_name: data.tool_name as string,
        arguments_json: data.arguments_json as string,
        description: data.description as string,
      });
      break;
    case "session_state":
      store.updateSessionState(podKey, sessionId, data.state as string);
      if (data.state === "idle") {
        store.markLastMessageComplete(podKey);
      }
      break;
    case "log":
      // Agent-level logs (debug info, not shown in activity stream by default)
      if (data.level === "error" || data.level === "warn") {
        console.warn(`[ACP:${podKey}] ${data.level}: ${data.message}`);
      }
      break;
    default:
      console.warn("[ACP] Unknown event type:", eventType);
  }
}

/** Replay a full session snapshot into the store. */
function dispatchSnapshot(
  store: AcpStore,
  podKey: string,
  sessionId: string,
  data: Record<string, unknown>,
): void {
  // Clear first, then replay in order: state → plan → tool_calls → messages → permissions.
  store.clearSession(podKey);

  if (data.state) {
    store.updateSessionState(podKey, sessionId, data.state as string);
  }
  if (Array.isArray(data.plan)) {
    store.updatePlan(podKey, sessionId, data.plan as Parameters<AcpStore["updatePlan"]>[2]);
  }
  // Replay tool calls from snapshot (includes status + result in one object)
  if (Array.isArray(data.tool_calls)) {
    for (const tc of data.tool_calls as Array<{
      tool_call_id: string;
      tool_name: string;
      status: string;
      arguments_json: string;
      success?: boolean;
      result_text?: string;
      error_message?: string;
    }>) {
      store.updateToolCall(podKey, sessionId, tc as Parameters<AcpStore["updateToolCall"]>[2]);
      if (tc.success !== undefined && tc.success !== null) {
        store.setToolCallResult(
          podKey, sessionId,
          tc.tool_call_id,
          tc.success,
          tc.result_text ?? "",
          tc.error_message ?? "",
        );
      }
    }
  }
  if (Array.isArray(data.messages)) {
    for (const msg of data.messages as Array<{ text: string; role: string }>) {
      store.addContentChunk(podKey, sessionId, msg.text, msg.role);
    }
  }
  if (Array.isArray(data.pending_permissions)) {
    for (const perm of data.pending_permissions as Array<{
      request_id: string;
      tool_name: string;
      arguments_json: string;
      description: string;
    }>) {
      store.addPermissionRequest(podKey, perm);
    }
  }
}
