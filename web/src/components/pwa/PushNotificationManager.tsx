"use client";

import { useEffect, useState, useCallback } from "react";
import { create } from "zustand";
import { persist } from "zustand/middleware";

// VAPID public key would come from server
// This is a placeholder - in production, fetch from API
const VAPID_PUBLIC_KEY = process.env.NEXT_PUBLIC_VAPID_PUBLIC_KEY || "";

interface NotificationPreferences {
  podStatus: boolean;
  ticketAssigned: boolean;
  ticketUpdated: boolean;
  runnerOffline: boolean;
}

interface PushNotificationState {
  permission: NotificationPermission | "default";
  subscription: PushSubscription | null;
  preferences: NotificationPreferences;
  isSupported: boolean;
  isLoading: boolean;
  error: string | null;
  setPermission: (permission: NotificationPermission) => void;
  setSubscription: (subscription: PushSubscription | null) => void;
  setPreferences: (preferences: Partial<NotificationPreferences>) => void;
  setIsSupported: (isSupported: boolean) => void;
  setIsLoading: (isLoading: boolean) => void;
  setError: (error: string | null) => void;
}

export const usePushNotificationStore = create<PushNotificationState>()(
  persist(
    (set) => ({
      permission: "default",
      subscription: null,
      preferences: {
        podStatus: true,
        ticketAssigned: true,
        ticketUpdated: true,
        runnerOffline: true,
      },
      isSupported: false,
      isLoading: false,
      error: null,
      setPermission: (permission) => set({ permission }),
      setSubscription: (subscription) => set({ subscription }),
      setPreferences: (preferences) =>
        set((state) => ({
          preferences: { ...state.preferences, ...preferences },
        })),
      setIsSupported: (isSupported) => set({ isSupported }),
      setIsLoading: (isLoading) => set({ isLoading }),
      setError: (error) => set({ error }),
    }),
    {
      name: "agentsmesh-push-notifications",
      partialize: (state) => ({
        preferences: state.preferences,
      }),
    }
  )
);

// Convert base64 to Uint8Array for VAPID key
function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const rawData = window.atob(base64);
  const buffer = new ArrayBuffer(rawData.length);
  const outputArray = new Uint8Array(buffer);
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

export function usePushNotifications() {
  const permission = usePushNotificationStore((s) => s.permission);
  const subscription = usePushNotificationStore((s) => s.subscription);
  const preferences = usePushNotificationStore((s) => s.preferences);
  const isSupported = usePushNotificationStore((s) => s.isSupported);
  const isLoading = usePushNotificationStore((s) => s.isLoading);
  const error = usePushNotificationStore((s) => s.error);
  const setPermission = usePushNotificationStore((s) => s.setPermission);
  const setSubscription = usePushNotificationStore((s) => s.setSubscription);
  const setIsSupported = usePushNotificationStore((s) => s.setIsSupported);
  const setIsLoading = usePushNotificationStore((s) => s.setIsLoading);
  const setError = usePushNotificationStore((s) => s.setError);
  const setPreferences = usePushNotificationStore((s) => s.setPreferences);

  // Check support on mount
  useEffect(() => {
    const checkSupport = () => {
      const supported =
        typeof window !== "undefined" &&
        "serviceWorker" in navigator &&
        "PushManager" in window &&
        "Notification" in window;
      setIsSupported(supported);

      if (supported) {
        setPermission(Notification.permission);
      }
    };

    checkSupport();
  }, [setIsSupported, setPermission]);

  // Request permission
  const requestPermission = useCallback(async (): Promise<boolean> => {
    if (!isSupported) {
      setError("Push notifications are not supported");
      return false;
    }

    setIsLoading(true);
    setError(null);

    try {
      const result = await Notification.requestPermission();
      setPermission(result);

      if (result !== "granted") {
        setError("Permission denied");
        return false;
      }

      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to request permission");
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [isSupported, setIsLoading, setError, setPermission]);

  // Subscribe to push notifications
  const subscribe = useCallback(async (): Promise<PushSubscription | null> => {
    if (!isSupported || permission !== "granted") {
      return null;
    }

    if (!VAPID_PUBLIC_KEY) {
      console.warn("[Push] VAPID public key not configured");
      return null;
    }

    setIsLoading(true);
    setError(null);

    try {
      const registration = await navigator.serviceWorker.ready;

      // Check for existing subscription
      const existingSubscription = await registration.pushManager.getSubscription();

      if (existingSubscription) {
        setSubscription(existingSubscription);
        return existingSubscription;
      }

      // Create new subscription
      const newSubscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(VAPID_PUBLIC_KEY),
      });

      setSubscription(newSubscription);

      // Send subscription to server
      await sendSubscriptionToServer(newSubscription);

      return newSubscription;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to subscribe");
      return null;
    } finally {
      setIsLoading(false);
    }
  }, [isSupported, permission, setIsLoading, setError, setSubscription]);

  // Unsubscribe from push notifications
  const unsubscribe = useCallback(async (): Promise<boolean> => {
    if (!subscription) return true;

    setIsLoading(true);
    setError(null);

    try {
      await subscription.unsubscribe();
      setSubscription(null);

      // Notify server of unsubscription
      await removeSubscriptionFromServer(subscription);

      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to unsubscribe");
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [subscription, setIsLoading, setError, setSubscription]);

  // Update preferences on server
  const updatePreferences = useCallback(
    async (newPreferences: Partial<NotificationPreferences>) => {
      setPreferences(newPreferences);

      if (subscription) {
        await updatePreferencesOnServer(subscription, {
          ...preferences,
          ...newPreferences,
        });
      }
    },
    [preferences, subscription, setPreferences]
  );

  return {
    permission,
    subscription,
    preferences,
    isSupported,
    isLoading,
    error,
    requestPermission,
    subscribe,
    unsubscribe,
    updatePreferences,
  };
}

// API helpers
async function sendSubscriptionToServer(subscription: PushSubscription) {
  try {
    await fetch("/api/push/subscribe", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(subscription),
    });
  } catch (error) {
    console.error("[Push] Failed to send subscription to server:", error);
  }
}

async function removeSubscriptionFromServer(subscription: PushSubscription) {
  try {
    await fetch("/api/push/unsubscribe", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ endpoint: subscription.endpoint }),
    });
  } catch (error) {
    console.error("[Push] Failed to remove subscription from server:", error);
  }
}

async function updatePreferencesOnServer(
  subscription: PushSubscription,
  preferences: NotificationPreferences
) {
  try {
    await fetch("/api/push/preferences", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        endpoint: subscription.endpoint,
        preferences,
      }),
    });
  } catch (error) {
    console.error("[Push] Failed to update preferences on server:", error);
  }
}

// Component wrapper
interface PushNotificationManagerProps {
  autoSubscribe?: boolean;
  children?: React.ReactNode;
}

export function PushNotificationManager({
  autoSubscribe = false,
  children,
}: PushNotificationManagerProps) {
  const { isSupported, permission, subscribe } =
    usePushNotifications();
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (initialized || !isSupported) return;

    const init = async () => {
      setInitialized(true);

      if (!autoSubscribe) return;

      // Check existing subscription
      const registration = await navigator.serviceWorker.ready;
      const existingSubscription = await registration.pushManager.getSubscription();

      if (existingSubscription) {
        return;
      }

      // Only auto-subscribe if permission already granted
      if (permission === "granted") {
        await subscribe();
      }
    };

    init();
  }, [initialized, isSupported, autoSubscribe, permission, subscribe]);

  return <>{children}</>;
}

export default PushNotificationManager;
