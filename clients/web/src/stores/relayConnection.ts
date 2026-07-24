import { getRelayManager } from "@agentsmesh/service-runtime";
import { selectRelayEndpoint } from "./relayEndpointSelection";
import { RelayPodSession, type RelayResizeRequest } from "./relayPodSession";
import { RelayPodRegistry } from "./relayPodRegistry";
import { RelayConnectionEvents } from "./relayConnectionEvents";
import { relayAbortError, waitForRelayWithAbort } from "./relaySubscriptionAbort";
import type { ConnectionStatus, ConnectionHandle, StatusListener } from "./relayConnectionTypes";

export type { ConnectionStatus, ConnectionHandle, RelayStatusInfo } from "./relayConnectionTypes";

type OnMessage = (data: Uint8Array | string) => void;
type AcpListener = (msgType: number, payload: unknown) => void;

// Thin web/Electron adapter over the Rust relay pool. It owns endpoint selection,
// the legacy "none" status fan-out, and pre-baseline resize retention; transport,
// reconnect, codec, snapshot replay, dedup, and debounce remain in Rust.
class RelayConnectionPool {
  private readonly pods = new RelayPodRegistry();
  private readonly events = new RelayConnectionEvents(
    this.pods,
    (podKey, session) => this.flushPendingResize(podKey, session),
  );

  constructor() {
    if (typeof window !== "undefined") {
      window.addEventListener("beforeunload", () => this.disconnectAll());
    }
  }

  private mgr() {
    return getRelayManager();
  }

  async subscribe(
    podKey: string,
    subscriptionId: string,
    onMessage: OnMessage,
    signal?: AbortSignal,
  ): Promise<ConnectionHandle> {
    this.events.ensureStatusUpstream(podKey);
    const session = this.pods.getOrCreate(podKey);
    const attempt = session.beginSubscription(subscriptionId);
    let managerStarted = false;
    const cancel = () => {
      if (!session.removeSubscription(subscriptionId, attempt)) return;
      if (managerStarted) {
        const unsubscribe = this.mgr().unsubscribe(podKey, subscriptionId);
        // Enqueue removal in the manager before a ready sibling can release a
        // queued resize. The Rust actor is the final baseline gate, while this
        // preserves the same ordering at the adapter boundary.
        this.flushPendingResize(podKey, session);
        void unsubscribe;
      } else {
        this.flushPendingResize(podKey, session);
      }
    };
    signal?.addEventListener("abort", cancel, { once: true });
    try {
      if (signal?.aborted) {
        cancel();
        throw relayAbortError();
      }
      const endpoint = selectRelayEndpoint(podKey);
      const { url, token } = signal
        ? await waitForRelayWithAbort(endpoint, signal)
        : await endpoint;
      if (signal?.aborted) throw relayAbortError();
      managerStarted = session.markManagerStarted(attempt);
      if (!managerStarted) {
        throw new DOMException("Relay subscription cancelled", "AbortError");
      }
      // Both WASM and Electron resolve subscribe only after this subscriber's
      // baseline has been delivered. Resizes before then would mutate the
      // Runner VT before the recovery snapshot is materialized.
      await this.mgr().subscribe(podKey, subscriptionId, url, token, onMessage);
      if (signal?.aborted) {
        cancel();
        throw relayAbortError();
      }
      if (!session.markSubscriptionReady(attempt)) {
        throw new DOMException("Relay subscription superseded", "AbortError");
      }
      this.flushPendingResize(podKey, session);
      return {
        send: (data) => this.send(podKey, data),
        unsubscribe: cancel,
      };
    } catch (error) {
      if (session.removeSubscription(subscriptionId, attempt)) {
        this.flushPendingResize(podKey, session);
      }
      throw error;
    } finally {
      signal?.removeEventListener("abort", cancel);
    }
  }

  send(podKey: string, data: string): void {
    void this.mgr().send(podKey, data);
  }

  sendResize(podKey: string, cols: number, rows: number): void {
    this.sendOrQueueResize(podKey, cols, rows, false);
  }

  forceResize(podKey: string, cols: number, rows: number): void {
    this.sendOrQueueResize(podKey, cols, rows, true);
  }

  unsubscribe(podKey: string, subscriptionId: string): void {
    const session = this.pods.get(podKey);
    const removed = session?.removeSubscription(subscriptionId) === true;
    const unsubscribe = this.mgr().unsubscribe(podKey, subscriptionId);
    if (removed) this.flushPendingResize(podKey, session);
    void unsubscribe;
  }

  disconnect(podKey: string): void {
    this.pods.clear(podKey);
    void this.mgr().disconnect(podKey);
  }

  disconnectAll(): void {
    this.pods.clearAll();
    void this.mgr().disconnect_all();
  }

  onStatusChange(podKey: string, listener: StatusListener): () => void {
    return this.events.onStatusChange(podKey, listener);
  }

  onAcpMessage(podKey: string, listener: AcpListener): () => void {
    return this.events.onAcpMessage(podKey, listener);
  }

  sendAcpCommand(podKey: string, command: Record<string, unknown>): void {
    void this.mgr().send_acp_command(podKey, JSON.stringify(command));
  }

  getStatus(podKey: string): ConnectionStatus | "none" {
    return this.events.getStatus(podKey);
  }

  isConnected(podKey: string): boolean {
    return this.events.isConnected(podKey);
  }

  isRunnerDisconnected(podKey: string): boolean {
    return this.events.isRunnerDisconnected(podKey);
  }

  getPodSize(): { rows: number; cols: number } | undefined {
    // podSize lives in the Rust pool; no synchronous consumer needs it.
    return undefined;
  }

  private sendOrQueueResize(
    podKey: string,
    cols: number,
    rows: number,
    force: boolean,
  ): void {
    if (cols <= 0 || rows <= 0) return;
    const resize = this.pods.get(podKey)?.scheduleResize({ cols, rows, force });
    if (resize) this.dispatchResize(podKey, resize);
  }

  private flushPendingResize(podKey: string, session: RelayPodSession): void {
    const resize = session.takeFlushableResize();
    if (resize) this.dispatchResize(podKey, resize);
  }

  private dispatchResize(podKey: string, resize: RelayResizeRequest): void {
    if (resize.force) {
      void this.mgr().force_resize(podKey, resize.cols, resize.rows);
    } else {
      void this.mgr().send_resize(podKey, resize.cols, resize.rows);
    }
  }
}

function getOrCreatePool(): RelayConnectionPool {
  const key = "__relayPool" as keyof typeof globalThis;
  const existing = globalThis[key] as RelayConnectionPool | undefined;
  if (existing) {
    if (process.env.NODE_ENV === "development") existing.disconnectAll();
    else return existing;
  }
  const pool = new RelayConnectionPool();
  (globalThis as Record<string, unknown>)[key] = pool;
  return pool;
}

export const relayPool = getOrCreatePool();
