import { useEffect } from "react";
import { relayPool } from "@/stores/relayConnection";
import { dispatchAcpRelayEvent } from "@/stores/acpEventDispatcher";

export function useAcpRelay(podKey: string, paneId: string, active: boolean): void {
  useEffect(() => {
    if (!active) return;

    const subscriptionId = `acp-${paneId}`;

    // Subscribe to share the WebSocket; terminal output is irrelevant for ACP.
    relayPool.subscribe(podKey, subscriptionId, () => {});

    const unsubAcp = relayPool.onAcpMessage(podKey, (msgType, payload) => {
      dispatchAcpRelayEvent(podKey, msgType, payload);
    });

    return () => {
      relayPool.unsubscribe(podKey, subscriptionId);
      unsubAcp();
    };
  }, [podKey, paneId, active]);
}
