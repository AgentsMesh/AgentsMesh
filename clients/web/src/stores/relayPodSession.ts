export type RelayResizeRequest = Readonly<{
  cols: number;
  rows: number;
  force: boolean;
}>;

export type RelaySubscriptionAttempt = Readonly<{
  subscriptionId: string;
  generation: number;
}>;

type AttemptState = {
  token: RelaySubscriptionAttempt;
  ready: boolean;
  managerStarted: boolean;
};

type PendingResize = {
  generation: number;
  request: RelayResizeRequest;
};

// Owns the state whose lifetime follows a pod's relay subscriptions. Transport
// readiness deliberately survives an empty subscription set because the Rust
// pool keeps the connection alive for a reconnect grace period. Resize state
// does not: it belongs to the generation that observed the terminal geometry.
export class RelayPodSession {
  private attempts = new Map<string, AttemptState>();
  private generation = 0;
  private pendingResize: PendingResize | undefined;
  private dataReady = false;

  constructor(private readonly podKey: string) {}

  beginSubscription(subscriptionId: string): RelaySubscriptionAttempt {
    if (this.attempts.has(subscriptionId)) {
      throw new DOMException(
        `Relay subscription already exists: ${this.podKey}/${subscriptionId}`,
        "InvalidStateError",
      );
    }
    if (this.attempts.size === 0) {
      this.generation += 1;
      this.pendingResize = undefined;
    }
    const token = {
      subscriptionId,
      generation: this.generation,
    };
    this.attempts.set(subscriptionId, {
      token,
      ready: false,
      managerStarted: false,
    });
    return token;
  }

  ownsSubscription(token: RelaySubscriptionAttempt): boolean {
    return this.stateFor(token) !== undefined;
  }

  markManagerStarted(token: RelaySubscriptionAttempt): boolean {
    const state = this.stateFor(token);
    if (!state) return false;
    state.managerStarted = true;
    return true;
  }

  markSubscriptionReady(token: RelaySubscriptionAttempt): boolean {
    const state = this.stateFor(token);
    if (!state) return false;
    state.ready = true;
    return true;
  }

  removeSubscription(
    subscriptionId: string,
    expected?: RelaySubscriptionAttempt,
  ): boolean {
    const state = this.attempts.get(subscriptionId);
    if (!state || (expected && state.token !== expected)) return false;
    this.attempts.delete(subscriptionId);
    if (this.attempts.size === 0) this.pendingResize = undefined;
    return true;
  }

  hasStartedConnection(): boolean {
    return Array.from(this.attempts.values()).some((state) => state.managerStarted);
  }

  setDataReady(ready: boolean): void {
    this.dataReady = ready;
  }

  // The pod-scoped callback carries no generation. Every attempt still visible
  // here may belong to work that raced the old driver's lock-free callback, so
  // preserve it and revoke only the retired transport's data readiness.
  handlePodDisconnected(): boolean {
    this.dataReady = false;
    if (this.attempts.size > 0) return true;

    this.pendingResize = undefined;
    this.generation += 1;
    return false;
  }

  scheduleResize(resize: RelayResizeRequest): RelayResizeRequest | undefined {
    if (this.attempts.size === 0) return undefined;
    if (this.canDispatchResize()) return resize;

    const previous = this.pendingResize;
    this.pendingResize = {
      generation: this.generation,
      request: {
        ...resize,
        force:
          resize.force ||
          (previous?.generation === this.generation && previous.request.force),
      },
    };
    return undefined;
  }

  takeFlushableResize(): RelayResizeRequest | undefined {
    if (!this.pendingResize || !this.canDispatchResize()) return undefined;
    const pending = this.pendingResize;
    this.pendingResize = undefined;
    return pending.generation === this.generation ? pending.request : undefined;
  }

  clear(): void {
    this.attempts.clear();
    this.pendingResize = undefined;
    this.dataReady = false;
    this.generation += 1;
  }

  private canDispatchResize(): boolean {
    return (
      this.dataReady &&
      this.attempts.size > 0 &&
      Array.from(this.attempts.values()).every((state) => state.ready)
    );
  }

  private stateFor(token: RelaySubscriptionAttempt): AttemptState | undefined {
    if (token.generation !== this.generation) return undefined;
    const state = this.attempts.get(token.subscriptionId);
    return state?.token === token ? state : undefined;
  }
}
