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
import { getWsBaseUrl } from "@/lib/env";

function buildEventsWsUrl(orgSlug: string, token: string): string {
  return `${getWsBaseUrl()}/api/v1/orgs/${orgSlug}/ws/events?token=${token}`;
}

/**
 * Manages the realtime events WebSocket connection.
 * Should be used once at the app root level (in RealtimeProvider).
 */
export function useRealtimeConnection() {
  const [connectionState, setConnectionState] =
    useState<ConnectionState>("disconnected");
  const { currentOrg, token } = useAuthStore();
  const managerRef = useRef(getEventSubscriptionManager());

  useEffect(() => {
    if (!currentOrg || !token) {
      managerRef.current.disconnect();
      return;
    }

    resetEventSubscriptionManager();
    const manager = getEventSubscriptionManager();
    managerRef.current = manager;
    manager.connect(() => {
      const { currentOrg: o, token: t } = useAuthStore.getState();
      return o && t ? buildEventsWsUrl(o.slug, t) : "";
    });

    const unsubscribe = manager.onConnectionStateChange(setConnectionState);

    return () => {
      unsubscribe();
      // Delay disconnect to avoid killing connection during React Strict Mode re-mount.
      const currentManager = manager;
      setTimeout(() => {
        if (managerRef.current === currentManager) {
          currentManager.disconnect();
        }
      }, 100);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentOrg?.id, token]);

  const reconnect = useCallback(() => {
    const { currentOrg: org, token: t } = useAuthStore.getState();
    if (!org || !t) return;
    resetEventSubscriptionManager();
    managerRef.current = getEventSubscriptionManager();
    managerRef.current.connect(() => {
      const { currentOrg: o, token: tk } = useAuthStore.getState();
      return o && tk ? buildEventsWsUrl(o.slug, tk) : "";
    });
  }, []);

  return {
    connectionState,
    reconnect,
  };
}

/**
 * Subscribe to a specific event type.
 */
export function useEventSubscription<T = unknown>(
  eventType: EventType,
  handler: EventHandler<T>,
  deps: React.DependencyList = []
) {
  const handlerRef = useRef(handler);

  useEffect(() => {
    handlerRef.current = handler;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [handler, ...deps]);

  useEffect(() => {
    const wrappedHandler: EventHandler<T> = (event) => {
      handlerRef.current(event);
    };

    let unsubscribe = getEventSubscriptionManager().subscribe(eventType, wrappedHandler);

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
 * Subscribe to all events.
 */
export function useAllEventsSubscription(
  handler: EventHandler,
  deps: React.DependencyList = []
) {
  const handlerRef = useRef(handler);

  useEffect(() => {
    handlerRef.current = handler;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [handler, ...deps]);

  useEffect(() => {
    const wrappedHandler: EventHandler = (event) => {
      handlerRef.current(event);
    };

    let unsubscribe = getEventSubscriptionManager().subscribeAll(wrappedHandler);

    const unsubscribeReset = onManagerReset((newManager) => {
      unsubscribe();
      unsubscribe = newManager.subscribeAll(wrappedHandler);
    });

    return () => {
      unsubscribe();
      unsubscribeReset();
    };
  }, []);
}

/**
 * Get the latest event of a specific type as React state.
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
