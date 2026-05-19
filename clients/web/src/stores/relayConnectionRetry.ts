
import { ApiError } from "@/lib/api/api-types";
import { MsgType, encodeMessage } from "./relayProtocol";
import type { RelayConnection } from "./relayConnectionTypes";

const NON_RETRYABLE_STATUSES = [400, 403, 404];

export function isNonRetryableError(error: unknown): boolean {
  return error instanceof ApiError && NON_RETRYABLE_STATUSES.includes(error.status);
}

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
    if (c.transport.isOpen) {
      c.transport.send(encodeMessage(MsgType.Resync, new Uint8Array(0)));
      scheduleSnapshotRetry(getConnection, podKey, attempt + 1);
    }
  }, 2000);
}

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

function computeReconnectDelay(attempt: number, baseDelay: number): number {
  const base = Math.min(baseDelay * Math.pow(2, attempt), 30000);
  const jitter = base * (Math.random() * 0.4 - 0.2);
  return Math.round(base + jitter);
}
