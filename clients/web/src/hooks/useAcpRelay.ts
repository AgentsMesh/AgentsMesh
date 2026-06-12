import { useEffect } from "react";
import { relayPool } from "@/stores/relayConnection";
import { dispatchAcpRelayEvent } from "@/stores/acpEventDispatcher";
import { dispatchLoopalRelayEvent } from "@/stores/loopalDispatcher";
import { isResourceNotFound, isPodNotConnectable } from "@/lib/errors/serviceError";

export function useAcpRelay(podKey: string, paneId: string, active: boolean): void {
  useEffect(() => {
    if (!active) return;

    const subscriptionId = `acp-${paneId}`;

    // Subscribe to share the WebSocket; terminal output is irrelevant for ACP.
    // subscribe() is async — handle its rejection (mirrors useTerminalConnection)
    // so a connection-setup failure never escapes as an unhandled rejection.
    // not-found / not-yet-connectable are benign lifecycle transients (the
    // `active` dep re-runs this effect when pod status changes); only surface a
    // genuine connection failure.
    relayPool.subscribe(podKey, subscriptionId, () => {}).catch((error: unknown) => {
      if (isResourceNotFound(error) || isPodNotConnectable(error)) return;
      console.error("ACP relay subscribe failed:", error);
    });

    const unsubAcp = relayPool.onAcpMessage(podKey, (msgType, payload) => {
      if (dispatchLoopalRelayEvent(podKey, msgType, payload)) return;
      dispatchAcpRelayEvent(podKey, msgType, payload);
    });

    return () => {
      relayPool.unsubscribe(podKey, subscriptionId);
      unsubAcp();
    };
  }, [podKey, paneId, active]);
}
