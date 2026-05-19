"use client";

import { useCallback, useEffect, useState, useSyncExternalStore } from "react";

export interface BrowserNotificationOptions {
  title: string;
  body?: string;
  icon?: string;
  tag?: string;
  data?: Record<string, unknown>;
  onClick?: () => void;
}

interface UseBrowserNotificationReturn {
  permission: NotificationPermission | "unsupported";
  isSupported: boolean;
  isPWA: boolean;
  requestPermission: () => Promise<boolean>;
  showNotification: (options: BrowserNotificationOptions) => Promise<boolean>;
}

function getIsPWA(): boolean {
  if (typeof window === "undefined") return false;
  return (
    window.matchMedia("(display-mode: standalone)").matches ||
    // @ts-expect-error - iOS Safari specific
    window.navigator.standalone === true
  );
}

function getServiceWorkerSupported(): boolean {
  if (typeof window === "undefined") return false;
  return "serviceWorker" in navigator && "PushManager" in window;
}

function getIsSupported(): boolean {
  if (typeof window === "undefined") return false;
  return "Notification" in window || getServiceWorkerSupported();
}

function getPermission(): NotificationPermission | "unsupported" {
  if (typeof window === "undefined") return "unsupported";

  if ("Notification" in window) {
    return Notification.permission;
  }

  if (getServiceWorkerSupported()) {
    return "default";
  }

  return "unsupported";
}

function getServerSnapshot(): NotificationPermission | "unsupported" {
  return "default";
}

function subscribeToPermissionChanges(callback: () => void): () => void {
  if (typeof document === "undefined") return () => {};
  document.addEventListener("visibilitychange", callback);
  return () => document.removeEventListener("visibilitychange", callback);
}

async function getServiceWorkerRegistration(): Promise<ServiceWorkerRegistration | null> {
  if (!getServiceWorkerSupported()) return null;

  try {
    const registration = await navigator.serviceWorker.ready;
    return registration;
  } catch {
    return null;
  }
}

async function showNotificationViaSW(
  registration: ServiceWorkerRegistration,
  options: BrowserNotificationOptions
): Promise<boolean> {
  try {
    await registration.showNotification(options.title, {
      body: options.body,
      icon: options.icon || "/icons/icon.svg",
      badge: "/icons/icon.svg",
      tag: options.tag || `notification-${Date.now()}`,
      data: {
        ...options.data,
        url: window.location.href,
      },
      silent: false,
    });

    console.log("[BrowserNotification] Shown via Service Worker");
    return true;
  } catch (error) {
    console.error("[BrowserNotification] SW notification failed:", error);
    return false;
  }
}

function showNotificationDirect(options: BrowserNotificationOptions): boolean {
  try {
    const notification = new Notification(options.title, {
      body: options.body,
      icon: options.icon || "/icons/icon.svg",
      tag: options.tag,
      data: options.data,
      requireInteraction: false,
    });

    if (options.onClick) {
      notification.onclick = (event) => {
        event.preventDefault();
        window.focus();
        options.onClick?.();
        notification.close();
      };
    }

    setTimeout(() => {
      notification.close();
    }, 5000);

    console.log("[BrowserNotification] Shown via Notification API");
    return true;
  } catch (error) {
    console.error("[BrowserNotification] Direct notification failed:", error);
    return false;
  }
}

export function useBrowserNotification(): UseBrowserNotificationReturn {
  const permission = useSyncExternalStore(
    subscribeToPermissionChanges,
    getPermission,
    getServerSnapshot
  );

  const [isSupported] = useState(() => getIsSupported());
  const [isPWA] = useState(() => getIsPWA());
  const [swRegistration, setSwRegistration] = useState<ServiceWorkerRegistration | null>(null);

  useEffect(() => {
    getServiceWorkerRegistration().then(setSwRegistration);
  }, []);

  const requestPermission = useCallback(async (): Promise<boolean> => {
    if (!isSupported) {
      console.warn("[BrowserNotification] Notifications not supported");
      return false;
    }

    if ("Notification" in window && Notification.permission === "granted") {
      return true;
    }

    try {
      if ("Notification" in window) {
        const result = await Notification.requestPermission();
        return result === "granted";
      }

      console.warn("[BrowserNotification] Cannot request permission without Notification API");
      return false;
    } catch (error) {
      console.error("[BrowserNotification] Failed to request permission:", error);
      return false;
    }
  }, [isSupported]);

  const showNotification = useCallback(
    async (options: BrowserNotificationOptions): Promise<boolean> => {
      if (!isSupported) {
        console.warn("[BrowserNotification] Notifications not supported");
        return false;
      }

      const currentPermission = getPermission();
      if (currentPermission !== "granted") {
        console.warn("[BrowserNotification] Permission not granted:", currentPermission);
        return false;
      }

      if (swRegistration) {
        const swResult = await showNotificationViaSW(swRegistration, options);
        if (swResult) return true;
      }

      if ("Notification" in window) {
        return showNotificationDirect(options);
      }

      return false;
    },
    [isSupported, swRegistration]
  );

  return {
    permission,
    isSupported,
    isPWA,
    requestPermission,
    showNotification,
  };
}
