"use client";

import { useEffect, useCallback, useRef, useState } from "react";
import {
  getEventSubscriptionManager,
  resetEventSubscriptionManager,
  onManagerReset,
  type EventType,
  type EventHandler,
  type RealtimeEvent,
  type ConnectionState,
} from "@/lib/realtime";
import { useAuthStore } from "@/stores/auth";

/**
 * Hook to manage the realtime events connection
 *
 * Should be used once at the app root level (in RealtimeProvider)
 */
export function useRealtimeConnection() {
  const [connectionState, setConnectionState] =
    useState<ConnectionState>("disconnected");
  const { currentOrg, token } = useAuthStore();
  const managerRef = useRef(getEventSubscriptionManager());

  // Connect and subscribe to state changes when org/token are available
  // Combined into single useEffect to avoid race conditions between
  // manager creation and state subscription
  useEffect(() => {
    if (!currentOrg || !token) {
      // disconnect() will trigger onConnectionStateChange callback
      managerRef.current.disconnect();
      return;
    }

    // Reset and reconnect when org or token changes
    resetEventSubscriptionManager();
    const manager = getEventSubscriptionManager();
    managerRef.current = manager;
    manager.connect();

    // Subscribe to connection state changes (same manager instance)
    const unsubscribe = manager.onConnectionStateChange(setConnectionState);

    return () => {
      unsubscribe();
      // In React Strict Mode, cleanup runs immediately after mount.
      // Only disconnect if we're actually unmounting (not just strict mode re-render).
      // The manager will handle reconnection if needed.
      // Using setTimeout to delay disconnect slightly allows re-mount to cancel it.
      const currentManager = manager;
      setTimeout(() => {
        // Check if this manager is still the active one
        if (managerRef.current === currentManager) {
          currentManager.disconnect();
        }
      }, 100);
    };
  }, [currentOrg?.id, token]);

  const reconnect = useCallback(() => {
    resetEventSubscriptionManager();
    managerRef.current = getEventSubscriptionManager();
    managerRef.current.connect();
  }, []);

  return {
    connectionState,
    reconnect,
  };
}

/**
 * Hook to subscribe to a specific event type
 *
 * @param eventType - The event type to subscribe to
 * @param handler - The handler function to call when the event is received
 * @param deps - Dependencies array for the handler (similar to useEffect)
 */
export function useEventSubscription<T = unknown>(
  eventType: EventType,
  handler: EventHandler<T>,
  deps: React.DependencyList = []
) {
  const handlerRef = useRef(handler);

  // Update handler ref when handler changes
  useEffect(() => {
    handlerRef.current = handler;
  }, [handler, ...deps]);

  useEffect(() => {
    const wrappedHandler: EventHandler<T> = (event) => {
      handlerRef.current(event);
    };

    // Subscribe to current manager
    let unsubscribe = getEventSubscriptionManager().subscribe(eventType, wrappedHandler);

    // Re-subscribe when manager is reset
    const unsubscribeReset = onManagerReset((newManager) => {
      unsubscribe();
      unsubscribe = newManager.subscribe(eventType, wrappedHandler);
    });

    return () => {
      unsubscribe();
      unsubscribeReset();
    };
  }, [eventType]);
}

/**
 * Hook to subscribe to all events
 *
 * @param handler - The handler function to call when any event is received
 * @param deps - Dependencies array for the handler
 */
export function useAllEventsSubscription(
  handler: EventHandler,
  deps: React.DependencyList = []
) {
  const handlerRef = useRef(handler);

  useEffect(() => {
    handlerRef.current = handler;
  }, [handler, ...deps]);

  useEffect(() => {
    const wrappedHandler: EventHandler = (event) => {
      handlerRef.current(event);
    };

    // Subscribe to current manager
    let unsubscribe = getEventSubscriptionManager().subscribeAll(wrappedHandler);

    // Re-subscribe when manager is reset (via callback, not polling!)
    const unsubscribeReset = onManagerReset((newManager) => {
      unsubscribe();
      unsubscribe = newManager.subscribeAll(wrappedHandler);
    });

    return () => {
      unsubscribe();
      unsubscribeReset();
    };
  }, []); // No dependencies - handler updates via ref
}

/**
 * Hook to get the latest event of a specific type
 *
 * @param eventType - The event type to watch
 * @returns The latest event or null
 */
export function useLatestEvent<T = unknown>(
  eventType: EventType
): RealtimeEvent<T> | null {
  const [latestEvent, setLatestEvent] = useState<RealtimeEvent<T> | null>(null);

  useEventSubscription<T>(
    eventType,
    (event) => {
      setLatestEvent(event as RealtimeEvent<T>);
    },
    []
  );

  return latestEvent;
}
