/**
 * Types and interfaces for relay WebSocket connections.
 */

/**
 * Connection status of a relay WebSocket.
 * Single source of truth — import this type instead of redefining inline.
 */
export type ConnectionStatus = "connecting" | "connected" | "disconnected" | "error";

/**
 * Relay connection state
 */
export interface RelayConnection {
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
  /** Timer for snapshot resync retry */
  snapshotTimer: ReturnType<typeof setTimeout> | null;
  snapshotReceived: boolean;
  pendingResize?: { rows: number; cols: number };
  podSize?: { rows: number; cols: number };
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
  status: RelayConnection["status"] | "none";
  runnerDisconnected: boolean;
};

export type StatusListener = (info: RelayStatusInfo) => void;
