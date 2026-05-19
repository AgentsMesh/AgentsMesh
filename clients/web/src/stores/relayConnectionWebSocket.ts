import { MsgType, encodeMessage, encodeResize } from "./relayProtocol";
import type { RelayConnection, ConnectionHandle } from "./relayConnectionTypes";
import { getRelayBackend, type IRelayTransport } from "./relayBackend";
import { dispatchRelayMessage, handleSnapshot, handleControl, handleRunnerDisconnected, handleRunnerReconnected } from "./relayConnectionHandlers";
import { scheduleReconnect, scheduleSnapshotRetry, isNonRetryableError } from "./relayConnectionRetry";

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

  const conn: RelayConnection = {
    transport: null!,
    podKey,
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

  const transport = getRelayBackend().connect(url, {
    onOpen: () => handleOpen(ctx, podKey),
    onMessage: (data) => handleMessage(ctx, podKey, data),
    onClose: () => handleClose(ctx, podKey),
    onError: (error) => handleError(ctx, podKey, error),
  });
  conn.transport = transport;

  ctx.connections.set(podKey, conn);
  ctx.notifyStatusChange(podKey);
  return ctx.createHandle(podKey, subscriptionId);
}

function handleOpen(ctx: PoolContext, podKey: string): void {
  const c = ctx.connections.get(podKey);
  if (!c) return;
  c.status = "connected";
  c.lastActivity = Date.now();
  c.reconnectAttempts = 0;
  c.snapshotReceived = false;
  ctx.notifyStatusChange(podKey);
  if (c.pendingResize) {
    doSendResize(ctx, podKey, c.pendingResize.cols, c.pendingResize.rows);
    c.pendingResize = undefined;
  }
  if (c.transport.isOpen) {
    c.transport.send(encodeMessage(MsgType.Resync, new Uint8Array(0)));
  }
  scheduleSnapshotRetry((pk) => ctx.connections.get(pk), podKey);
}

function handleMessage(ctx: PoolContext, podKey: string, data: ArrayBuffer): void {
  const c = ctx.connections.get(podKey);
  if (!c) return;
  c.lastActivity = Date.now();
  dispatchRelayMessage(c, data, {
    onSnapshot: (conn, payload) => {
      conn.snapshotReceived = true;
      if (conn.snapshotTimer) { clearTimeout(conn.snapshotTimer); conn.snapshotTimer = null; }
      handleSnapshot(conn, payload);
    },
    onControl: (conn, payload) => handleControl(conn, payload),
    onRunnerDisconnected: (conn) => { handleRunnerDisconnected(conn); ctx.notifyStatusChange(conn.podKey); },
    onRunnerReconnected: (conn) => { handleRunnerReconnected(conn); ctx.notifyStatusChange(conn.podKey); },
    onAcpMessage: (pk, msgType, payload) => ctx.notifyAcpListeners(pk, msgType, payload),
  });
}

function handleError(ctx: PoolContext, podKey: string, error: unknown): void {
  console.error(`Relay WebSocket error for ${podKey}:`, error);
  const c = ctx.connections.get(podKey);
  if (!c) return;
  c.status = "error";
  ctx.notifyStatusChange(podKey);
  if (c.subscribers.size > 0 && !c.reconnectTimer) {
    scheduleReconnect(
      (pk) => ctx.connections.get(pk),
      (pk) => reconnectConnection(ctx, pk),
      podKey, ctx.maxReconnectAttempts, ctx.baseReconnectDelay,
    );
  }
}

function handleClose(ctx: PoolContext, podKey: string): void {
  const c = ctx.connections.get(podKey);
  if (!c) return;
  c.status = "disconnected";
  ctx.notifyStatusChange(podKey);
  if (c.subscribers.size > 0) {
    scheduleReconnect(
      (pk) => ctx.connections.get(pk),
      (pk) => reconnectConnection(ctx, pk),
      podKey, ctx.maxReconnectAttempts, ctx.baseReconnectDelay,
    );
  }
}

export function doSendResize(ctx: PoolContext, podKey: string, cols: number, rows: number): void {
  const conn = ctx.connections.get(podKey);
  if (!conn) return;
  if (conn.transport.isOpen) {
    conn.transport.send(encodeMessage(MsgType.Resize, encodeResize(cols, rows)));
  } else if (!conn.transport.isClosed) {
    conn.pendingResize = { rows, cols };
  }
}

export async function reconnectConnection(ctx: PoolContext, podKey: string): Promise<void> {
  const oldConn = ctx.connections.get(podKey);
  if (!oldConn || oldConn.subscribers.size === 0) return;

  console.warn(`[Relay] Reconnecting terminal for ${podKey}`);
  const subscribersCopy = new Map(oldConn.subscribers);
  const reconnectAttempts = oldConn.reconnectAttempts;

  oldConn.transport.close();
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
    const stubTransport: IRelayTransport = {
      get isOpen() { return false; },
      get isClosed() { return true; },
      send() {},
      close() {},
    };
    const stub: RelayConnection = {
      transport: stubTransport,
      podKey,
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
    scheduleReconnect(
      (pk) => ctx.connections.get(pk),
      (pk) => reconnectConnection(ctx, pk),
      podKey, ctx.maxReconnectAttempts, ctx.baseReconnectDelay,
    );
  }
}
