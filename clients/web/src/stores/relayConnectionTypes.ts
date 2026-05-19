
import type { IRelayTransport } from "./relayBackend";

export type ConnectionStatus = "connecting" | "connected" | "disconnected" | "error";

export interface RelayConnection {
  transport: IRelayTransport;
  podKey: string;
  status: ConnectionStatus;
  lastActivity: number;
  subscribers: Map<string, (data: Uint8Array | string) => void>;
  reconnectAttempts: number;
  reconnectTimer: ReturnType<typeof setTimeout> | null;
  disconnectTimer: ReturnType<typeof setTimeout> | null;
  snapshotTimer: ReturnType<typeof setTimeout> | null;
  snapshotReceived: boolean;
  pendingResize?: { rows: number; cols: number };
  podSize?: { rows: number; cols: number };
  relayUrl: string;
  relayToken: string;
  runnerDisconnected: boolean;
}

export interface ConnectionHandle {
  send: (data: string) => void;
  unsubscribe: () => void;
}

export type RelayStatusInfo = {
  status: RelayConnection["status"] | "none";
  runnerDisconnected: boolean;
};

export type StatusListener = (info: RelayStatusInfo) => void;
