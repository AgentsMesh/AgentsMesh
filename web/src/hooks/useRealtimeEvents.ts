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
import { useCurrentUser, useCurrentOrg, readCurrentOrg } from "@/stores/auth";
import { getAuthManager } from "@/lib/wasm-core";
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
  const currentOrg = useCurrentOrg();
  const user = useCurrentUser();
  const managerRef = useRef(getEventSubscriptionManager());

  // Connect and subscribe to state changes when org/user are available
  useEffect(() => {
    if (!currentOrg || !user) {
      // disconnect() will trigger onConnectionStateChange callback
      managerRef.current.disconnect();
      return;
    }

    // Reset and reconnect when org or user changes
    resetEventSubscriptionManager();
    const manager = getEventSubscriptionManager();
    managerRef.current = manager;
    manager.connect(() => {
      const o = readCurrentOrg();
      const t = getAuthManager().get_token?.();
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
  }, [currentOrg?.id, user]);

  const reconnect = useCallback(() => {
    const org = readCurrentOrg();
    const t = getAuthManager().get_token?.();
    if (!org || !t) return;
    resetEventSubscriptionManager();
    managerRef.current = getEventSubscriptionManager();
    managerRef.current.connect(() => {
      const o = readCurrentOrg();
      const tk = getAuthManager().get_token?.();
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
