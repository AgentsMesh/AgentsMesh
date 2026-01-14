"use client";

import { useSyncExternalStore } from "react";
import { ServiceWorkerRegistration } from "./ServiceWorkerRegistration";
import { PushNotificationManager } from "./PushNotificationManager";

interface PWAProviderProps {
  children: React.ReactNode;
}

// Detect client-side mount using useSyncExternalStore
function subscribe() {
  return () => {};
}
function getSnapshot() {
  return true;
}
function getServerSnapshot() {
  return false;
}
function useIsMounted() {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

export function PWAProvider({ children }: PWAProviderProps) {
  const mounted = useIsMounted();

  // Don't render PWA components during SSR
  if (!mounted) {
    return <>{children}</>;
  }

  return (
    <>
      <ServiceWorkerRegistration />
      <PushNotificationManager autoSubscribe={false}>
        {children}
      </PushNotificationManager>
    </>
  );
}

export default PWAProvider;
