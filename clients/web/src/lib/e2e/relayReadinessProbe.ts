import { getRelayManager } from "@agentsmesh/service-runtime";

export interface RelayReadinessProbe {
  readonly managerConstructorName: string;
  beginPendingSubscribe(relayUrl: string): void;
  cancelPendingSubscribe(): Promise<never>;
}

declare global {
  interface Window {
    __agentsmeshE2ERelayReadiness?: RelayReadinessProbe;
  }
}

type PendingSubscribe = {
  podKey: string;
  subscriptionId: string;
  phase: "pending" | "cancelling";
  outcome: Promise<
    | { kind: "resolved" }
    | { kind: "rejected"; error: unknown }
  >;
};

const DRIVER_TEARDOWN_TIMEOUT_MS = 5_000;

async function waitForDriverTeardown(
  manager: ReturnType<typeof getRelayManager>,
  podKey: string,
): Promise<void> {
  const deadline = performance.now() + DRIVER_TEARDOWN_TIMEOUT_MS;
  while (await manager.get_status(podKey) !== "disconnected") {
    if (performance.now() >= deadline) {
      throw new Error(`relay driver did not tear down for ${podKey}`);
    }
    // Yield a browser task so the Rust executor can consume Disconnect and
    // remove the driver handle. Readiness is still governed by get_status.
    await new Promise<void>((resolve) => setTimeout(resolve, 0));
  }
}

export function installRelayReadinessProbe(target: Window): void {
  const manager = getRelayManager();
  let pending: PendingSubscribe | undefined;
  target.__agentsmeshE2ERelayReadiness = {
    managerConstructorName: manager.constructor.name,
    beginPendingSubscribe(relayUrl: string): void {
      if (pending) {
        throw new DOMException("relay readiness probe already has pending work", "InvalidStateError");
      }
      const nonce = `${Date.now()}-${Math.random()}`;
      const podKey = `e2e-wasm-readiness-${nonce}`;
      const subscriptionId = `cancel-${nonce}`;
      const outcome = manager
        .subscribe(
          podKey,
          subscriptionId,
          relayUrl,
          "e2e-invalid-token",
          () => undefined,
        )
        .then(
          () => ({ kind: "resolved" as const }),
          (error: unknown) => ({ kind: "rejected" as const, error }),
        );
      pending = { podKey, subscriptionId, phase: "pending", outcome };
    },

    async cancelPendingSubscribe(): Promise<never> {
      const current = pending;
      if (!current || current.phase !== "pending") {
        throw new DOMException("relay readiness probe has no pending subscribe", "InvalidStateError");
      }
      current.phase = "cancelling";
      try {
        await manager.unsubscribe(current.podKey, current.subscriptionId);
        const settled = await current.outcome;
        if (settled.kind === "resolved") {
          throw new Error("relay readiness unexpectedly resolved without a baseline");
        }
        throw settled.error;
      } finally {
        pending = undefined;
        await manager.disconnect(current.podKey);
        await waitForDriverTeardown(manager, current.podKey);
      }
    },
  };
}
