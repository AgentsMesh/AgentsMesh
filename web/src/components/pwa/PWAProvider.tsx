"use client";

import { useEffect, useState } from "react";
import { ServiceWorkerRegistration } from "./ServiceWorkerRegistration";
import { PushNotificationManager } from "./PushNotificationManager";

interface PWAProviderProps {
  children: React.ReactNode;
}

export function PWAProvider({ children }: PWAProviderProps) {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

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
