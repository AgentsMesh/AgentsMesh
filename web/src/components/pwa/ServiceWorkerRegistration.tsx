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
  const [updateAvailable, setUpdateAvailable] = useState(false);

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

    const registerSW = async () => {
      try {
        const reg = await navigator.serviceWorker.register("/sw.js", {
          scope: "/",
        });

        setRegistration(reg);
        onRegistered?.(reg);

        // Check for updates
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

        // Handle controller change (when skipWaiting is called)
        navigator.serviceWorker.addEventListener("controllerchange", () => {
          // Optionally reload the page
        });

        console.log("[PWA] Service Worker registered successfully");
      } catch (error) {
        console.error("[PWA] Service Worker registration failed:", error);
      }
    };

    registerSW();

    // Cleanup
    return () => {
      // No cleanup needed for service worker
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

// Hook to get service worker registration
export function useServiceWorker() {
  const [registration, setRegistration] = useState<ServiceWorkerRegistration | null>(null);

  // Derive isSupported from environment - no state needed
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
