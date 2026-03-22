import { useEffect } from "react";
import { relayPool } from "@/stores/relayConnection";
import { dispatchAcpRelayEvent } from "@/stores/acpEventDispatcher";

/**
 * Subscribe to ACP relay messages for a pod.
 * Manages the relay subscription lifecycle (subscribe/unsubscribe)
 * and routes incoming ACP events to the session store.
 */
export function useAcpRelay(podKey: string, paneId: string, active: boolean): void {
  useEffect(() => {
    if (!active) return;

    const subscriptionId = `acp-${paneId}`;

    // Subscribe to the relay connection (shares the existing WebSocket).
    // The callback is a no-op because terminal output is irrelevant for ACP panels.
    relayPool.subscribe(podKey, subscriptionId, () => {});

    // Listen for ACP-specific messages and route to store.
    const unsubAcp = relayPool.onAcpMessage(podKey, (msgType, payload) => {
      dispatchAcpRelayEvent(podKey, msgType, payload);
    });

    return () => {
      relayPool.unsubscribe(podKey, subscriptionId);
      unsubAcp();
    };
  }, [podKey, paneId, active]);
}
