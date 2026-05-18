"use client";

import { useSyncExternalStore } from "react";
import { readLightSession, type LightSession } from "@/lib/light-session";

// React hook for marketing pages — reads PersistedSession from localStorage
// after mount (avoids SSR flash) and listens for cross-tab logout via the
// `storage` event so a logout in (auth) reflects on the open tab.
//
// Both `session` and `hydrated` route through useSyncExternalStore so React
// uses getServerSnapshot during SSR and swaps to the client snapshot only
// after hydration commit — preventing hydration mismatch when callers gate
// rendering on `hydrated`.

const subscribe = (cb: () => void) => {
  const handler = (e: StorageEvent) => {
    if (e.key && e.key.startsWith("agentsmesh-auth/")) cb();
  };
  window.addEventListener("storage", handler);
  return () => window.removeEventListener("storage", handler);
};

// Cache the last snapshot — useSyncExternalStore requires referential
// stability between calls when underlying state hasn't changed, otherwise
// React bails out with "getSnapshot returned a different value every time".
let cachedSession: LightSession | null = null;
let cachedKey = "";

const getSnapshot = (): LightSession | null => {
  const next = readLightSession();
  const nextKey = next ? `${next.userId}:${next.expiresAt}:${next.currentOrgSlug}` : "";
  if (nextKey !== cachedKey) {
    cachedSession = next;
    cachedKey = nextKey;
  }
  return cachedSession;
};

const getServerSnapshot = (): LightSession | null => null;

const noopSubscribe = () => () => {};
const getHydratedClient = () => true;
const getHydratedServer = () => false;

export function useLightSession(): { session: LightSession | null; hydrated: boolean } {
  const session = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const hydrated = useSyncExternalStore(noopSubscribe, getHydratedClient, getHydratedServer);
  return { session, hydrated };
}
