import { EventSubscriptionManager } from "./EventSubscriptionManager";
import { logger } from "@/lib/logger";

let instance: EventSubscriptionManager | null = null;

type ManagerResetListener = (newManager: EventSubscriptionManager) => void;
const managerResetListeners: Set<ManagerResetListener> = new Set();

export function getEventSubscriptionManager(): EventSubscriptionManager {
  if (!instance) {
    instance = new EventSubscriptionManager({
      onConnectionStateChange: (state) => {
        // Route through logger (not console.log) so the realtime connection
        // timeline lands in the rolling log file alongside Rust events —
        // this is the cross-cutting signal we need when chasing #185-class
        // store/reconnect loops.
        logger.info("EventSubscriptionManager", `Connection state: ${state}`);
      },
    });
  }
  return instance;
}

export function resetEventSubscriptionManager(): void {
  if (instance) {
    instance.disconnect();
    instance = null;
  }
  const newManager = getEventSubscriptionManager();
  managerResetListeners.forEach((listener) => {
    try {
      listener(newManager);
    } catch (error) {
      console.error("[EventSubscriptionManager] Reset listener error:", error);
    }
  });
}

export function onManagerReset(listener: ManagerResetListener): () => void {
  managerResetListeners.add(listener);
  return () => {
    managerResetListeners.delete(listener);
  };
}
