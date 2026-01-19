import { useAuthStore } from "@/stores/auth";
import { getWsBaseUrl } from "@/lib/env";
import type {
  EventType,
  EventHandler,
  RealtimeEvent,
  ConnectionState,
} from "./types";

const WS_BASE_URL = getWsBaseUrl();

/**
 * Configuration options for EventSubscriptionManager
 */
interface EventSubscriptionManagerOptions {
  /** Maximum number of reconnection attempts (default: 10) */
  maxReconnectAttempts?: number;
  /** Initial reconnection delay in ms (default: 1000) */
  initialReconnectDelay?: number;
  /** Maximum reconnection delay in ms (default: 30000) */
  maxReconnectDelay?: number;
  /** Ping interval in ms (default: 30000) */
  pingInterval?: number;
  /** Pong timeout in ms (default: 10000) */
  pongTimeout?: number;
  /** Callback when connection state changes */
  onConnectionStateChange?: (state: ConnectionState) => void;
}

/**
 * EventSubscriptionManager manages WebSocket connections for real-time events
 *
 * Features:
 * - Automatic reconnection with exponential backoff
 * - Heartbeat detection (ping/pong)
 * - Event subscription/unsubscription
 * - Connection state management
 */
export class EventSubscriptionManager {
  private ws: WebSocket | null = null;
  private connectionState: ConnectionState = "disconnected";
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private pongTimer: ReturnType<typeof setTimeout> | null = null;

  // Event handlers by event type
  private handlers: Map<EventType, Set<EventHandler>> = new Map();
  // Handlers that listen to all events
  private globalHandlers: Set<EventHandler> = new Set();

  // Configuration
  private readonly maxReconnectAttempts: number;
  private readonly initialReconnectDelay: number;
  private readonly maxReconnectDelay: number;
  private readonly pingInterval: number;
  private readonly pongTimeout: number;

  // Connection state change listeners
  private connectionStateListeners: Set<(state: ConnectionState) => void> = new Set();

  constructor(options: EventSubscriptionManagerOptions = {}) {
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
    this.initialReconnectDelay = options.initialReconnectDelay ?? 1000;
    this.maxReconnectDelay = options.maxReconnectDelay ?? 30000;
    this.pingInterval = options.pingInterval ?? 30000;
    this.pongTimeout = options.pongTimeout ?? 10000;

    // Register initial listener if provided
    if (options.onConnectionStateChange) {
      this.connectionStateListeners.add(options.onConnectionStateChange);
    }
  }

  /**
   * Get the WebSocket URL for the events channel
   */
  private getWebSocketUrl(): string | null {
    const { currentOrg, token } = useAuthStore.getState();
    if (!currentOrg || !token) {
      return null;
    }
    return `${WS_BASE_URL}/api/v1/orgs/${currentOrg.slug}/ws/events?token=${token}`;
  }

  /**
   * Update connection state and notify listeners
   */
  private setConnectionState(state: ConnectionState): void {
    if (this.connectionState !== state) {
      this.connectionState = state;
      // Notify all registered listeners
      this.connectionStateListeners.forEach((listener) => {
        try {
          listener(state);
        } catch (error) {
          console.error("[EventSubscriptionManager] Connection state listener error:", error);
        }
      });
    }
  }

  /**
   * Connect to the events WebSocket
   */
  connect(): void {
    // Don't connect if already connected or connecting
    if (this.ws && (this.connectionState === "connected" || this.connectionState === "connecting")) {
      return;
    }

    const url = this.getWebSocketUrl();
    if (!url) {
      console.warn("[EventSubscriptionManager] Cannot connect: no org or token");
      return;
    }

    this.setConnectionState("connecting");
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log("[EventSubscriptionManager] Connected");
      this.setConnectionState("connected");
      this.reconnectAttempts = 0;
      this.startPingInterval();
    };

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as RealtimeEvent;
        this.handleMessage(message);
      } catch (error) {
        console.error("[EventSubscriptionManager] Failed to parse message:", error);
      }
    };

    this.ws.onclose = (event) => {
      console.log("[EventSubscriptionManager] Disconnected:", event.code, event.reason);
      this.cleanup();

      // Don't reconnect if it was a clean close
      if (event.code === 1000) {
        this.setConnectionState("disconnected");
        return;
      }

      // Attempt to reconnect
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      // WebSocket error events don't contain useful info, log connection details instead
      console.error("[EventSubscriptionManager] WebSocket error:", {
        url: url,
        readyState: this.ws?.readyState,
        // Common causes: server not running, CORS, network issues
      });
    };
  }

  /**
   * Disconnect from the events WebSocket
   */
  disconnect(): void {
    this.cleanup();
    if (this.ws) {
      this.ws.close(1000, "Client disconnect");
      this.ws = null;
    }
    this.setConnectionState("disconnected");
    this.reconnectAttempts = 0;
  }

  /**
   * Subscribe to a specific event type
   */
  subscribe<T = unknown>(eventType: EventType, handler: EventHandler<T>): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    this.handlers.get(eventType)!.add(handler as EventHandler);

    // Return unsubscribe function
    return () => {
      this.handlers.get(eventType)?.delete(handler as EventHandler);
    };
  }

  /**
   * Subscribe to all events
   */
  subscribeAll(handler: EventHandler): () => void {
    this.globalHandlers.add(handler);
    return () => {
      this.globalHandlers.delete(handler);
    };
  }

  /**
   * Get current connection state
   */
  getConnectionState(): ConnectionState {
    return this.connectionState;
  }

  /**
   * Subscribe to connection state changes
   * @returns Unsubscribe function
   */
  onConnectionStateChange(listener: (state: ConnectionState) => void): () => void {
    this.connectionStateListeners.add(listener);
    // Immediately call with current state
    listener(this.connectionState);
    return () => {
      this.connectionStateListeners.delete(listener);
    };
  }

  /**
   * Handle incoming WebSocket message
   */
  private handleMessage(event: RealtimeEvent): void {
    // Handle ping/pong
    if (event.type === "pong") {
      this.clearPongTimeout();
      return;
    }

    // Dispatch to specific handlers
    const handlers = this.handlers.get(event.type);
    if (handlers) {
      handlers.forEach((handler) => {
        try {
          handler(event);
        } catch (error) {
          console.error(`[EventSubscriptionManager] Handler error for ${event.type}:`, error);
        }
      });
    }

    // Dispatch to global handlers
    this.globalHandlers.forEach((handler) => {
      try {
        handler(event);
      } catch (error) {
        console.error("[EventSubscriptionManager] Global handler error:", error);
      }
    });
  }

  /**
   * Start the ping interval
   */
  private startPingInterval(): void {
    this.stopPingInterval();
    this.pingTimer = setInterval(() => {
      this.sendPing();
    }, this.pingInterval);
  }

  /**
   * Stop the ping interval
   */
  private stopPingInterval(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }

  /**
   * Send a ping message
   */
  private sendPing(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "ping", timestamp: Date.now() }));
      this.startPongTimeout();
    }
  }

  /**
   * Start pong timeout
   */
  private startPongTimeout(): void {
    this.clearPongTimeout();
    this.pongTimer = setTimeout(() => {
      console.warn("[EventSubscriptionManager] Pong timeout, reconnecting...");
      this.ws?.close(4000, "Pong timeout");
    }, this.pongTimeout);
  }

  /**
   * Clear pong timeout
   */
  private clearPongTimeout(): void {
    if (this.pongTimer) {
      clearTimeout(this.pongTimer);
      this.pongTimer = null;
    }
  }

  /**
   * Schedule a reconnection attempt
   */
  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("[EventSubscriptionManager] Max reconnect attempts reached");
      this.setConnectionState("disconnected");
      return;
    }

    this.setConnectionState("reconnecting");
    this.reconnectAttempts++;

    // Exponential backoff with jitter
    const delay = Math.min(
      this.initialReconnectDelay * Math.pow(2, this.reconnectAttempts - 1) +
        Math.random() * 1000,
      this.maxReconnectDelay
    );

    console.log(
      `[EventSubscriptionManager] Reconnecting in ${Math.round(delay)}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`
    );

    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  /**
   * Cleanup timers and state
   */
  private cleanup(): void {
    this.stopPingInterval();
    this.clearPongTimeout();

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}

// Singleton instance
let instance: EventSubscriptionManager | null = null;

// Listeners that get notified when the manager is reset
type ManagerResetListener = (newManager: EventSubscriptionManager) => void;
const managerResetListeners: Set<ManagerResetListener> = new Set();

/**
 * Get the singleton EventSubscriptionManager instance
 */
export function getEventSubscriptionManager(): EventSubscriptionManager {
  if (!instance) {
    instance = new EventSubscriptionManager({
      onConnectionStateChange: (state) => {
        console.log(`[EventSubscriptionManager] Connection state: ${state}`);
      },
    });
  }
  return instance;
}

/**
 * Reset the singleton instance (for testing or org switching)
 */
export function resetEventSubscriptionManager(): void {
  if (instance) {
    instance.disconnect();
    instance = null;
  }
  // Create new instance and notify listeners
  const newManager = getEventSubscriptionManager();
  managerResetListeners.forEach((listener) => {
    try {
      listener(newManager);
    } catch (error) {
      console.error("[EventSubscriptionManager] Reset listener error:", error);
    }
  });
}

/**
 * Subscribe to manager reset events
 * This is called when the manager is reset (e.g., on org switch)
 * Subscribers should re-register their event handlers with the new manager
 * @returns Unsubscribe function
 */
export function onManagerReset(listener: ManagerResetListener): () => void {
  managerResetListeners.add(listener);
  return () => {
    managerResetListeners.delete(listener);
  };
}
