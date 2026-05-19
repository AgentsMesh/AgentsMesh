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

export function useRealtimeConnection() {
  const [connectionState, setConnectionState] =
    useState<ConnectionState>("disconnected");
  const currentOrg = useCurrentOrg();
  const user = useCurrentUser();
  const managerRef = useRef(getEventSubscriptionManager());

  // deps use `user?.id` (not `user`) — useCurrentUser returns a fresh object on every
  // store tick, so reference-deps would close()'d the WS mid-handshake on login.
  useEffect(() => {
    if (!currentOrg || !user) {
      managerRef.current.disconnect();
      return;
    }

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
  }, [currentOrg?.id, user?.id]);

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
