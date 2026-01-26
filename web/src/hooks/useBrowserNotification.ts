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

// Check if running as PWA (standalone mode)
function getIsPWA(): boolean {
  if (typeof window === "undefined") return false;
  return (
    window.matchMedia("(display-mode: standalone)").matches ||
    // @ts-expect-error - iOS Safari specific
    window.navigator.standalone === true
  );
}

// Check if Service Worker is supported and can show notifications
function getServiceWorkerSupported(): boolean {
  if (typeof window === "undefined") return false;
  return "serviceWorker" in navigator && "PushManager" in window;
}

// Check if Notification API is supported (directly or via SW)
function getIsSupported(): boolean {
  if (typeof window === "undefined") return false;
  // Support either direct Notification API or Service Worker notifications
  return "Notification" in window || getServiceWorkerSupported();
}

// Get current permission status
function getPermission(): NotificationPermission | "unsupported" {
  if (typeof window === "undefined") return "unsupported";

  // Check Notification API permission
  if ("Notification" in window) {
    return Notification.permission;
  }

  // For iOS PWA without Notification in window, check via SW
  if (getServiceWorkerSupported()) {
    // On iOS PWA, permission is managed through the PWA settings
    // We can't directly check, so return "default" to prompt user action
    return "default";
  }

  return "unsupported";
}

// Server-side snapshot
function getServerSnapshot(): NotificationPermission | "unsupported" {
  return "default";
}

// Subscribe to permission changes (via visibility change as a proxy)
function subscribeToPermissionChanges(callback: () => void): () => void {
  if (typeof document === "undefined") return () => {};
  document.addEventListener("visibilitychange", callback);
  return () => document.removeEventListener("visibilitychange", callback);
}

// Get Service Worker registration
async function getServiceWorkerRegistration(): Promise<ServiceWorkerRegistration | null> {
  if (!getServiceWorkerSupported()) return null;

  try {
    // Wait for service worker to be ready
    const registration = await navigator.serviceWorker.ready;
    return registration;
  } catch {
    return null;
  }
}

// Show notification via Service Worker (for iOS PWA support)
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
        // Store click handler info for SW to process
        url: window.location.href,
      },
      // iOS PWA specific options
      silent: false,
    });

    console.log("[BrowserNotification] Shown via Service Worker");
    return true;
  } catch (error) {
    console.error("[BrowserNotification] SW notification failed:", error);
    return false;
  }
}

// Show notification directly via Notification API
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

    // Auto-close after 5 seconds
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

/**
 * Hook for browser native notifications with iOS PWA support
 *
 * Priority:
 * 1. Service Worker notifications (iOS PWA compatible)
 * 2. Direct Notification API (desktop/Android)
 *
 * iOS Requirements:
 * - iOS 16.4+
 * - Added to Home Screen (PWA mode)
 * - Notification permission granted
 */
export function useBrowserNotification(): UseBrowserNotificationReturn {
  const permission = useSyncExternalStore(
    subscribeToPermissionChanges,
    getPermission,
    getServerSnapshot
  );

  // Initialize state with lazy evaluation to avoid hydration mismatch
  const [isSupported] = useState(() => getIsSupported());
  const [isPWA] = useState(() => getIsPWA());
  const [swRegistration, setSwRegistration] = useState<ServiceWorkerRegistration | null>(null);

  // Get SW registration on mount (async operation is OK in effect)
  useEffect(() => {
    getServiceWorkerRegistration().then(setSwRegistration);
  }, []);

  // Request notification permission
  const requestPermission = useCallback(async (): Promise<boolean> => {
    if (!isSupported) {
      console.warn("[BrowserNotification] Notifications not supported");
      return false;
    }

    // If permission already granted
    if ("Notification" in window && Notification.permission === "granted") {
      return true;
    }

    try {
      // Request permission via Notification API if available
      if ("Notification" in window) {
        const result = await Notification.requestPermission();
        return result === "granted";
      }

      // iOS PWA without Notification API in window
      // Permission is managed via iOS Settings, we can't request it programmatically
      // Return false to indicate permission cannot be requested
      console.warn("[BrowserNotification] Cannot request permission without Notification API");
      return false;
    } catch (error) {
      console.error("[BrowserNotification] Failed to request permission:", error);
      return false;
    }
  }, [isSupported]);

  // Show a notification (auto-selects best method)
  const showNotification = useCallback(
    async (options: BrowserNotificationOptions): Promise<boolean> => {
      if (!isSupported) {
        console.warn("[BrowserNotification] Notifications not supported");
        return false;
      }

      // Check permission
      const currentPermission = getPermission();
      if (currentPermission !== "granted") {
        console.warn("[BrowserNotification] Permission not granted:", currentPermission);
        return false;
      }

      // Strategy: Try Service Worker first (better iOS PWA support), fallback to direct
      if (swRegistration) {
        const swResult = await showNotificationViaSW(swRegistration, options);
        if (swResult) return true;
      }

      // Fallback to direct Notification API
      // Note: If we reach here, Notification API must exist (otherwise getPermission
      // would have returned non-"granted" and we'd have exited at the permission check)
      if ("Notification" in window) {
        return showNotificationDirect(options);
      }

      // This path is technically unreachable: if Notification doesn't exist and
      // SW failed, getPermission() returns "default" or "unsupported", not "granted"
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
