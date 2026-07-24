import { getRelayManager } from "@agentsmesh/service-runtime";
import type {
  ConnectionStatus,
  RelayStatusInfo,
  StatusListener,
} from "./relayConnectionTypes";
import type { RelayPodSession } from "./relayPodSession";
import type { RelayPodRegistry } from "./relayPodRegistry";

type AcpListener = (msgType: number, payload: unknown) => void;
type StatusInfoRaw = { status: string; runnerDisconnected: boolean };

const NONE: RelayStatusInfo = { status: "none", runnerDisconnected: false };

export class RelayConnectionEvents {
  private readonly statusListeners = new Map<string, Set<StatusListener>>();
  private readonly acpListeners = new Map<string, Set<AcpListener>>();
  private readonly statusCache = new Map<string, RelayStatusInfo>();
  private readonly statusUpstream = new Set<string>();
  private readonly acpUpstream = new Set<string>();
  private disconnectHookWired = false;

  constructor(
    private readonly pods: RelayPodRegistry,
    private readonly flushPendingResize: (podKey: string, session: RelayPodSession) => void,
  ) {}

  onStatusChange(podKey: string, listener: StatusListener): () => void {
    let set = this.statusListeners.get(podKey);
    if (!set) { set = new Set(); this.statusListeners.set(podKey, set); }
    set.add(listener);
    this.ensureStatusUpstream(podKey);
    listener(this.statusCache.get(podKey) ?? NONE);
    return () => {
      set.delete(listener);
      if (set.size === 0) this.statusListeners.delete(podKey);
    };
  }

  onAcpMessage(podKey: string, listener: AcpListener): () => void {
    let set = this.acpListeners.get(podKey);
    if (!set) { set = new Set(); this.acpListeners.set(podKey, set); }
    set.add(listener);
    this.ensureAcpUpstream(podKey);
    return () => {
      set.delete(listener);
      if (set.size === 0) this.acpListeners.delete(podKey);
    };
  }

  getStatus(podKey: string): ConnectionStatus | "none" {
    return this.statusCache.get(podKey)?.status ?? "none";
  }

  isConnected(podKey: string): boolean {
    return this.statusCache.get(podKey)?.status === "connected";
  }

  isRunnerDisconnected(podKey: string): boolean {
    return this.statusCache.get(podKey)?.runnerDisconnected ?? false;
  }

  ensureStatusUpstream(podKey: string): void {
    this.ensureDisconnectHook();
    if (this.statusUpstream.has(podKey)) return;
    this.statusUpstream.add(podKey);
    void getRelayManager().on_status_change(podKey, (raw: StatusInfoRaw) => {
      const session = this.pods.get(podKey);
      const info: RelayStatusInfo =
        raw.status === "disconnected" && !session?.hasStartedConnection()
          ? { status: "none", runnerDisconnected: raw.runnerDisconnected }
          : { status: raw.status as ConnectionStatus, runnerDisconnected: raw.runnerDisconnected };
      this.statusCache.set(podKey, info);
      if (info.status === "connected") {
        const activeSession = session ?? this.pods.getOrCreate(podKey);
        activeSession.setDataReady(true);
        this.flushPendingResize(podKey, activeSession);
      } else {
        session?.setDataReady(false);
      }
      const listeners = this.statusListeners.get(podKey);
      if (listeners) for (const listener of listeners) listener(info);
    });
  }

  private ensureAcpUpstream(podKey: string): void {
    this.ensureDisconnectHook();
    if (this.acpUpstream.has(podKey)) return;
    this.acpUpstream.add(podKey);
    void getRelayManager().on_acp_message(podKey, (msgType: number, payload: unknown) => {
      const listeners = this.acpListeners.get(podKey);
      if (listeners) for (const listener of listeners) listener(msgType, payload);
    });
  }

  // Rust clears pod listeners before this callback. Preserve any newer JS work,
  // reset visible status immediately, then rebind everything that remains live.
  private ensureDisconnectHook(): void {
    if (this.disconnectHookWired) return;
    this.disconnectHookWired = true;
    void getRelayManager().on_pod_disconnected((podKey: string) => {
      this.statusUpstream.delete(podKey);
      this.acpUpstream.delete(podKey);
      this.statusCache.delete(podKey);
      const keepSession = this.pods.handleDisconnected(podKey);
      const statusListeners = this.statusListeners.get(podKey);
      if (statusListeners) for (const listener of statusListeners) listener(NONE);

      if (keepSession || statusListeners) this.ensureStatusUpstream(podKey);
      if (this.acpListeners.has(podKey)) this.ensureAcpUpstream(podKey);
    });
  }
}
