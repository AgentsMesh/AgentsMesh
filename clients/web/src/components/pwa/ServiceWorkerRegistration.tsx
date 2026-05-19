"use client";

import { useEffect, useState, useCallback } from "react";
import { toast } from "sonner";

interface ServiceWorkerRegistrationProps {
  onRegistered?: (registration: ServiceWorkerRegistration) => void;
  onUpdateAvailable?: (registration: ServiceWorkerRegistration) => void;
}

export function ServiceWorkerRegistration({
  onRegistered,
  onUpdateAvailable,
}: ServiceWorkerRegistrationProps) {
  const [registration, setRegistration] = useState<ServiceWorkerRegistration | null>(null);
  const [, setUpdateAvailable] = useState(false);

  const handleUpdate = useCallback(() => {
    if (registration?.waiting) {
      registration.waiting.postMessage({ type: "SKIP_WAITING" });
      window.location.reload();
    }
  }, [registration]);

  useEffect(() => {
    if (typeof window === "undefined" || !("serviceWorker" in navigator)) {
      return;
    }

    // Dev mode: don't register the SW, and unregister any stale copy from a
    // previous session. A stale SW can serve 503 "Offline" responses when the
    // dev-server URL doesn't match its caches. Also skip when explicitly
    // requested via ?nosw=1 for debugging.
    const isDev = process.env.NODE_ENV !== "production";
    const nosw =
      typeof window !== "undefined" &&
      new URLSearchParams(window.location.search).get("nosw") !== null;

    if (isDev || nosw) {
      navigator.serviceWorker
        .getRegistrations()
        .then((regs) => regs.forEach((r) => r.unregister()))
        .catch(() => undefined);
      if ("caches" in window) {
        caches.keys().then((keys) => keys.forEach((k) => caches.delete(k))).catch(() => undefined);
      }
      return;
    }

    const registerSW = async () => {
      try {
        const reg = await navigator.serviceWorker.register("/sw.js", {
          scope: "/",
        });

        setRegistration(reg);
        onRegistered?.(reg);

        reg.addEventListener("updatefound", () => {
          const newWorker = reg.installing;
          if (!newWorker) return;

          newWorker.addEventListener("statechange", () => {
            if (newWorker.state === "installed" && navigator.serviceWorker.controller) {
              setUpdateAvailable(true);
              onUpdateAvailable?.(reg);
              toast.info("New version available", {
                description: "Click to update the application",
                action: {
                  label: "Update",
                  onClick: handleUpdate,
                },
                duration: 10000,
              });
            }
          });
        });

        navigator.serviceWorker.addEventListener("controllerchange", () => {
        });

        console.log("[PWA] Service Worker registered successfully");
      } catch (error) {
        console.error("[PWA] Service Worker registration failed:", error);
      }
    };

    registerSW();

    return () => {
    };
  }, [onRegistered, onUpdateAvailable, handleUpdate]);

  // Periodically check for updates (every 60 seconds)
  useEffect(() => {
    if (!registration) return;

    const interval = setInterval(() => {
      registration.update();
    }, 60 * 1000);

    return () => clearInterval(interval);
  }, [registration]);

  return null;
}

export function useServiceWorker() {
  const [registration, setRegistration] = useState<ServiceWorkerRegistration | null>(null);

  const isSupported = typeof window !== "undefined" && "serviceWorker" in navigator;

  useEffect(() => {
    if (!isSupported) return;

    let mounted = true;
    navigator.serviceWorker.ready.then((reg) => {
      if (mounted) setRegistration(reg);
    });

    return () => {
      mounted = false;
    };
  }, [isSupported]);

  return { registration, isSupported };
}

export default ServiceWorkerRegistration;
