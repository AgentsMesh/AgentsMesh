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
    const sessionId = (data.sessionId as string) || "";

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
    case "contentChunk":
      store.addContentChunk(podKey, sessionId, data.text as string, data.role as string);
      break;
    case "toolCallUpdate":
      store.updateToolCall(podKey, sessionId, data as unknown as Parameters<AcpStore["updateToolCall"]>[2]);
      break;
    case "toolCallResult":
      store.setToolCallResult(
        podKey, sessionId,
        data.toolCallId as string,
        data.success as boolean,
        data.resultText as string,
        data.errorMessage as string,
      );
      break;
    case "planUpdate":
      store.updatePlan(podKey, sessionId, data.steps as Parameters<AcpStore["updatePlan"]>[2]);
      break;
    case "thinkingUpdate":
      store.addThinking(podKey, sessionId, data.text as string);
      break;
    case "permissionRequest":
      store.addPermissionRequest(podKey, {
        requestId: data.requestId as string,
        toolName: data.toolName as string,
        argumentsJson: data.argumentsJson as string,
        description: data.description as string,
      });
      break;
    case "sessionState":
      store.updateSessionState(podKey, sessionId, data.state as string);
      if (data.state === "idle") {
        store.markLastMessageComplete(podKey);
      }
      break;
    case "log":
      if (data.level === "error" || data.level === "warn") {
        console.warn(`[ACP:${podKey}] ${data.level}: ${data.message}`);
        store.addLog(podKey, data.level as string, data.message as string);
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
  // Clear first, then replay in order: state -> plan -> toolCalls -> messages -> permissions.
  store.clearSession(podKey);

  if (data.state) {
    store.updateSessionState(podKey, sessionId, data.state as string);
  }
  if (Array.isArray(data.plan)) {
    store.updatePlan(podKey, sessionId, data.plan as Parameters<AcpStore["updatePlan"]>[2]);
  }
  // Replay tool calls from snapshot (includes status + result in one object)
  if (Array.isArray(data.toolCalls)) {
    for (const tc of data.toolCalls as Array<{
      toolCallId: string;
      toolName: string;
      status: string;
      argumentsJson: string;
      success?: boolean;
      resultText?: string;
      errorMessage?: string;
    }>) {
      store.updateToolCall(podKey, sessionId, tc as Parameters<AcpStore["updateToolCall"]>[2]);
      if (tc.success !== undefined && tc.success !== null) {
        store.setToolCallResult(
          podKey, sessionId,
          tc.toolCallId,
          tc.success,
          tc.resultText ?? "",
          tc.errorMessage ?? "",
        );
      }
    }
  }
  if (Array.isArray(data.messages)) {
    for (const msg of data.messages as Array<{ text: string; role: string }>) {
      store.addContentChunk(podKey, sessionId, msg.text, msg.role);
    }
  }
  if (Array.isArray(data.pendingPermissions)) {
    for (const perm of data.pendingPermissions as Array<{
      requestId: string;
      toolName: string;
      argumentsJson: string;
      description: string;
    }>) {
      store.addPermissionRequest(podKey, perm);
    }
  }
}
