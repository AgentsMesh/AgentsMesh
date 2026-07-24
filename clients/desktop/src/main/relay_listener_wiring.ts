import { logEvent, type AppState } from "@agentsmesh/node-bridge";
import type { RelayOutputSubscriptions } from "./relay_output_subscriptions";
import type { WindowRegistry } from "./window_registry";
import {
  parsePodAcpEvent,
  parsePodDriverDisconnected,
  parsePodStatusEvent,
  type PodStatusEvent,
} from "./relay_listener_events";

export interface RelayPodListeners {
  leaseId: string;
  onStatus: (_error: unknown, json: string) => void;
  onAcp: (_error: unknown, json: string) => void;
  onBound: (_error: unknown, generation: number) => void;
}

interface PodListenerLease extends RelayPodListeners {
  generation: number;
  retiredGeneration: number;
  lastStatusRevision: number;
  rebind?: Promise<void>;
}

// A bridge can be rebuilt before Rust finishes disconnecting its drivers.
let nextListenerLease = 0;

export class RelayListenerWiring {
  private readonly leases = new Map<string, PodListenerLease>();

  constructor(
    private readonly appState: AppState,
    private readonly subscriptions: RelayOutputSubscriptions,
    private readonly registry: WindowRegistry,
  ) {}

  forPod(podKey: string): PodListenerLease {
    const existing = this.leases.get(podKey);
    if (existing) return existing;
    const lease: PodListenerLease = {
      leaseId: `desktop-listeners:${++nextListenerLease}`,
      generation: 0,
      retiredGeneration: 0,
      lastStatusRevision: -1,
      onStatus: (_error, json) => this.handleStatus(podKey, lease, json),
      onAcp: (_error, json) => this.handleAcp(podKey, lease, json),
      onBound: (_error, generation) => { this.accept(podKey, lease, generation); },
    };
    this.leases.set(podKey, lease);
    return lease;
  }

  dropPod(podKey: string): void {
    this.leases.delete(podKey);
  }

  dropPodIfUnused(podKey: string): void {
    if (!this.subscriptions.hasPod(podKey)) this.dropPod(podKey);
  }

  clear(): void {
    this.leases.clear();
  }

  handleDriverDisconnected(raw: string): void {
    const event = parsePodDriverDisconnected(raw);
    if (!event) {
      logEvent("warn", "relay", `malformed pod disconnect: ${raw}`);
      return;
    }
    const { podKey, generation } = event;
    const lease = this.leases.get(podKey);
    if (lease) {
      lease.retiredGeneration = Math.max(lease.retiredGeneration, generation);
      if (lease.generation === generation) {
        lease.generation = 0;
        lease.lastStatusRevision = -1;
      }
    }
    if (this.subscriptions.hasPod(podKey)) {
      const current = lease ?? this.forPod(podKey);
      if (current.generation === 0 || current.generation <= generation) {
        void this.rebind(podKey, current);
      } else {
        logEvent("debug", "relay", `ignore stale pod disconnect ${podKey}/${generation}`);
      }
      return;
    }
    logEvent("info", "relay", `pod disconnected ${podKey}/${generation}`);
    if (this.leases.get(podKey) === lease) this.dropPod(podKey);
    this.registry.broadcast("relay:pod-disconnected", { podKey, generation });
  }

  private handleStatus(podKey: string, lease: PodListenerLease, json: string): void {
    if (this.leases.get(podKey) !== lease) return;
    const event = parsePodStatusEvent(json);
    if (!event) {
      logEvent("warn", "relay", `malformed status event for ${podKey}`);
      return;
    }
    if (event.generation <= lease.retiredGeneration || event.generation < lease.generation) return;
    if (event.generation > lease.generation) this.accept(podKey, lease, event.generation);
    this.deliverStatus(podKey, lease, event, json);
  }

  private handleAcp(podKey: string, lease: PodListenerLease, json: string): void {
    if (this.leases.get(podKey) !== lease) return;
    const event = parsePodAcpEvent(json);
    if (!event) {
      logEvent("warn", "relay", `malformed ACP event for ${podKey}`);
      return;
    }
    if (event.generation <= lease.retiredGeneration || event.generation < lease.generation) return;
    if (event.generation > lease.generation) this.accept(podKey, lease, event.generation);
    this.subscriptions.sendToPod(podKey, "relay:acp", { podKey, json });
  }

  private accept(podKey: string, lease: PodListenerLease, generation: number): boolean {
    if (!Number.isSafeInteger(generation) || generation <= 0) return false;
    if (this.leases.get(podKey) !== lease || generation <= lease.retiredGeneration) return false;
    if (generation < lease.generation) return false;
    if (generation > lease.generation) lease.lastStatusRevision = -1;
    lease.generation = generation;
    return true;
  }

  private deliverStatus(
    podKey: string,
    lease: PodListenerLease,
    event: PodStatusEvent,
    json: string,
  ): void {
    if (event.revision <= lease.lastStatusRevision) return;
    lease.lastStatusRevision = event.revision;
    this.subscriptions.sendToPod(podKey, "relay:status", { podKey, json });
  }

  private rebind(podKey: string, lease: PodListenerLease): Promise<void> {
    if (lease.rebind) return lease.rebind;
    const rebind = this.appState.relayBindPodListeners(
      podKey,
      lease.onStatus,
      lease.onAcp,
      lease.leaseId,
    )
      .then((generation) => { this.accept(podKey, lease, generation); })
      .catch((error: unknown) => {
        logEvent("warn", "relay", `listener rebind ${podKey} failed: ${error}`);
      })
      .finally(() => {
        if (this.leases.get(podKey) === lease && lease.rebind === rebind) lease.rebind = undefined;
      });
    lease.rebind = rebind;
    return rebind;
  }
}
