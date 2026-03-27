/**
 * Relay message handlers — extracted from RelayConnectionPool for SRP.
 * Each handler processes a specific message type from the relay WebSocket.
 */

import type { RelayConnection } from "./relayConnectionTypes";
import { MsgType, decodeMessage, decodeJsonPayload } from "./relayProtocol";

/**
 * Dispatch a decoded relay message to the appropriate handler.
 */
export function dispatchRelayMessage(
  conn: RelayConnection,
  data: ArrayBuffer | string,
  callbacks: {
    onSnapshot: (conn: RelayConnection, payload: Uint8Array) => void;
    onControl: (conn: RelayConnection, payload: Uint8Array) => void;
    onRunnerDisconnected: (conn: RelayConnection) => void;
    onRunnerReconnected: (conn: RelayConnection) => void;
    onAcpMessage: (podKey: string, msgType: number, payload: unknown) => void;
  },
): void {
  if (typeof data === "string") {
    console.warn("Received string message from Relay, expected binary");
    return;
  }

  const bytes = new Uint8Array(data);
  const { type, payload } = decodeMessage(bytes);

  switch (type) {
    case MsgType.Snapshot:
      callbacks.onSnapshot(conn, payload);
      break;
    case MsgType.Output:
      for (const callback of conn.subscribers.values()) {
        callback(payload);
      }
      break;
    case MsgType.Control:
      callbacks.onControl(conn, payload);
      break;
    case MsgType.RunnerDisconnected:
      callbacks.onRunnerDisconnected(conn);
      break;
    case MsgType.RunnerReconnected:
      callbacks.onRunnerReconnected(conn);
      break;
    case MsgType.AcpEvent:
    case MsgType.AcpSnapshot:
    case MsgType.AcpCommand: {
      const parsed = decodeJsonPayload(payload);
      if (parsed !== null) {
        callbacks.onAcpMessage(conn.podKey, type, parsed);
      }
      break;
    }
    case MsgType.Pong:
      break;
    default:
      console.warn(`Unknown message type from Relay: ${type}`);
  }
}

/**
 * Handle a snapshot message — replay terminal content to all subscribers.
 */
export function handleSnapshot(conn: RelayConnection, payload: Uint8Array): void {
  conn.snapshotReceived = true;
  if (conn.snapshotTimer) {
    clearTimeout(conn.snapshotTimer);
    conn.snapshotTimer = null;
  }
  try {
    const snapshot = JSON.parse(new TextDecoder().decode(payload));
    if (snapshot.cols > 0 && snapshot.rows > 0) {
      conn.podSize = { rows: snapshot.rows, cols: snapshot.cols };
    }
    if (snapshot.serialized_content) {
      const clearSeq = new TextEncoder().encode("\x1b[2J\x1b[H\x1b[3J");
      const content = new TextEncoder().encode(snapshot.serialized_content);
      for (const callback of conn.subscribers.values()) {
        callback(clearSeq);
        callback(content);
      }
    }
  } catch (e) {
    console.error("Failed to parse snapshot message:", e);
  }
}

/**
 * Handle a control message — e.g. pod_resized.
 */
export function handleControl(conn: RelayConnection, payload: Uint8Array): void {
  try {
    const msg = JSON.parse(new TextDecoder().decode(payload));
    if (msg.type === "pod_resized") {
      conn.podSize = { rows: msg.rows, cols: msg.cols };
    }
  } catch (e) {
    console.error("Failed to parse control message:", e);
  }
}

/**
 * Handle runner disconnected notification.
 */
export function handleRunnerDisconnected(conn: RelayConnection): void {
  console.warn(`Runner disconnected for pod ${conn.podKey}`);
  conn.runnerDisconnected = true;
  const msg = new TextEncoder().encode(
    "\r\n\x1b[33m⚠ Runner disconnected. Waiting for reconnection...\x1b[0m\r\n"
  );
  for (const callback of conn.subscribers.values()) {
    callback(msg);
  }
}

/**
 * Handle runner reconnected notification.
 */
export function handleRunnerReconnected(conn: RelayConnection): void {
  console.log(`Runner reconnected for pod ${conn.podKey}`);
  conn.runnerDisconnected = false;
  const msg = new TextEncoder().encode(
    "\r\n\x1b[32m✓ Runner reconnected.\x1b[0m\r\n"
  );
  for (const callback of conn.subscribers.values()) {
    callback(msg);
  }
}
