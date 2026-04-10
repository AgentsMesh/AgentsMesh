import type {
  EventType,
  EventHandler,
  RealtimeEvent,
  ConnectionState,
} from "./types";

export interface EventSubscriptionManagerOptions {
  maxReconnectAttempts?: number;
  initialReconnectDelay?: number;
  maxReconnectDelay?: number;
  pingInterval?: number;
  pongTimeout?: number;
  onConnectionStateChange?: (state: ConnectionState) => void;
}

export class EventSubscriptionManager {
  private ws: WebSocket | null = null;
  private urlProvider: (() => string) | null = null;
  private connectionState: ConnectionState = "disconnected";
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private pongTimer: ReturnType<typeof setTimeout> | null = null;

  private handlers: Map<EventType, Set<EventHandler>> = new Map();
  private globalHandlers: Set<EventHandler> = new Set();

  private readonly maxReconnectAttempts: number;
  private readonly initialReconnectDelay: number;
  private readonly maxReconnectDelay: number;
  private readonly pingInterval: number;
  private readonly pongTimeout: number;

  private connectionStateListeners: Set<(state: ConnectionState) => void> = new Set();

  constructor(options: EventSubscriptionManagerOptions = {}) {
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
    this.initialReconnectDelay = options.initialReconnectDelay ?? 1000;
    this.maxReconnectDelay = options.maxReconnectDelay ?? 30000;
    this.pingInterval = options.pingInterval ?? 30000;
    this.pongTimeout = options.pongTimeout ?? 10000;

    if (options.onConnectionStateChange) {
      this.connectionStateListeners.add(options.onConnectionStateChange);
    }
  }

  private setConnectionState(state: ConnectionState): void {
    if (this.connectionState !== state) {
      this.connectionState = state;
      this.connectionStateListeners.forEach((listener) => {
        try { listener(state); }
        catch (error) { console.error("[EventSubscriptionManager] Connection state listener error:", error); }
      });
    }
  }

  connect(urlProvider?: (() => string) | string): void {
    if (typeof urlProvider === "function") this.urlProvider = urlProvider;
    else if (urlProvider) this.urlProvider = () => urlProvider;
    if (this.ws && (this.connectionState === "connected" || this.connectionState === "connecting")) return;
    const url = this.urlProvider?.();
    if (!url) { console.warn("[EventSubscriptionManager] Cannot connect: no URL"); return; }

    this.setConnectionState("connecting");
    this.ws = new WebSocket(url);
    this.ws.onopen = () => { this.setConnectionState("connected"); this.reconnectAttempts = 0; this.startPingInterval(); };
    this.ws.onmessage = (event) => { try { this.handleMessage(JSON.parse(event.data) as RealtimeEvent); } catch (error) { console.error("[EventSubscriptionManager] Failed to parse message:", error); } };
    this.ws.onclose = (event) => { this.cleanup(); if (event.code === 1000) { this.setConnectionState("disconnected"); return; } this.scheduleReconnect(); };
    this.ws.onerror = () => { console.warn("[EventSubscriptionManager] WebSocket error:", { readyState: this.ws?.readyState }); };
  }

  disconnect(): void {
    this.cleanup();
    if (this.ws) { this.ws.close(1000, "Client disconnect"); this.ws = null; }
    this.setConnectionState("disconnected");
    this.reconnectAttempts = 0;
  }

  subscribe<T = unknown>(eventType: EventType, handler: EventHandler<T>): () => void {
    if (!this.handlers.has(eventType)) this.handlers.set(eventType, new Set());
    this.handlers.get(eventType)!.add(handler as EventHandler);
    return () => { this.handlers.get(eventType)?.delete(handler as EventHandler); };
  }

  subscribeAll(handler: EventHandler): () => void {
    this.globalHandlers.add(handler);
    return () => { this.globalHandlers.delete(handler); };
  }

  getConnectionState(): ConnectionState { return this.connectionState; }

  onConnectionStateChange(listener: (state: ConnectionState) => void): () => void {
    this.connectionStateListeners.add(listener);
    listener(this.connectionState);
    return () => { this.connectionStateListeners.delete(listener); };
  }

  private handleMessage(event: RealtimeEvent): void {
    if (event.type === "pong") { this.clearPongTimeout(); return; }
    this.handlers.get(event.type)?.forEach((handler) => {
      try { handler(event); } catch (error) { console.error(`[EventSubscriptionManager] Handler error for ${event.type}:`, error); }
    });
    this.globalHandlers.forEach((handler) => {
      try { handler(event); } catch (error) { console.error("[EventSubscriptionManager] Global handler error:", error); }
    });
  }

  private startPingInterval(): void {
    this.stopPingInterval();
    this.pingTimer = setInterval(() => this.sendPing(), this.pingInterval);
  }

  private stopPingInterval(): void {
    if (this.pingTimer) { clearInterval(this.pingTimer); this.pingTimer = null; }
  }

  private sendPing(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "ping", timestamp: Date.now() }));
      this.startPongTimeout();
    }
  }

  private startPongTimeout(): void {
    this.clearPongTimeout();
    this.pongTimer = setTimeout(() => {
      console.warn("[EventSubscriptionManager] Pong timeout, reconnecting...");
      this.ws?.close(4000, "Pong timeout");
    }, this.pongTimeout);
  }

  private clearPongTimeout(): void {
    if (this.pongTimer) { clearTimeout(this.pongTimer); this.pongTimer = null; }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("[EventSubscriptionManager] Max reconnect attempts reached");
      this.setConnectionState("disconnected");
      return;
    }
    this.setConnectionState("reconnecting");
    this.reconnectAttempts++;
    const delay = Math.min(
      this.initialReconnectDelay * Math.pow(2, this.reconnectAttempts - 1) + Math.random() * 1000,
      this.maxReconnectDelay
    );
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }

  private cleanup(): void {
    this.stopPingInterval();
    this.clearPongTimeout();
    if (this.reconnectTimer) { clearTimeout(this.reconnectTimer); this.reconnectTimer = null; }
  }
}
