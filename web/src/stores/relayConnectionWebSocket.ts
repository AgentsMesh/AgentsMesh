import { MsgType, encodeMessage, encodeResize } from "./relayProtocol";
import type { RelayConnection, ConnectionHandle } from "./relayConnectionTypes";
import { dispatchRelayMessage, handleSnapshot, handleControl, handleRunnerDisconnected, handleRunnerReconnected } from "./relayConnectionHandlers";
import { scheduleSnapshotRetry, scheduleReconnect, isNonRetryableError } from "./relayConnectionRetry";

export interface PoolContext {
  connections: Map<string, RelayConnection>;
  notifyStatusChange: (podKey: string) => void;
  notifyAcpListeners: (podKey: string, msgType: number, payload: unknown) => void;
  createHandle: (podKey: string, subscriptionId: string) => ConnectionHandle;
  subscribe: (podKey: string, subscriptionId: string, onMessage: (data: Uint8Array | string) => void) => Promise<ConnectionHandle>;
  maxReconnectAttempts: number;
  baseReconnectDelay: number;
}

export async function createNewConnection(
  ctx: PoolContext,
  podKey: string,
  relayUrl: string,
  relayToken: string,
  subscriptionId: string,
  onMessage: (data: Uint8Array | string) => void,
): Promise<ConnectionHandle> {
  const url = `${relayUrl}/browser/relay?token=${encodeURIComponent(relayToken)}`;
  const ws = new WebSocket(url);
  ws.binaryType = "arraybuffer";

  const conn: RelayConnection = {
    ws, podKey,
    status: "connecting",
    lastActivity: Date.now(),
    subscribers: new Map([[subscriptionId, onMessage]]),
    reconnectAttempts: 0,
    reconnectTimer: null,
    disconnectTimer: null,
    snapshotTimer: null,
    snapshotReceived: false,
    relayUrl,
    relayToken,
    runnerDisconnected: false,
  };

  ctx.connections.set(podKey, conn);
  setupWebSocketHandlers(ctx, podKey, ws);
  ctx.notifyStatusChange(podKey);

  return ctx.createHandle(podKey, subscriptionId);
}

function setupWebSocketHandlers(ctx: PoolContext, podKey: string, ws: WebSocket): void {
  const getConn = (pk: string) => ctx.connections.get(pk);

  ws.onopen = () => {
    const c = ctx.connections.get(podKey);
    if (c) {
      c.status = "connected";
      c.lastActivity = Date.now();
      c.reconnectAttempts = 0;
      c.snapshotReceived = false;
      ctx.notifyStatusChange(podKey);
      if (c.pendingResize) {
        doSendResize(ctx, podKey, c.pendingResize.cols, c.pendingResize.rows);
        c.pendingResize = undefined;
      }
      scheduleSnapshotRetry(getConn, podKey);
    }
  };

  ws.onmessage = (event) => {
    const c = ctx.connections.get(podKey);
    if (!c) return;
    c.lastActivity = Date.now();
    dispatchRelayMessage(c, event.data, {
      onSnapshot: (conn, payload) => { handleSnapshot(conn, payload); },
      onControl: (conn, payload) => { handleControl(conn, payload); },
      onRunnerDisconnected: (conn) => {
        handleRunnerDisconnected(conn);
        ctx.notifyStatusChange(conn.podKey);
      },
      onRunnerReconnected: (conn) => {
        handleRunnerReconnected(conn);
        ctx.notifyStatusChange(conn.podKey);
      },
      onAcpMessage: (pk, msgType, payload) => {
        ctx.notifyAcpListeners(pk, msgType, payload);
      },
    });
  };

  ws.onerror = (error) => {
    console.error(`Relay WebSocket error for ${podKey}:`, error);
    const c = ctx.connections.get(podKey);
    if (c) {
      c.status = "error";
      ctx.notifyStatusChange(podKey);
      if (c.subscribers.size > 0 && !c.reconnectTimer) {
        scheduleReconnect(getConn, (pk) => reconnectConnection(ctx, pk), podKey, ctx.maxReconnectAttempts, ctx.baseReconnectDelay);
      }
    }
  };

  ws.onclose = () => {
    const c = ctx.connections.get(podKey);
    if (c) {
      c.status = "disconnected";
      ctx.notifyStatusChange(podKey);
      if (c.subscribers.size > 0) {
        scheduleReconnect(getConn, (pk) => reconnectConnection(ctx, pk), podKey, ctx.maxReconnectAttempts, ctx.baseReconnectDelay);
      }
    }
  };
}

export function doSendResize(ctx: PoolContext, podKey: string, cols: number, rows: number): void {
  const conn = ctx.connections.get(podKey);
  if (!conn) return;

  if (conn.ws.readyState === WebSocket.OPEN) {
    conn.ws.send(encodeMessage(MsgType.Resize, encodeResize(cols, rows)));
  } else if (conn.ws.readyState === WebSocket.CONNECTING) {
    conn.pendingResize = { rows, cols };
  }
}

export async function reconnectConnection(ctx: PoolContext, podKey: string): Promise<void> {
  const oldConn = ctx.connections.get(podKey);
  if (!oldConn || oldConn.subscribers.size === 0) return;

  console.warn(`[Relay] Reconnecting terminal for ${podKey}`);

  const subscribersCopy = new Map(oldConn.subscribers);
  const reconnectAttempts = oldConn.reconnectAttempts;

  if (oldConn.ws.readyState === WebSocket.OPEN || oldConn.ws.readyState === WebSocket.CONNECTING) {
    oldConn.ws.close();
  }
  ctx.connections.delete(podKey);

  const firstEntry = subscribersCopy.entries().next().value;
  if (!firstEntry) return;

  const [firstId, firstCallback] = firstEntry;
  try {
    await ctx.subscribe(podKey, firstId, firstCallback);

    const newConn = ctx.connections.get(podKey);
    if (newConn) {
      subscribersCopy.forEach((callback, id) => {
        if (id !== firstId) newConn.subscribers.set(id, callback);
      });
      newConn.reconnectAttempts = reconnectAttempts;
    }
  } catch (error) {
    if (isNonRetryableError(error)) {
      console.warn(`[Relay] Non-retryable error for ${podKey}, stopping reconnection:`, error);
      ctx.notifyStatusChange(podKey);
      return;
    }
    console.warn(`[Relay] Retryable error for ${podKey}, scheduling retry:`, error);
    const getConn = (pk: string) => ctx.connections.get(pk);
    const placeholderWs = { readyState: WebSocket.CLOSED, close() {} } as unknown as WebSocket;
    const stub: RelayConnection = {
      ws: placeholderWs, podKey,
      status: "error",
      lastActivity: Date.now(),
      subscribers: subscribersCopy,
      reconnectAttempts,
      reconnectTimer: null,
      disconnectTimer: null,
      snapshotTimer: null,
      snapshotReceived: false,
      relayUrl: oldConn.relayUrl,
      relayToken: oldConn.relayToken,
      runnerDisconnected: oldConn.runnerDisconnected,
    };
    ctx.connections.set(podKey, stub);
    ctx.notifyStatusChange(podKey);
    scheduleReconnect(getConn, (pk) => reconnectConnection(ctx, pk), podKey, ctx.maxReconnectAttempts, ctx.baseReconnectDelay);
  }
}
