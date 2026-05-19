"use client";

import { useSyncExternalStore } from "react";
import { readLightSession, type LightSession } from "@/lib/light-session";

const subscribe = (cb: () => void) => {
  const handler = (e: StorageEvent) => {
    if (e.key && e.key.startsWith("agentsmesh-auth/")) cb();
  };
  window.addEventListener("storage", handler);
  return () => window.removeEventListener("storage", handler);
};

let cachedSession: LightSession | null = null;
let cachedKey = "";

const getSnapshot = (): LightSession | null => {
  const next = readLightSession();
  const nextKey = next ? `${next.expiresAt}:${next.currentOrgSlug}` : "";
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
