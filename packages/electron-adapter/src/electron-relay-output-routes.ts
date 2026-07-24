import type { RelayOutputPayload, RelaySubscribeResult } from "./relay-ipc-contract";

export type RelayOutputCallback = (data: Uint8Array) => void;

interface OutputCandidate {
  attemptId: string;
  generation: number;
  callback: RelayOutputCallback;
  buffered: Uint8Array[];
  outputReached: Promise<void>;
  cancellation: Promise<void>;
  resolveOutputReached: () => void;
  resolveCancellation: () => void;
  outputSeen: boolean;
  cancelled: boolean;
}

interface OutputRoute {
  active?: OutputCandidate;
  attempts: Map<string, OutputCandidate>;
}

export interface RelayOutputAttempt {
  podKey: string;
  subId: string;
  attemptId: string;
  route: OutputRoute;
  candidate: OutputCandidate;
}

function outputCandidate(
  attemptId: string,
  generation: number,
  callback: RelayOutputCallback,
): OutputCandidate {
  let resolveOutputReached!: () => void;
  let resolveCancellation!: () => void;
  return {
    attemptId,
    generation,
    callback,
    buffered: [],
    outputReached: new Promise<void>((resolve) => { resolveOutputReached = resolve; }),
    cancellation: new Promise<void>((resolve) => { resolveCancellation = resolve; }),
    resolveOutputReached,
    resolveCancellation,
    outputSeen: false,
    cancelled: false,
  };
}

export class ElectronRelayOutputRoutes {
  private readonly byPod = new Map<string, Map<string, OutputRoute>>();
  private generation = 0;
  begin(
    podKey: string,
    subId: string,
    callback: RelayOutputCallback,
  ): RelayOutputAttempt {
    const routes = this.ensurePod(podKey);
    let route = routes.get(subId);
    if (!route) {
      route = { attempts: new Map() };
      routes.set(subId, route);
    }
    const generation = ++this.generation;
    const attemptId = `renderer:${generation}`;
    const candidate = outputCandidate(attemptId, generation, callback);
    route.attempts.set(attemptId, candidate);
    return { podKey, subId, attemptId, route, candidate };
  }
  async complete(
    attempt: RelayOutputAttempt,
    invocation: Promise<RelaySubscribeResult>,
    onPromote?: () => void,
  ): Promise<void> {
    const { podKey, subId, attemptId, route, candidate } = attempt;
    const observed = invocation.then(
      (committed) => ({ kind: "committed" as const, committed }),
      (error: unknown) => ({ kind: "failed" as const, error }),
    );
    const outcome = await Promise.race([
      observed,
      candidate.cancellation.then(() => ({ kind: "cancelled" as const })),
    ]);
    if (outcome.kind === "cancelled") {
      route.attempts.delete(attemptId);
      this.prune(podKey, subId, route);
      return;
    }
    if (outcome.kind === "failed") {
      route.attempts.delete(attemptId);
      this.cancelCandidate(candidate);
      this.prune(podKey, subId, route);
      throw outcome.error;
    }
    const stillOwned = this.byPod.get(podKey)?.get(subId) === route
      && route.attempts.get(attemptId) === candidate;
    if (outcome.committed === false || !stillOwned) {
      route.attempts.delete(attemptId);
      this.cancelCandidate(candidate);
      this.prune(podKey, subId, route);
      return;
    }
    this.promote(route, candidate);
    onPromote?.();
    // Main commit can precede its baseline IPC batch; wait for the marker.
    await Promise.race([candidate.outputReached, candidate.cancellation]);
  }
  receive({ podKey, subId, attemptId, data }: RelayOutputPayload): void {
    const route = this.byPod.get(podKey)?.get(subId);
    if (route?.active?.attemptId === attemptId) {
      this.record(route.active, data, true);
      return;
    }
    const candidate = route?.attempts.get(attemptId);
    if (candidate) this.record(candidate, data, false);
  }
  hasPod(podKey: string): boolean { return this.byPod.has(podKey); }
  hasActivePod(podKey: string): boolean {
    return [...(this.byPod.get(podKey)?.values() ?? [])].some((route) => !!route.active);
  }
  remove(podKey: string, subId: string): (() => void) | undefined {
    const routes = this.byPod.get(podKey);
    const route = routes?.get(subId);
    if (routes) {
      routes.delete(subId);
      if (routes.size === 0) this.byPod.delete(podKey);
    }
    if (!route) return undefined;
    this.cancelRoute(route);
    const { active } = route;
    if (!active) return undefined;
    return () => {
      if (!this.byPod.get(podKey)?.has(subId)) {
        this.ensurePod(podKey).set(subId, { active, attempts: new Map() });
      }
    };
  }

  dropPod(podKey: string): void {
    const routes = this.byPod.get(podKey);
    if (!routes) return;
    this.byPod.delete(podKey);
    for (const route of routes.values()) this.cancelRoute(route);
  }
  dropAll(): void { for (const podKey of [...this.byPod.keys()]) this.dropPod(podKey); }

  private promote(route: OutputRoute, candidate: OutputCandidate): void {
    route.attempts.delete(candidate.attemptId);
    for (const [attemptId, other] of route.attempts) {
      if (other.generation < candidate.generation) {
        route.attempts.delete(attemptId);
        this.cancelCandidate(other);
      }
    }
    const previous = route.active;
    route.active = candidate;
    if (previous && previous !== candidate) this.cancelCandidate(previous);
    for (const data of candidate.buffered) candidate.callback(data);
    candidate.buffered.length = 0;
  }
  private record(candidate: OutputCandidate, data: Uint8Array, active: boolean): void {
    if (!candidate.outputSeen) {
      candidate.outputSeen = true;
      candidate.resolveOutputReached();
    }
    // An empty baseline is an ordering marker, not terminal data.
    if (data.byteLength === 0) return;
    if (active) candidate.callback(data);
    else candidate.buffered.push(data);
  }
  private cancelCandidate(candidate: OutputCandidate): void {
    if (candidate.cancelled) return;
    candidate.cancelled = true;
    candidate.resolveCancellation();
    candidate.resolveOutputReached();
  }
  private cancelRoute(route: OutputRoute): void {
    if (route.active) this.cancelCandidate(route.active);
    for (const candidate of route.attempts.values()) this.cancelCandidate(candidate);
    route.attempts.clear();
  }
  private ensurePod(podKey: string): Map<string, OutputRoute> {
    let routes = this.byPod.get(podKey);
    if (!routes) {
      routes = new Map();
      this.byPod.set(podKey, routes);
    }
    return routes;
  }
  private prune(podKey: string, subId: string, route: OutputRoute): void {
    if (route.active || route.attempts.size > 0) return;
    const routes = this.byPod.get(podKey);
    if (routes?.get(subId) !== route) return;
    routes.delete(subId);
    if (routes.size === 0) this.byPod.delete(podKey);
  }
}
