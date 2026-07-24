import type {
  RelayOutputPayload,
  RelayPushChannel,
  RelayPushPayload,
} from "@agentsmesh/electron-adapter/relay-ipc-contract";
import { type WindowRegistry } from "./window_registry";

export interface SubscriptionAttempt {
  podKey: string;
  wcId: number;
  subId: string;
  attemptId: string;
  coreSubId: string;
  supersededCoreSubId?: string;
}

export interface SubscriptionCommit {
  committed: boolean;
  replaced?: string;
}

export interface RemovedSubscriptions {
  podKey: string;
  coreSubIds: string[];
  podUnused: boolean;
}

interface PendingOutput extends SubscriptionAttempt {
  bytes: number[];
}

interface SubscriptionRoute {
  active?: string;
  candidate?: string;
}

// Owns the renderer-subscription identity map and the high-volume output batch.
// A core subscription is valid only while it is the current generation for its
// (pod, webContents, renderer subId) tuple.
export class RelayOutputSubscriptions {
  private readonly byPod = new Map<string, Map<number, Map<string, SubscriptionRoute>>>();
  private readonly pending = new Map<string, PendingOutput>();
  private nextCoreSub = 0;
  private flushScheduled = false;

  constructor(private readonly registry: WindowRegistry) {}

  begin(wcId: number, podKey: string, subId: string, attemptId: string): SubscriptionAttempt {
    let byWc = this.byPod.get(podKey);
    if (!byWc) {
      byWc = new Map();
      this.byPod.set(podKey, byWc);
    }
    let subs = byWc.get(wcId);
    if (!subs) {
      subs = new Map();
      byWc.set(wcId, subs);
    }
    let route = subs.get(subId);
    if (!route) {
      route = {};
      subs.set(subId, route);
    }
    const coreSubId = `desktop:${wcId}:${++this.nextCoreSub}:${subId}`;
    const supersededCoreSubId = route.candidate;
    if (supersededCoreSubId) this.pending.delete(supersededCoreSubId);
    route.candidate = coreSubId;
    return { podKey, wcId, subId, attemptId, coreSubId, supersededCoreSubId };
  }

  commit(attempt: SubscriptionAttempt): SubscriptionCommit {
    const route = this.route(attempt);
    if (route?.candidate !== attempt.coreSubId) return { committed: false };
    const replaced = route.active;
    route.active = attempt.coreSubId;
    route.candidate = undefined;
    this.scheduleFlush();
    return { committed: true, replaced };
  }

  rollback(attempt: SubscriptionAttempt): boolean {
    this.pending.delete(attempt.coreSubId);
    const byWc = this.byPod.get(attempt.podKey);
    const subs = byWc?.get(attempt.wcId);
    const route = subs?.get(attempt.subId);
    if (!byWc || !subs || route?.candidate !== attempt.coreSubId) return false;
    route.candidate = undefined;
    if (!route.active) {
      subs.delete(attempt.subId);
      if (subs.size === 0) byWc.delete(attempt.wcId);
      if (byWc.size === 0) this.byPod.delete(attempt.podKey);
    }
    return true;
  }

  take(wcId: number, podKey: string, subId: string): RemovedSubscriptions | undefined {
    const byWc = this.byPod.get(podKey);
    const subs = byWc?.get(wcId);
    const route = subs?.get(subId);
    if (!byWc || !subs || !route) return undefined;
    const coreSubIds = [route.active, route.candidate].filter((id): id is string => !!id);
    subs.delete(subId);
    for (const id of coreSubIds) this.pending.delete(id);
    if (subs.size === 0) byWc.delete(wcId);
    if (byWc.size === 0) this.byPod.delete(podKey);
    return { podKey, coreSubIds, podUnused: byWc.size === 0 };
  }

  takePod(wcId: number, podKey: string): RemovedSubscriptions | undefined {
    const byWc = this.byPod.get(podKey);
    const subs = byWc?.get(wcId);
    if (!byWc || !subs) return undefined;
    const coreSubIds = [...subs.values()].flatMap((route) =>
      [route.active, route.candidate].filter((id): id is string => !!id),
    );
    for (const id of coreSubIds) this.pending.delete(id);
    byWc.delete(wcId);
    if (byWc.size === 0) this.byPod.delete(podKey);
    return { podKey, coreSubIds, podUnused: byWc.size === 0 };
  }

  takeWindow(wcId: number): RemovedSubscriptions[] {
    const removed: RemovedSubscriptions[] = [];
    for (const podKey of [...this.byPod.keys()]) {
      const item = this.takePod(wcId, podKey);
      if (item) removed.push(item);
    }
    return removed;
  }

  hasPod(podKey: string): boolean {
    return this.byPod.has(podKey);
  }

  sendToPod<C extends RelayPushChannel>(
    podKey: string,
    channel: C,
    payload: RelayPushPayload<C>,
  ): void {
    const byWc = this.byPod.get(podKey);
    if (!byWc) return;
    for (const wcId of byWc.keys()) this.sendTo(wcId, channel, payload);
  }

  onOutput(attempt: SubscriptionAttempt): (_error: unknown, bytes: number[]) => void {
    return (_error: unknown, bytes: number[]) => {
      const output = this.pending.get(attempt.coreSubId);
      if (output) {
        for (const byte of bytes) output.bytes.push(byte);
      } else {
        this.pending.set(attempt.coreSubId, { ...attempt, bytes: [...bytes] });
      }
      this.scheduleFlush();
    };
  }

  clear(): void {
    this.byPod.clear();
    this.pending.clear();
  }

  private scheduleFlush(): void {
    if (this.flushScheduled) return;
    this.flushScheduled = true;
    setImmediate(() => {
      this.flushScheduled = false;
      for (const [id, output] of this.pending) {
        const route = this.route(output);
        if (route?.active === id) {
          const payload: RelayOutputPayload = {
            podKey: output.podKey,
            subId: output.subId,
            attemptId: output.attemptId,
            data: Uint8Array.from(output.bytes),
          };
          this.sendTo(output.wcId, "relay:output", payload);
          this.pending.delete(id);
        } else if (route?.candidate !== id) {
          this.pending.delete(id);
        }
      }
    });
  }

  private route(ref: SubscriptionAttempt): SubscriptionRoute | undefined {
    return this.byPod.get(ref.podKey)?.get(ref.wcId)?.get(ref.subId);
  }

  private sendTo<C extends RelayPushChannel>(
    wcId: number,
    channel: C,
    payload: RelayPushPayload<C>,
  ): void {
    this.registry.sendTo(wcId, channel, payload);
  }
}
