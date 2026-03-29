import { useEffect } from "react";
import { relayPool } from "@/stores/relayConnection";

/**
 * Subscribe to ACP relay messages for a pod.
 * Manages the relay subscription lifecycle (subscribe/unsubscribe).
 * ACP events are dispatched directly to the store in the WebSocket handler,
 * so no listener registration is needed here.
 */
export function useAcpRelay(podKey: string, paneId: string, active: boolean): void {
  useEffect(() => {
    if (!active) return;

    const subscriptionId = `acp-${paneId}`;

    // Subscribe to the relay connection (shares the existing WebSocket).
    // The callback is a no-op because terminal output is irrelevant for ACP panels.
    relayPool.subscribe(podKey, subscriptionId, () => {});

    return () => {
      relayPool.unsubscribe(podKey, subscriptionId);
    };
  }, [podKey, paneId, active]);
}
