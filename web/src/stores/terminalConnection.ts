/**
 * Terminal WebSocket connection management
 * Handles connection pooling, reconnection, and message buffering
 */

/**
 * Terminal connection state
 */
export interface TerminalConnection {
  ws: WebSocket;
  podKey: string;
  buffer: Uint8Array[];
  status: "connecting" | "connected" | "disconnected" | "error";
  lastActivity: number;
  listeners: Set<(data: Uint8Array | string) => void>;
  reconnectAttempts: number;
  reconnectTimer: ReturnType<typeof setTimeout> | null;
  pendingResize?: { rows: number; cols: number };
  ptySize?: { rows: number; cols: number };
}

/**
 * Connection result with send and disconnect methods
 */
export interface ConnectionHandle {
  send: (data: string) => void;
  disconnect: () => void;
}

/**
 * Serialized terminal state for restoration
 */
export interface SerializedTerminalState {
  data: string;
  cols: number;
  rows: number;
  timestamp: number;
}

/**
 * Terminal connection pool for managing WebSocket connections
 */
class TerminalConnectionPool {
  private connections: Map<string, TerminalConnection> = new Map();
  private serializedStates: Map<string, SerializedTerminalState> = new Map();
  private maxBufferSize = 100;
  private maxReconnectAttempts = 5;
  private baseReconnectDelay = 1000;
  private resizeDebounceTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();
  private resizeDebounceMs = 150;
  private maxSerializedStateAge = 30 * 60 * 1000; // 30 minutes

  getConnection(podKey: string): TerminalConnection | undefined {
    return this.connections.get(podKey);
  }

  connect(podKey: string, onMessage: (data: Uint8Array | string) => void): ConnectionHandle {
    let conn = this.connections.get(podKey);

    if (conn) {
      conn.listeners.add(onMessage);
      // Note: Don't replay buffer here - backend will send scrollback data on new WebSocket connection
      // The buffer is only used for data continuity during brief disconnections within the same session
      return {
        send: (data: string) => this.send(podKey, data),
        disconnect: () => this.removeListener(podKey, onMessage),
      };
    }

    const ws = this.createWebSocket(podKey);
    conn = {
      ws,
      podKey,
      buffer: [],
      status: "connecting",
      lastActivity: Date.now(),
      listeners: new Set([onMessage]),
      reconnectAttempts: 0,
      reconnectTimer: null,
    };

    this.connections.set(podKey, conn);
    this.setupWebSocketHandlers(podKey, ws);

    return {
      send: (data: string) => this.send(podKey, data),
      disconnect: () => this.removeListener(podKey, onMessage),
    };
  }

  private createWebSocket(podKey: string): WebSocket {
    let wsUrl = process.env.NEXT_PUBLIC_WS_URL;
    if (!wsUrl) {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL;
      if (apiUrl) {
        wsUrl = apiUrl.replace(/^http/, "ws");
      } else if (typeof window !== "undefined") {
        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        const host = window.location.hostname;
        wsUrl = `${protocol}//${host}:8080`;
      } else {
        wsUrl = "ws://localhost:8080";
      }
    }

    let token = null;
    let orgSlug = null;
    try {
      const authData = localStorage.getItem("agentsmesh-auth");
      if (authData) {
        const parsed = JSON.parse(authData);
        token = parsed.state?.token;
        orgSlug = parsed.state?.currentOrg?.slug;
      }
    } catch (e) {
      console.error("Failed to parse auth data:", e);
    }

    const ws = new WebSocket(`${wsUrl}/api/v1/orgs/${orgSlug}/ws/terminal/${podKey}?token=${token}`);
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
        if (c.pendingResize) {
          this.doSendResize(podKey, c.pendingResize.rows, c.pendingResize.cols);
          c.pendingResize = undefined;
        }
      }
    };

    ws.onmessage = (event) => {
      const c = this.connections.get(podKey);
      if (!c) return;

      c.lastActivity = Date.now();
      let data: Uint8Array | string;
      if (event.data instanceof ArrayBuffer) {
        data = new Uint8Array(event.data);
      } else {
        data = event.data;
      }

      if (typeof data === "string") {
        try {
          const msg = JSON.parse(data);
          if (msg.type === "pty_resized" && typeof msg.cols === "number" && typeof msg.rows === "number") {
            c.ptySize = { rows: msg.rows, cols: msg.cols };
            return;
          }
        } catch {
          // Not JSON, continue as terminal output
        }
      }

      if (typeof data !== "string") {
        c.buffer.push(data);
        if (c.buffer.length > this.maxBufferSize) {
          c.buffer.shift();
        }
      }

      for (const listener of c.listeners) {
        listener(data);
      }
    };

    ws.onerror = (error) => {
      console.error(`Terminal WebSocket error for ${podKey}:`, error);
      const c = this.connections.get(podKey);
      if (c) {
        c.status = "error";
        if (c.listeners.size > 0 && !c.reconnectTimer) {
          this.scheduleReconnect(podKey);
        }
      }
    };

    ws.onclose = () => {
      const c = this.connections.get(podKey);
      if (c) {
        c.status = "disconnected";
        if (c.listeners.size > 0) {
          this.scheduleReconnect(podKey);
        }
      }
    };
  }

  send(podKey: string, data: string): void {
    const conn = this.connections.get(podKey);
    if (conn && conn.ws.readyState === WebSocket.OPEN) {
      conn.ws.send(JSON.stringify({ type: "input", data }));
      conn.lastActivity = Date.now();
    }
  }

  sendResize(podKey: string, rows: number, cols: number): void {
    if (rows <= 0 || cols <= 0) return;

    const existingTimer = this.resizeDebounceTimers.get(podKey);
    if (existingTimer) {
      clearTimeout(existingTimer);
    }

    const timer = setTimeout(() => {
      this.doSendResize(podKey, rows, cols);
      this.resizeDebounceTimers.delete(podKey);
    }, this.resizeDebounceMs);

    this.resizeDebounceTimers.set(podKey, timer);
  }

  private doSendResize(podKey: string, rows: number, cols: number): void {
    const conn = this.connections.get(podKey);
    if (!conn) return;

    if (conn.ws.readyState === WebSocket.OPEN) {
      conn.ws.send(JSON.stringify({ type: "resize", rows, cols }));
    } else if (conn.ws.readyState === WebSocket.CONNECTING) {
      conn.pendingResize = { rows, cols };
    }
  }

  forceResize(podKey: string, rows: number, cols: number): void {
    if (rows <= 0 || cols <= 0) return;

    const existingTimer = this.resizeDebounceTimers.get(podKey);
    if (existingTimer) {
      clearTimeout(existingTimer);
      this.resizeDebounceTimers.delete(podKey);
    }

    const conn = this.connections.get(podKey);
    if (conn && conn.ws.readyState === WebSocket.OPEN) {
      conn.ws.send(JSON.stringify({ type: "resize", rows, cols }));
    }
  }

  getPtySize(podKey: string): { rows: number; cols: number } | undefined {
    return this.connections.get(podKey)?.ptySize;
  }

  removeListener(podKey: string, listener: (data: Uint8Array | string) => void): void {
    const conn = this.connections.get(podKey);
    if (conn) {
      conn.listeners.delete(listener);
      if (conn.listeners.size === 0) {
        setTimeout(() => {
          const currentConn = this.connections.get(podKey);
          if (currentConn && currentConn.listeners.size === 0) {
            this.disconnect(podKey);
          }
        }, 100);
      }
    }
  }

  disconnect(podKey: string): void {
    const conn = this.connections.get(podKey);
    if (conn) {
      if (conn.reconnectTimer) {
        clearTimeout(conn.reconnectTimer);
        conn.reconnectTimer = null;
      }
      if (conn.ws.readyState === WebSocket.OPEN || conn.ws.readyState === WebSocket.CONNECTING) {
        conn.ws.close();
      }
      this.connections.delete(podKey);
    }
  }

  private scheduleReconnect(podKey: string): void {
    const conn = this.connections.get(podKey);
    if (!conn || conn.reconnectAttempts >= this.maxReconnectAttempts) {
      if (conn) console.log(`Max reconnect attempts reached for ${podKey}`);
      return;
    }

    const delay = Math.min(this.baseReconnectDelay * Math.pow(2, conn.reconnectAttempts), 30000);
    console.log(`Scheduling reconnect for ${podKey} in ${delay}ms`);

    conn.reconnectTimer = setTimeout(() => {
      conn.reconnectAttempts++;
      this.reconnect(podKey);
    }, delay);
  }

  private reconnect(podKey: string): void {
    const oldConn = this.connections.get(podKey);
    if (!oldConn || oldConn.listeners.size === 0) return;

    console.log(`Reconnecting terminal for ${podKey}...`);

    const listeners = new Set(oldConn.listeners);
    // Note: Don't preserve buffer on reconnect - backend will send full scrollback data
    const reconnectAttempts = oldConn.reconnectAttempts;

    if (oldConn.ws.readyState === WebSocket.OPEN || oldConn.ws.readyState === WebSocket.CONNECTING) {
      oldConn.ws.close();
    }
    this.connections.delete(podKey);

    const firstListener = listeners.values().next().value;
    if (firstListener) {
      this.connect(podKey, firstListener);

      const newConn = this.connections.get(podKey);
      if (newConn) {
        listeners.forEach((l) => {
          if (l !== firstListener) newConn.listeners.add(l);
        });
        newConn.reconnectAttempts = reconnectAttempts;
        // buffer starts fresh - backend sends scrollback on new connection
      }
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

  /**
   * Save serialized terminal state for later restoration
   */
  saveSerializedState(podKey: string, data: string, cols: number, rows: number): void {
    this.serializedStates.set(podKey, {
      data,
      cols,
      rows,
      timestamp: Date.now(),
    });
  }

  /**
   * Get serialized terminal state if available and not expired
   */
  getSerializedState(podKey: string): SerializedTerminalState | null {
    const state = this.serializedStates.get(podKey);
    if (!state) return null;

    // Check if state is expired
    if (Date.now() - state.timestamp > this.maxSerializedStateAge) {
      this.serializedStates.delete(podKey);
      return null;
    }

    return state;
  }

  /**
   * Clear serialized state after successful restoration
   */
  clearSerializedState(podKey: string): void {
    this.serializedStates.delete(podKey);
  }
}

// Singleton instance
export const terminalPool = new TerminalConnectionPool();
