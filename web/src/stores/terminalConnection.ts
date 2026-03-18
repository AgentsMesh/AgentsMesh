/**
 * Terminal WebSocket connection management
 * Handles connection pooling, reconnection, and message buffering
 *
 * Architecture:
 * - Browser connects to Relay (not Backend) for terminal data
 * - Control flow: Browser -> Backend (REST) -> Runner (gRPC)
 * - Data flow: Browser <-> Relay <-> Runner (WebSocket)
 */

import { podApi } from "@/lib/api/pod";
import { MsgType, encodeMessage, decodeMessage, encodeResize } from "./relayProtocol";

// Re-export protocol symbols for consumers that import from this module
export { MsgType, encodeMessage } from "./relayProtocol";

/**
 * Connection status of a terminal WebSocket.
 * Single source of truth — import this type instead of redefining inline.
 */
export type ConnectionStatus = "connecting" | "connected" | "disconnected" | "error";

/**
 * Terminal connection state
 */
export interface TerminalConnection {
  ws: WebSocket;
  podKey: string;
  status: ConnectionStatus;
  lastActivity: number;
  /** Subscribers map: subscriptionId -> callback */
  subscribers: Map<string, (data: Uint8Array | string) => void>;
  reconnectAttempts: number;
  reconnectTimer: ReturnType<typeof setTimeout> | null;
  /** Timer for delayed disconnect when all subscribers leave */
  disconnectTimer: ReturnType<typeof setTimeout> | null;
  pendingResize?: { rows: number; cols: number };
  ptySize?: { rows: number; cols: number };
  relayUrl: string;
  relayToken: string;
  runnerDisconnected: boolean;
}

/**
 * Connection result with send and unsubscribe methods
 */
export interface ConnectionHandle {
  send: (data: string) => void;
  /** Unsubscribe from terminal output. Connection stays open if other subscribers exist. */
  unsubscribe: () => void;
  /** @deprecated Use unsubscribe() instead. Kept for backward compatibility. */
  disconnect: () => void;
}

export type RelayStatusInfo = {
  status: TerminalConnection["status"] | "none";
  runnerDisconnected: boolean;
};

type StatusListener = (info: RelayStatusInfo) => void;

/**
 * Terminal connection pool for managing WebSocket connections to Relay.
 *
 * - Connections are keyed by podKey and shared across multiple subscribers
 * - Each subscriber has a unique subscriptionId for idempotent add/remove
 * - Connection stays open as long as at least one subscriber exists
 * - Uses delayed disconnect (30s) when last subscriber leaves
 */
class TerminalConnectionPool {
  private connections: Map<string, TerminalConnection> = new Map();
  private maxReconnectAttempts = 50;
  private baseReconnectDelay = 1000;
  private resizeDebounceTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();
  private resizeDebounceMs = 150;
  private disconnectDelay = 30000;
  private lastInputs: Map<string, { data: string; time: number }> = new Map();
  private deduplicateWindow = 50;
  private statusListeners: Map<string, Set<StatusListener>> = new Map();

  getConnection(podKey: string): TerminalConnection | undefined {
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
    listener({
      status: conn?.status ?? "none",
      runnerDisconnected: conn?.runnerDisconnected ?? false,
    });

    return () => {
      listeners!.delete(listener);
      if (listeners!.size === 0) {
        this.statusListeners.delete(podKey);
      }
    };
  }

  private notifyStatusChange(podKey: string): void {
    const listeners = this.statusListeners.get(podKey);
    if (!listeners || listeners.size === 0) return;
    const conn = this.connections.get(podKey);
    const info: RelayStatusInfo = {
      status: conn?.status ?? "none",
      runnerDisconnected: conn?.runnerDisconnected ?? false,
    };
    for (const listener of listeners) {
      listener(info);
    }
  }

  async subscribe(
    podKey: string,
    subscriptionId: string,
    onMessage: (data: Uint8Array | string) => void
  ): Promise<ConnectionHandle> {
    let conn = this.connections.get(podKey);

    if (conn) {
      const hadPrevious = conn.subscribers.has(subscriptionId);

      if (hadPrevious) {
        conn.subscribers.set(subscriptionId, onMessage);
        return this.createHandle(podKey, subscriptionId);
      }

      if (conn.disconnectTimer) {
        clearTimeout(conn.disconnectTimer);
        conn.disconnectTimer = null;
      }
      conn.subscribers.set(subscriptionId, onMessage);

      if (conn.ws.readyState === WebSocket.OPEN) {
        conn.ws.send(encodeMessage(MsgType.Resync, new Uint8Array(0)));
      }

      return this.createHandle(podKey, subscriptionId);
    }

    const relayInfo = await podApi.getTerminalConnection(podKey);
    const ws = this.createRelayWebSocket(relayInfo);

    conn = {
      ws,
      podKey,
      status: "connecting",
      lastActivity: Date.now(),
      subscribers: new Map([[subscriptionId, onMessage]]),
      reconnectAttempts: 0,
      reconnectTimer: null,
      disconnectTimer: null,
      relayUrl: relayInfo.relay_url,
      relayToken: relayInfo.token,
      runnerDisconnected: false,
    };

    this.connections.set(podKey, conn);
    this.setupWebSocketHandlers(podKey, ws);
    this.notifyStatusChange(podKey);

    return this.createHandle(podKey, subscriptionId);
  }

  /**
   * @deprecated Use subscribe() instead.
   */
  async connect(podKey: string, onMessage: (data: Uint8Array | string) => void): Promise<ConnectionHandle> {
    const subscriptionId = `legacy-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
    console.warn(`[Relay] connect() is deprecated, use subscribe() with stable subscriptionId`);
    return this.subscribe(podKey, subscriptionId, onMessage);
  }

  send(podKey: string, data: string): void {
    const conn = this.connections.get(podKey);
    if (!conn || conn.ws.readyState !== WebSocket.OPEN) return;

    const now = Date.now();
    if (data.length > 1) {
      const lastInput = this.lastInputs.get(podKey);
      if (lastInput && lastInput.data === data && (now - lastInput.time) < this.deduplicateWindow) {
        return;
      }
      this.lastInputs.set(podKey, { data, time: now });
    }

    conn.ws.send(encodeMessage(MsgType.Input, data));
    conn.lastActivity = now;
  }

  sendResize(podKey: string, cols: number, rows: number): void {
    if (rows <= 0 || cols <= 0) return;

    const existingTimer = this.resizeDebounceTimers.get(podKey);
    if (existingTimer) clearTimeout(existingTimer);

    const timer = setTimeout(() => {
      this.doSendResize(podKey, cols, rows);
      this.resizeDebounceTimers.delete(podKey);
    }, this.resizeDebounceMs);

    this.resizeDebounceTimers.set(podKey, timer);
  }

  forceResize(podKey: string, cols: number, rows: number): void {
    if (rows <= 0 || cols <= 0) return;

    const existingTimer = this.resizeDebounceTimers.get(podKey);
    if (existingTimer) {
      clearTimeout(existingTimer);
      this.resizeDebounceTimers.delete(podKey);
    }

    this.doSendResize(podKey, cols, rows);
  }

  getPtySize(podKey: string): { rows: number; cols: number } | undefined {
    return this.connections.get(podKey)?.ptySize;
  }

  unsubscribe(podKey: string, subscriptionId: string): void {
    const conn = this.connections.get(podKey);
    if (!conn) return;

    conn.subscribers.delete(subscriptionId);

    if (conn.subscribers.size === 0 && !conn.disconnectTimer) {
      conn.disconnectTimer = setTimeout(() => {
        const currentConn = this.connections.get(podKey);
        if (currentConn && currentConn.subscribers.size === 0) {
          this.disconnect(podKey);
        }
      }, this.disconnectDelay);
    }
  }

  /**
   * @deprecated Use unsubscribe() instead.
   */
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  removeListener(_podKey: string, _listener: (data: Uint8Array | string) => void): void {
    console.warn(`[Relay] removeListener() is deprecated, use subscribe()/unsubscribe() with stable subscriptionId`);
  }

  disconnect(podKey: string): void {
    const conn = this.connections.get(podKey);
    if (!conn) return;

    if (conn.reconnectTimer) {
      clearTimeout(conn.reconnectTimer);
      conn.reconnectTimer = null;
    }
    if (conn.disconnectTimer) {
      clearTimeout(conn.disconnectTimer);
      conn.disconnectTimer = null;
    }
    this.connections.delete(podKey);
    this.lastInputs.delete(podKey);
    this.notifyStatusChange(podKey);
    conn.ws.onopen = null;
    conn.ws.onmessage = null;
    conn.ws.onerror = null;
    conn.ws.onclose = null;
    if (conn.ws.readyState === WebSocket.OPEN || conn.ws.readyState === WebSocket.CONNECTING) {
      conn.ws.close();
    }
  }

  disconnectAll(): void {
    for (const [podKey] of this.connections) {
      this.disconnect(podKey);
    }
  }

  getStatus(podKey: string): TerminalConnection["status"] | "none" {
    return this.connections.get(podKey)?.status || "none";
  }

  isConnected(podKey: string): boolean {
    const conn = this.connections.get(podKey);
    return conn?.status === "connected" && conn.ws.readyState === WebSocket.OPEN;
  }

  isRunnerDisconnected(podKey: string): boolean {
    return this.connections.get(podKey)?.runnerDisconnected ?? false;
  }

  // --- Private helpers ---

  private createHandle(podKey: string, subscriptionId: string): ConnectionHandle {
    return {
      send: (data: string) => this.send(podKey, data),
      unsubscribe: () => this.unsubscribe(podKey, subscriptionId),
      disconnect: () => this.unsubscribe(podKey, subscriptionId),
    };
  }

  private createRelayWebSocket(relayInfo: { relay_url: string; token: string }): WebSocket {
    const url = `${relayInfo.relay_url}/browser/terminal?token=${encodeURIComponent(relayInfo.token)}`;
    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    return ws;
  }

  private setupWebSocketHandlers(podKey: string, ws: WebSocket): void {
    ws.onopen = () => {
      const c = this.connections.get(podKey);
      if (c) {
        c.status = "connected";
        c.lastActivity = Date.now();
        c.reconnectAttempts = 0;
        this.notifyStatusChange(podKey);
        if (c.pendingResize) {
          this.doSendResize(podKey, c.pendingResize.cols, c.pendingResize.rows);
          c.pendingResize = undefined;
        }
      }
    };

    ws.onmessage = (event) => {
      const c = this.connections.get(podKey);
      if (!c) return;
      c.lastActivity = Date.now();
      this.handleRelayMessage(c, event.data);
    };

    ws.onerror = (error) => {
      console.error(`Terminal WebSocket error for ${podKey}:`, error);
      const c = this.connections.get(podKey);
      if (c) {
        c.status = "error";
        this.notifyStatusChange(podKey);
        if (c.subscribers.size > 0 && !c.reconnectTimer) {
          this.scheduleReconnect(podKey);
        }
      }
    };

    ws.onclose = () => {
      const c = this.connections.get(podKey);
      if (c) {
        c.status = "disconnected";
        this.notifyStatusChange(podKey);
        if (c.subscribers.size > 0) {
          this.scheduleReconnect(podKey);
        }
      }
    };
  }

  private handleRelayMessage(conn: TerminalConnection, data: ArrayBuffer | string): void {
    if (typeof data === "string") {
      console.warn("Received string message from Relay, expected binary");
      return;
    }

    const bytes = new Uint8Array(data);
    const { type, payload } = decodeMessage(bytes);

    switch (type) {
      case MsgType.Snapshot:
        this.handleSnapshot(conn, payload);
        break;
      case MsgType.Output:
        for (const callback of conn.subscribers.values()) {
          callback(payload);
        }
        break;
      case MsgType.Control:
        this.handleControl(conn, payload);
        break;
      case MsgType.RunnerDisconnected:
        this.handleRunnerDisconnected(conn);
        break;
      case MsgType.RunnerReconnected:
        this.handleRunnerReconnected(conn);
        break;
      case MsgType.Pong:
        break;
      default:
        console.warn(`Unknown message type from Relay: ${type}`);
    }
  }

  private handleSnapshot(conn: TerminalConnection, payload: Uint8Array): void {
    try {
      const snapshot = JSON.parse(new TextDecoder().decode(payload));
      if (snapshot.cols > 0 && snapshot.rows > 0) {
        conn.ptySize = { rows: snapshot.rows, cols: snapshot.cols };
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

  private handleControl(conn: TerminalConnection, payload: Uint8Array): void {
    try {
      const msg = JSON.parse(new TextDecoder().decode(payload));
      if (msg.type === "pty_resized") {
        conn.ptySize = { rows: msg.rows, cols: msg.cols };
      }
    } catch (e) {
      console.error("Failed to parse control message:", e);
    }
  }

  private handleRunnerDisconnected(conn: TerminalConnection): void {
    console.warn(`Runner disconnected for pod ${conn.podKey}`);
    conn.runnerDisconnected = true;
    this.notifyStatusChange(conn.podKey);
    const msg = new TextEncoder().encode(
      "\r\n\x1b[33m⚠ Runner disconnected. Waiting for reconnection...\x1b[0m\r\n"
    );
    for (const callback of conn.subscribers.values()) {
      callback(msg);
    }
  }

  private handleRunnerReconnected(conn: TerminalConnection): void {
    console.log(`Runner reconnected for pod ${conn.podKey}`);
    conn.runnerDisconnected = false;
    this.notifyStatusChange(conn.podKey);
    const msg = new TextEncoder().encode(
      "\r\n\x1b[32m✓ Runner reconnected.\x1b[0m\r\n"
    );
    for (const callback of conn.subscribers.values()) {
      callback(msg);
    }
  }

  private doSendResize(podKey: string, cols: number, rows: number): void {
    const conn = this.connections.get(podKey);
    if (!conn) return;

    if (conn.ws.readyState === WebSocket.OPEN) {
      conn.ws.send(encodeMessage(MsgType.Resize, encodeResize(cols, rows)));
    } else if (conn.ws.readyState === WebSocket.CONNECTING) {
      conn.pendingResize = { rows, cols };
    }
  }

  private scheduleReconnect(podKey: string): void {
    const conn = this.connections.get(podKey);
    if (!conn || conn.reconnectAttempts >= this.maxReconnectAttempts) return;

    const baseDelay = Math.min(this.baseReconnectDelay * Math.pow(2, conn.reconnectAttempts), 30000);
    const jitter = baseDelay * (Math.random() * 0.4 - 0.2);
    const delay = Math.round(baseDelay + jitter);

    conn.reconnectTimer = setTimeout(() => {
      conn.reconnectAttempts++;
      this.reconnect(podKey);
    }, delay);
  }

  private async reconnect(podKey: string): Promise<void> {
    const oldConn = this.connections.get(podKey);
    if (!oldConn || oldConn.subscribers.size === 0) return;

    console.warn(`[Relay] Reconnecting terminal for ${podKey}`);

    const subscribersCopy = new Map(oldConn.subscribers);
    const reconnectAttempts = oldConn.reconnectAttempts;

    if (oldConn.ws.readyState === WebSocket.OPEN || oldConn.ws.readyState === WebSocket.CONNECTING) {
      oldConn.ws.close();
    }
    this.connections.delete(podKey);

    const firstEntry = subscribersCopy.entries().next().value;
    if (firstEntry) {
      const [firstId, firstCallback] = firstEntry;
      await this.subscribe(podKey, firstId, firstCallback);

      const newConn = this.connections.get(podKey);
      if (newConn) {
        subscribersCopy.forEach((callback, id) => {
          if (id !== firstId) {
            newConn.subscribers.set(id, callback);
          }
        });
        newConn.reconnectAttempts = reconnectAttempts;
      }
    }
  }
}

// Singleton instance
export const terminalPool = new TerminalConnectionPool();
