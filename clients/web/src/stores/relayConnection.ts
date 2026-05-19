import { MsgType, encodeMessage, encodeJsonMessage } from "./relayProtocol";
import { getPodService } from "@/lib/wasm-core";
import { getLocalRunnerService } from "@agentsmesh/service-runtime";
import type { RelayConnection, ConnectionHandle, StatusListener } from "./relayConnectionTypes";
import { createNewConnection, doSendResize, type PoolContext } from "./relayConnectionWebSocket";
import { probeRelayOpen } from "./relayProbe";

export { MsgType, encodeMessage } from "./relayProtocol";
export type { ConnectionStatus, RelayConnection, ConnectionHandle, RelayStatusInfo } from "./relayConnectionTypes";

class RelayConnectionPool {
  private connections: Map<string, RelayConnection> = new Map();
  private pendingSubscriptions: Map<string, Promise<ConnectionHandle>> = new Map();
  private maxReconnectAttempts = 50;
  private baseReconnectDelay = 1000;
  private resizeDebounceTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();
  private resizeDebounceMs = 150;
  private disconnectDelay = 30000;
  private lastInputs: Map<string, { data: string; time: number }> = new Map();
  private deduplicateWindow = 50;
  private statusListeners: Map<string, Set<StatusListener>> = new Map();
  private acpListeners: Map<string, Set<(msgType: number, payload: unknown) => void>> = new Map();

  constructor() {
    if (typeof window !== "undefined") {
      window.addEventListener("beforeunload", () => this.disconnectAll());
    }
  }

  private get ctx(): PoolContext {
    return {
      connections: this.connections,
      notifyStatusChange: (pk) => this.notifyStatusChange(pk),
      notifyAcpListeners: (pk, mt, pl) => this.notifyAcpListeners(pk, mt, pl),
      createHandle: (pk, sid) => this.createHandle(pk, sid),
      subscribe: (pk, sid, cb) => this.subscribe(pk, sid, cb),
      maxReconnectAttempts: this.maxReconnectAttempts,
      baseReconnectDelay: this.baseReconnectDelay,
    };
  }

  getConnection(podKey: string): RelayConnection | undefined {
    return this.connections.get(podKey);
  }

  onStatusChange(podKey: string, listener: StatusListener): () => void {
    let listeners = this.statusListeners.get(podKey);
    if (!listeners) {
      listeners = new Set();
      this.statusListeners.set(podKey, listeners);
    }
    listeners.add(listener);
    const conn = this.connections.get(podKey);
    listener({ status: conn?.status ?? "none", runnerDisconnected: conn?.runnerDisconnected ?? false });
    return () => {
      listeners!.delete(listener);
      if (listeners!.size === 0) this.statusListeners.delete(podKey);
    };
  }

  private notifyStatusChange(podKey: string): void {
    const listeners = this.statusListeners.get(podKey);
    if (!listeners || listeners.size === 0) return;
    const conn = this.connections.get(podKey);
    const info = { status: conn?.status ?? ("none" as const), runnerDisconnected: conn?.runnerDisconnected ?? false };
    for (const listener of listeners) listener(info);
  }

  private isConnectionAlive(conn: RelayConnection): boolean {
    return conn.transport.isOpen && conn.status === "connected";
  }

  async subscribe(podKey: string, subscriptionId: string, onMessage: (data: Uint8Array | string) => void): Promise<ConnectionHandle> {
    const conn = this.connections.get(podKey);
    if (conn) {
      if (conn.transport.isClosed) {
        this.connections.delete(podKey);
        return this.subscribe(podKey, subscriptionId, onMessage);
      }
      if (conn.subscribers.has(subscriptionId)) {
        conn.subscribers.set(subscriptionId, onMessage);
        return this.createHandle(podKey, subscriptionId);
      }
      if (conn.disconnectTimer) { clearTimeout(conn.disconnectTimer); conn.disconnectTimer = null; }
      conn.subscribers.set(subscriptionId, onMessage);
      if (conn.transport.isOpen) {
        conn.transport.send(encodeMessage(MsgType.Resync, new Uint8Array(0)));
      }
      return this.createHandle(podKey, subscriptionId);
    }

    const pending = this.pendingSubscriptions.get(podKey);
    if (pending) { await pending; return this.subscribe(podKey, subscriptionId, onMessage); }

    const createPromise = (async () => {
      const relayInfo = JSON.parse(await getPodService().get_pod_connection(podKey));
      if (relayInfo.local_relay_url && relayInfo.local_token && await isSameHostRunner(relayInfo.local_relay_node_id)) {
        const ok = await probeRelayOpen(relayInfo.local_relay_url, relayInfo.local_token, 1000);
        if (ok) {
          return createNewConnection(this.ctx, podKey, relayInfo.local_relay_url, relayInfo.local_token, subscriptionId, onMessage);
        }
      }
      return createNewConnection(this.ctx, podKey, relayInfo.relay_url, relayInfo.token, subscriptionId, onMessage);
    })();
    this.pendingSubscriptions.set(podKey, createPromise);
    try { return await createPromise; } finally { this.pendingSubscriptions.delete(podKey); }
  }

  send(podKey: string, data: string): void {
    const conn = this.connections.get(podKey);
    if (!conn || !conn.transport.isOpen) return;
    const now = Date.now();
    if (data.length > 1) {
      const lastInput = this.lastInputs.get(podKey);
      if (lastInput && lastInput.data === data && (now - lastInput.time) < this.deduplicateWindow) return;
      this.lastInputs.set(podKey, { data, time: now });
    }
    conn.transport.send(encodeMessage(MsgType.Input, data));
    conn.lastActivity = now;
  }

  sendResize(podKey: string, cols: number, rows: number): void {
    if (rows <= 0 || cols <= 0) return;
    const existingTimer = this.resizeDebounceTimers.get(podKey);
    if (existingTimer) clearTimeout(existingTimer);
    const timer = setTimeout(() => {
      doSendResize(this.ctx, podKey, cols, rows);
      this.resizeDebounceTimers.delete(podKey);
    }, this.resizeDebounceMs);
    this.resizeDebounceTimers.set(podKey, timer);
  }

  forceResize(podKey: string, cols: number, rows: number): void {
    if (rows <= 0 || cols <= 0) return;
    const existingTimer = this.resizeDebounceTimers.get(podKey);
    if (existingTimer) { clearTimeout(existingTimer); this.resizeDebounceTimers.delete(podKey); }
    doSendResize(this.ctx, podKey, cols, rows);
  }

  getPodSize(podKey: string): { rows: number; cols: number } | undefined {
    return this.connections.get(podKey)?.podSize;
  }

  unsubscribe(podKey: string, subscriptionId: string): void {
    const conn = this.connections.get(podKey);
    if (!conn) return;
    conn.subscribers.delete(subscriptionId);
    if (conn.subscribers.size === 0 && !conn.disconnectTimer) {
      conn.disconnectTimer = setTimeout(() => {
        const currentConn = this.connections.get(podKey);
        if (currentConn && currentConn.subscribers.size === 0) this.disconnect(podKey);
      }, this.disconnectDelay);
    }
  }

  disconnect(podKey: string): void {
    const conn = this.connections.get(podKey);
    if (!conn) return;
    if (conn.reconnectTimer) { clearTimeout(conn.reconnectTimer); conn.reconnectTimer = null; }
    if (conn.disconnectTimer) { clearTimeout(conn.disconnectTimer); conn.disconnectTimer = null; }
    this.connections.delete(podKey);
    this.lastInputs.delete(podKey);
    this.acpListeners.delete(podKey);
    this.notifyStatusChange(podKey);
    conn.transport.close();
  }

  disconnectAll(): void { for (const [podKey] of this.connections) this.disconnect(podKey); }
  getStatus(podKey: string): RelayConnection["status"] | "none" { return this.connections.get(podKey)?.status || "none"; }
  isConnected(podKey: string): boolean { const c = this.connections.get(podKey); return c?.status === "connected" && c.transport.isOpen; }
  isRunnerDisconnected(podKey: string): boolean { return this.connections.get(podKey)?.runnerDisconnected ?? false; }

  sendAcpCommand(podKey: string, command: Record<string, unknown>): void {
    const conn = this.connections.get(podKey);
    if (!conn || !conn.transport.isOpen) return;
    conn.transport.send(encodeJsonMessage(MsgType.AcpCommand, command));
    conn.lastActivity = Date.now();
  }

  onAcpMessage(podKey: string, listener: (msgType: number, payload: unknown) => void): () => void {
    let set = this.acpListeners.get(podKey);
    if (!set) { set = new Set(); this.acpListeners.set(podKey, set); }
    set.add(listener);
    return () => { set!.delete(listener); if (set!.size === 0) this.acpListeners.delete(podKey); };
  }

  private createHandle(podKey: string, subscriptionId: string): ConnectionHandle {
    return { send: (data) => this.send(podKey, data), unsubscribe: () => this.unsubscribe(podKey, subscriptionId) };
  }

  private notifyAcpListeners(podKey: string, msgType: number, payload: unknown): void {
    const listeners = this.acpListeners.get(podKey);
    if (!listeners) return;
    for (const listener of listeners) listener(msgType, payload);
  }
}

function getOrCreatePool(): RelayConnectionPool {
  const key = "__relayPool" as keyof typeof globalThis;
  const existing = globalThis[key] as RelayConnectionPool | undefined;
  if (existing) {
    if (process.env.NODE_ENV === "development") {
      existing.disconnectAll();
    } else {
      return existing;
    }
  }
  const pool = new RelayConnectionPool();
  (globalThis as Record<string, unknown>)[key] = pool;
  return pool;
}

// Cache only resolved non-empty IDs — pre-onboarding null must not pin renderer to "different host".
let cachedNodeIdPromise: Promise<string | null> | null = null;

async function resolveLocalNodeId(): Promise<string | null> {
  const svc = getLocalRunnerService();
  if (!svc) return null;
  if (!cachedNodeIdPromise) {
    cachedNodeIdPromise = svc.local_node_id().then(
      (id) => {
        if (!id) cachedNodeIdPromise = null;
        return id;
      },
      () => {
        cachedNodeIdPromise = null;
        return null;
      },
    );
  }
  return cachedNodeIdPromise;
}

async function isSameHostRunner(advertisedNodeID: string | undefined): Promise<boolean> {
  if (!advertisedNodeID) return true;
  if (!getLocalRunnerService()) return false;
  const myNodeID = await resolveLocalNodeId();
  return myNodeID !== null && myNodeID === advertisedNodeID;
}

export const relayPool = getOrCreatePool();
