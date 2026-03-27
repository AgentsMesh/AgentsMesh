import { EventSubscriptionManager } from "./EventSubscriptionManager";

// Singleton instance
let instance: EventSubscriptionManager | null = null;

// Listeners that get notified when the manager is reset
type ManagerResetListener = (newManager: EventSubscriptionManager) => void;
const managerResetListeners: Set<ManagerResetListener> = new Set();

/**
 * Get the singleton EventSubscriptionManager instance
 */
export function getEventSubscriptionManager(): EventSubscriptionManager {
  if (!instance) {
    instance = new EventSubscriptionManager({
      onConnectionStateChange: (state) => {
        console.log(`[EventSubscriptionManager] Connection state: ${state}`);
      },
    });
  }
  return instance;
}

/**
 * Reset the singleton instance (for testing or org switching)
 */
export function resetEventSubscriptionManager(): void {
  if (instance) {
    instance.disconnect();
    instance = null;
  }
  // Create new instance and notify listeners
  const newManager = getEventSubscriptionManager();
  managerResetListeners.forEach((listener) => {
    try {
      listener(newManager);
    } catch (error) {
      console.error("[EventSubscriptionManager] Reset listener error:", error);
    }
  });
}

/**
 * Subscribe to manager reset events
 * This is called when the manager is reset (e.g., on org switch)
 * Subscribers should re-register their event handlers with the new manager
 * @returns Unsubscribe function
 */
export function onManagerReset(listener: ManagerResetListener): () => void {
  managerResetListeners.add(listener);
  return () => {
    managerResetListeners.delete(listener);
  };
}
