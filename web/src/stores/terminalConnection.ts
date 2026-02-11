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

/**
 * Relay message types (binary protocol)
 * Must match relay/internal/protocol/message.go
 */
const MsgType = {
  Snapshot: 0x01,           // Complete terminal snapshot
  Output: 0x02,             // Terminal output (raw PTY data)
  Input: 0x03,              // User input to terminal
  Resize: 0x04,             // Terminal resize
  Ping: 0x05,               // Ping for keepalive
  Pong: 0x06,               // Pong response
  Control: 0x07,            // Control messages (JSON)
  RunnerDisconnected: 0x08, // Runner disconnected notification
  RunnerReconnected: 0x09,  // Runner reconnected notification
} as const;

/**
 * Terminal connection state
 */
export interface TerminalConnection {
  ws: WebSocket;
  podKey: string;
  status: "connecting" | "connected" | "disconnected" | "error";
  lastActivity: number;
  /** Subscribers map: subscriptionId -> callback */
  subscribers: Map<string, (data: Uint8Array | string) => void>;
  reconnectAttempts: number;
  reconnectTimer: ReturnType<typeof setTimeout> | null;
  /** Timer for delayed disconnect when all subscribers leave */
  disconnectTimer: ReturnType<typeof setTimeout> | null;
  pendingResize?: { rows: number; cols: number };
  ptySize?: { rows: number; cols: number };
  // Relay connection info
  relayUrl: string;
  relayToken: string;
  // Runner status
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

/**
 * Encode a message with type prefix (Relay binary protocol)
 */
function encodeMessage(msgType: number, payload: Uint8Array | string): Uint8Array {
  const payloadBytes = typeof payload === "string"
    ? new TextEncoder().encode(payload)
    : payload;
  const message = new Uint8Array(1 + payloadBytes.length);
  message[0] = msgType;
  message.set(payloadBytes, 1);
  return message;
}

/**
 * Decode a message with type prefix (Relay binary protocol)
 */
function decodeMessage(data: Uint8Array): { type: number; payload: Uint8Array } {
  if (data.length < 1) {
    return { type: 0, payload: new Uint8Array(0) };
  }
  return {
    type: data[0],
    payload: data.slice(1),
  };
}

/**
 * Terminal connection pool for managing WebSocket connections to Relay
 *
 * Architecture:
 * - Connections are keyed by podKey and shared across multiple subscribers
 * - Each subscriber has a unique subscriptionId for idempotent add/remove
 * - Connection stays open as long as at least one subscriber exists
 * - Uses delayed disconnect (30s) when last subscriber leaves to handle rapid open/close
 */
export type RelayStatusInfo = {
  status: TerminalConnection["status"] | "none";
  runnerDisconnected: boolean;
};

type StatusListener = (info: RelayStatusInfo) => void;

class TerminalConnectionPool {
  private connections: Map<string, TerminalConnection> = new Map();
  private maxReconnectAttempts = 5;
  private baseReconnectDelay = 1000;
  private resizeDebounceTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();
  private resizeDebounceMs = 150;
  /** Delay before disconnecting when all subscribers leave (ms) */
  private disconnectDelay = 30000;

  // Input deduplication to prevent duplicate sends on mobile with network latency
  // Tracks last input per pod to filter rapid duplicate sends
  private lastInputs: Map<string, { data: string; time: number }> = new Map();
  private deduplicateWindow = 50; // 50ms window for deduplication

  /** Status change listeners per podKey */
  private statusListeners: Map<string, Set<StatusListener>> = new Map();

  getConnection(podKey: string): TerminalConnection | undefined {
    return this.connections.get(podKey);
  }

  /**
   * Subscribe to status changes for a pod.
   * Listener is called immediately with current status and on every change.
   * @returns Unsubscribe function
   */
  onStatusChange(podKey: string, listener: StatusListener): () => void {
    let listeners = this.statusListeners.get(podKey);
    if (!listeners) {
      listeners = new Set();
      this.statusListeners.set(podKey, listeners);
    }
    listeners.add(listener);

    // Immediately call with current status
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

  /**
   * Notify all status listeners for a pod of the current status.
   */
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

  /**
   * Subscribe to terminal output for a pod.
   *
   * @param podKey - The pod identifier
   * @param subscriptionId - Stable identifier for this subscriber (e.g., `terminal-${podKey}`)
   *                         Same subscriptionId will replace previous subscription (idempotent)
   * @param onMessage - Callback for terminal output
   * @returns ConnectionHandle with send and unsubscribe methods
   */
  async subscribe(
    podKey: string,
    subscriptionId: string,
    onMessage: (data: Uint8Array | string) => void
  ): Promise<ConnectionHandle> {
    let conn = this.connections.get(podKey);

    if (conn) {
      // Check if this is a new subscriber (not just updating an existing one)
      const hadPrevious = conn.subscribers.has(subscriptionId);

      if (hadPrevious) {
        // Same subscriptionId - just update the callback (idempotent)
        conn.subscribers.set(subscriptionId, onMessage);
        console.log(`[Relay] Updated subscriber ${subscriptionId} for ${podKey}, total: ${conn.subscribers.size}`);

        return {
          send: (data: string) => this.send(podKey, data),
          unsubscribe: () => this.unsubscribe(podKey, subscriptionId),
          disconnect: () => this.unsubscribe(podKey, subscriptionId),
        };
      }

      // New subscriber joining existing connection
      // Close the existing connection and create a new one so Relay sends buffered output
      // This is cleaner than maintaining buffer on frontend
      console.log(`[Relay] New subscriber ${subscriptionId} for ${podKey}, reconnecting to get buffered output`);
      this.disconnect(podKey);
      // Fall through to create new connection
    }

    // Get Relay connection info from Backend
    const relayInfo = await podApi.getTerminalConnection(podKey);
    console.log(`Got Relay connection info for ${podKey}:`, relayInfo.relay_url);

    const ws = this.createRelayWebSocket(podKey, relayInfo);

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
    console.log(`[Relay] Created connection for ${podKey}, subscriber: ${subscriptionId}`);

    return {
      send: (data: string) => this.send(podKey, data),
      unsubscribe: () => this.unsubscribe(podKey, subscriptionId),
      disconnect: () => this.unsubscribe(podKey, subscriptionId),
    };
  }

  /**
   * @deprecated Use subscribe() instead. Kept for backward compatibility.
   * Generates a random subscriptionId which may cause subscriber accumulation issues.
   */
  async connect(podKey: string, onMessage: (data: Uint8Array | string) => void): Promise<ConnectionHandle> {
    // Generate unique ID for backward compatibility
    const subscriptionId = `legacy-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
    console.warn(`[Relay] Using deprecated connect() for ${podKey}. Consider using subscribe() with stable subscriptionId.`);
    return this.subscribe(podKey, subscriptionId, onMessage);
  }

  /**
   * Create WebSocket connection to Relay server
   */
  private createRelayWebSocket(
    podKey: string,
    relayInfo: { relay_url: string; token: string }
  ): WebSocket {
    // Connect to Relay browser endpoint with token auth
    // podKey is embedded in the token, no need to pass separately
    const url = `${relayInfo.relay_url}/browser/terminal?token=${encodeURIComponent(relayInfo.token)}`;
    console.log(`Connecting to Relay: ${relayInfo.relay_url} for pod ${podKey}`);
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
        console.log(`Terminal WebSocket connected to Relay for ${podKey}`);
        this.notifyStatusChange(podKey);
        if (c.pendingResize) {
          // Note: doSendResize signature is (podKey, cols, rows)
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
        console.log(`Terminal WebSocket disconnected for ${podKey}`);
        this.notifyStatusChange(podKey);
        if (c.subscribers.size > 0) {
          this.scheduleReconnect(podKey);
        }
      }
    };
  }

  /**
   * Handle messages from Relay (binary protocol with type prefix)
   */
  private handleRelayMessage(conn: TerminalConnection, data: ArrayBuffer | string): void {
    if (typeof data === "string") {
      // Relay should send binary, but handle string for debugging
      console.warn("Received string message from Relay, expected binary");
      return;
    }

    const bytes = new Uint8Array(data);
    const { type, payload } = decodeMessage(bytes);

    // Debug logging for troubleshooting
    console.log(`[Relay] Received message: type=${type}, payload_len=${payload.length}`);

    switch (type) {
      case MsgType.Snapshot: {
        // Complete terminal snapshot - used to restore terminal state
        // The payload contains serialized ANSI content that can be written directly to xterm
        try {
          const snapshot = JSON.parse(new TextDecoder().decode(payload));
          console.log(`[Relay] Received snapshot: cols=${snapshot.cols}, rows=${snapshot.rows}, alt_screen=${snapshot.is_alt_screen}`);

          // Update PTY size if provided
          if (snapshot.cols > 0 && snapshot.rows > 0) {
            conn.ptySize = { rows: snapshot.rows, cols: snapshot.cols };
          }

          // Forward serialized content to xterm (includes ANSI escape sequences)
          if (snapshot.serialized_content) {
            const content = new TextEncoder().encode(snapshot.serialized_content);
            for (const callback of conn.subscribers.values()) {
              callback(content);
            }
          }
        } catch (e) {
          console.error("Failed to parse snapshot message:", e);
        }
        break;
      }
      case MsgType.Output: {
        // Raw terminal output - forward directly to xterm
        // Note: Buffer is maintained by Relay, not frontend. When a new subscriber
        // joins, we reconnect to Relay to receive buffered output.
        console.log(`[Relay] Forwarding to ${conn.subscribers.size} subscribers, payload_len=${payload.length}`);
        for (const callback of conn.subscribers.values()) {
          callback(payload);
        }
        break;
      }
      case MsgType.Control: {
        // Control messages (JSON)
        try {
          const msg = JSON.parse(new TextDecoder().decode(payload));
          if (msg.type === "pty_resized") {
            conn.ptySize = { rows: msg.rows, cols: msg.cols };
          }
        } catch (e) {
          console.error("Failed to parse control message:", e);
        }
        break;
      }
      case MsgType.RunnerDisconnected: {
        // Runner has disconnected from Relay, waiting for reconnection
        console.warn(`Runner disconnected for pod ${conn.podKey}`);
        conn.runnerDisconnected = true;
        this.notifyStatusChange(conn.podKey);
        // Display disconnection message in terminal
        const disconnectMsg = new TextEncoder().encode(
          "\r\n\x1b[33m⚠ Runner disconnected. Waiting for reconnection...\x1b[0m\r\n"
        );
        for (const callback of conn.subscribers.values()) {
          callback(disconnectMsg);
        }
        break;
      }
      case MsgType.RunnerReconnected: {
        // Runner has reconnected to Relay
        console.log(`Runner reconnected for pod ${conn.podKey}`);
        conn.runnerDisconnected = false;
        this.notifyStatusChange(conn.podKey);
        // Display reconnection message in terminal
        const reconnectMsg = new TextEncoder().encode(
          "\r\n\x1b[32m✓ Runner reconnected.\x1b[0m\r\n"
        );
        for (const callback of conn.subscribers.values()) {
          callback(reconnectMsg);
        }
        break;
      }
      case MsgType.Pong:
        // Pong response - ignore
        break;
      default:
        console.warn(`Unknown message type from Relay: ${type}`);
    }
  }

  send(podKey: string, data: string): void {
    const conn = this.connections.get(podKey);
    if (!conn || conn.ws.readyState !== WebSocket.OPEN) return;

    // Input deduplication: skip if same data sent within deduplication window
    // This prevents duplicate input on mobile devices with network latency
    // Note: Single characters like space are allowed to repeat (for typing "   ")
    // but IME composed strings are deduplicated to prevent "你好你好" issues
    const now = Date.now();
    if (data.length > 1) {
      const lastInput = this.lastInputs.get(podKey);
      if (lastInput && lastInput.data === data && (now - lastInput.time) < this.deduplicateWindow) {
        console.log(`[Relay] Deduplicating input: "${data.substring(0, 20)}..." (within ${this.deduplicateWindow}ms)`);
        return;
      }
      this.lastInputs.set(podKey, { data, time: now });
    }

    // Relay binary protocol: MsgType.Input + raw data
    const message = encodeMessage(MsgType.Input, data);
    conn.ws.send(message);
    conn.lastActivity = now;
  }

  sendResize(podKey: string, cols: number, rows: number): void {
    if (rows <= 0 || cols <= 0) return;

    const existingTimer = this.resizeDebounceTimers.get(podKey);
    if (existingTimer) {
      clearTimeout(existingTimer);
    }

    const timer = setTimeout(() => {
      this.doSendResize(podKey, cols, rows);
      this.resizeDebounceTimers.delete(podKey);
    }, this.resizeDebounceMs);

    this.resizeDebounceTimers.set(podKey, timer);
  }

  private doSendResize(podKey: string, cols: number, rows: number): void {
    const conn = this.connections.get(podKey);
    if (!conn) return;

    if (conn.ws.readyState === WebSocket.OPEN) {
      // Relay binary protocol: MsgType.Resize + 4 bytes (cols: uint16 BE, rows: uint16 BE)
      const payload = new Uint8Array(4);
      payload[0] = (cols >> 8) & 0xff;
      payload[1] = cols & 0xff;
      payload[2] = (rows >> 8) & 0xff;
      payload[3] = rows & 0xff;
      const message = encodeMessage(MsgType.Resize, payload);
      conn.ws.send(message);
      console.log(`[Relay] Sent resize: cols=${cols}, rows=${rows}`);
    } else if (conn.ws.readyState === WebSocket.CONNECTING) {
      conn.pendingResize = { rows, cols };
    }
  }

  forceResize(podKey: string, cols: number, rows: number): void {
    if (rows <= 0 || cols <= 0) return;

    const existingTimer = this.resizeDebounceTimers.get(podKey);
    if (existingTimer) {
      clearTimeout(existingTimer);
      this.resizeDebounceTimers.delete(podKey);
    }

    const conn = this.connections.get(podKey);
    if (!conn) return;

    if (conn.ws.readyState === WebSocket.OPEN) {
      // Relay binary protocol: MsgType.Resize + 4 bytes (cols: uint16 BE, rows: uint16 BE)
      const payload = new Uint8Array(4);
      payload[0] = (cols >> 8) & 0xff;
      payload[1] = cols & 0xff;
      payload[2] = (rows >> 8) & 0xff;
      payload[3] = rows & 0xff;
      const message = encodeMessage(MsgType.Resize, payload);
      conn.ws.send(message);
      console.log(`[Relay] Sent forceResize: cols=${cols}, rows=${rows}`);
    } else if (conn.ws.readyState === WebSocket.CONNECTING) {
      // Save pending resize to send when connection opens
      conn.pendingResize = { rows, cols };
      console.log(`[Relay] Queued pendingResize: cols=${cols}, rows=${rows}`);
    }
  }

  getPtySize(podKey: string): { rows: number; cols: number } | undefined {
    return this.connections.get(podKey)?.ptySize;
  }

  /**
   * Unsubscribe from terminal output.
   * Connection is kept open for disconnectDelay (30s) if no other subscribers remain,
   * allowing quick re-subscribe without reconnection overhead.
   *
   * @param podKey - The pod identifier
   * @param subscriptionId - The subscriber's unique identifier
   */
  unsubscribe(podKey: string, subscriptionId: string): void {
    const conn = this.connections.get(podKey);
    if (!conn) return;

    const deleted = conn.subscribers.delete(subscriptionId);
    if (deleted) {
      console.log(`[Relay] Removed subscriber ${subscriptionId} from ${podKey}, remaining: ${conn.subscribers.size}`);
    }

    if (conn.subscribers.size === 0 && !conn.disconnectTimer) {
      // Schedule delayed disconnect to handle rapid open/close
      console.log(`[Relay] No subscribers left for ${podKey}, scheduling disconnect in ${this.disconnectDelay}ms`);
      conn.disconnectTimer = setTimeout(() => {
        const currentConn = this.connections.get(podKey);
        if (currentConn && currentConn.subscribers.size === 0) {
          console.log(`[Relay] Disconnecting ${podKey} after timeout (no subscribers)`);
          this.disconnect(podKey);
        }
      }, this.disconnectDelay);
    }
  }

  /**
   * @deprecated Use unsubscribe() instead. This method cannot reliably remove listeners
   * because function references change on each component render.
   */
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  removeListener(_podKey: string, _listener: (data: Uint8Array | string) => void): void {
    console.warn(`[Relay] removeListener() is deprecated and unreliable. Use subscribe()/unsubscribe() with stable subscriptionId.`);
    // Cannot reliably remove by function reference - this is the bug we're fixing
    // Just log warning, don't actually try to remove
  }

  disconnect(podKey: string): void {
    const conn = this.connections.get(podKey);
    if (conn) {
      if (conn.reconnectTimer) {
        clearTimeout(conn.reconnectTimer);
        conn.reconnectTimer = null;
      }
      if (conn.disconnectTimer) {
        clearTimeout(conn.disconnectTimer);
        conn.disconnectTimer = null;
      }
      // IMPORTANT: Delete from map BEFORE closing WebSocket
      // This prevents onclose handler from triggering reconnection
      this.connections.delete(podKey);
      // Clean up input deduplication state
      this.lastInputs.delete(podKey);
      // Notify listeners that connection is gone
      this.notifyStatusChange(podKey);
      // Clear all event handlers to prevent stale onclose/onerror from
      // interfering with a new connection created for the same podKey
      // (e.g., during React StrictMode remount or rapid reconnection).
      conn.ws.onopen = null;
      conn.ws.onmessage = null;
      conn.ws.onerror = null;
      conn.ws.onclose = null;
      // Now close WebSocket - handlers are already nulled so no side effects
      if (conn.ws.readyState === WebSocket.OPEN || conn.ws.readyState === WebSocket.CONNECTING) {
        conn.ws.close();
      }
    }
  }

  private scheduleReconnect(podKey: string): void {
    const conn = this.connections.get(podKey);
    if (!conn || conn.reconnectAttempts >= this.maxReconnectAttempts) {
      if (conn) console.log(`Max reconnect attempts reached for ${podKey}`);
      return;
    }

    // Exponential backoff with jitter (±20%) to prevent thundering herd
    const baseDelay = Math.min(this.baseReconnectDelay * Math.pow(2, conn.reconnectAttempts), 30000);
    const jitter = baseDelay * (Math.random() * 0.4 - 0.2);
    const delay = Math.round(baseDelay + jitter);
    console.log(`Scheduling reconnect for ${podKey} in ${delay}ms`);

    conn.reconnectTimer = setTimeout(() => {
      conn.reconnectAttempts++;
      this.reconnect(podKey);
    }, delay);
  }

  private async reconnect(podKey: string): Promise<void> {
    const oldConn = this.connections.get(podKey);
    if (!oldConn || oldConn.subscribers.size === 0) return;

    console.log(`Reconnecting terminal for ${podKey}...`);

    // Preserve subscribers and their IDs
    const subscribersCopy = new Map(oldConn.subscribers);
    const reconnectAttempts = oldConn.reconnectAttempts;

    if (oldConn.ws.readyState === WebSocket.OPEN || oldConn.ws.readyState === WebSocket.CONNECTING) {
      oldConn.ws.close();
    }
    this.connections.delete(podKey);

    // Re-subscribe the first subscriber to create connection
    const firstEntry = subscribersCopy.entries().next().value;
    if (firstEntry) {
      const [firstId, firstCallback] = firstEntry;
      await this.subscribe(podKey, firstId, firstCallback);

      const newConn = this.connections.get(podKey);
      if (newConn) {
        // Add remaining subscribers
        subscribersCopy.forEach((callback, id) => {
          if (id !== firstId) {
            newConn.subscribers.set(id, callback);
          }
        });
        newConn.reconnectAttempts = reconnectAttempts;
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

  isRunnerDisconnected(podKey: string): boolean {
    return this.connections.get(podKey)?.runnerDisconnected ?? false;
  }
}

// Singleton instance
export const terminalPool = new TerminalConnectionPool();
