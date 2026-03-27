/**
 * Retry and reconnect logic for relay WebSocket connections.
 * Extracted from RelayConnectionPool for SRP.
 */

import { MsgType, encodeMessage } from "./relayProtocol";
import type { RelayConnection } from "./relayConnectionTypes";

/**
 * Schedule a resync retry if no snapshot is received within 2 seconds.
 * Retries up to 3 times.
 */
export function scheduleSnapshotRetry(
  getConnection: (podKey: string) => RelayConnection | undefined,
  podKey: string,
  attempt = 0,
): void {
  const conn = getConnection(podKey);
  if (!conn || attempt >= 3) return;

  conn.snapshotTimer = setTimeout(() => {
    const c = getConnection(podKey);
    if (!c || c.snapshotReceived) return;
    if (c.ws.readyState === WebSocket.OPEN) {
      c.ws.send(encodeMessage(MsgType.Resync, new Uint8Array(0)));
      scheduleSnapshotRetry(getConnection, podKey, attempt + 1);
    }
  }, 2000);
}

/**
 * Schedule a reconnect with exponential backoff + jitter.
 */
export function scheduleReconnect(
  getConnection: (podKey: string) => RelayConnection | undefined,
  reconnectFn: (podKey: string) => void,
  podKey: string,
  maxAttempts: number,
  baseDelay: number,
): void {
  const conn = getConnection(podKey);
  if (!conn || conn.reconnectAttempts >= maxAttempts) return;

  const delay = computeReconnectDelay(conn.reconnectAttempts, baseDelay);

  conn.reconnectTimer = setTimeout(() => {
    conn.reconnectAttempts++;
    reconnectFn(podKey);
  }, delay);
}

/**
 * Compute reconnect delay with exponential backoff capped at 30s, plus jitter.
 */
function computeReconnectDelay(attempt: number, baseDelay: number): number {
  const base = Math.min(baseDelay * Math.pow(2, attempt), 30000);
  const jitter = base * (Math.random() * 0.4 - 0.2);
  return Math.round(base + jitter);
}
