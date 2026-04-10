import type { RelayConnection } from "./relayConnectionTypes";
import { MsgType, decodeMessage, decodeJsonPayload } from "./relayProtocol";

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

export function handleRunnerDisconnected(conn: RelayConnection): void {
  conn.runnerDisconnected = true;
}

export function handleRunnerReconnected(conn: RelayConnection): void {
  conn.runnerDisconnected = false;
}
