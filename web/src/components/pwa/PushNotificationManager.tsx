"use client";

import { useEffect, useState, useCallback } from "react";
import {
  usePushNotificationStore,
  VAPID_PUBLIC_KEY,
  urlBase64ToUint8Array,
  sendSubscriptionToServer,
  removeSubscriptionFromServer,
  updatePreferencesOnServer,
} from "./push-notification-store";
import type { NotificationPreferences } from "./push-notification-store";

export { usePushNotificationStore } from "./push-notification-store";

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

  useEffect(() => {
    const supported = typeof window !== "undefined" && "serviceWorker" in navigator
      && "PushManager" in window && "Notification" in window;
    setIsSupported(supported);
    if (supported) setPermission(Notification.permission);
  }, [setIsSupported, setPermission]);

  const requestPermission = useCallback(async (): Promise<boolean> => {
    if (!isSupported) { setError("Push notifications are not supported"); return false; }
    setIsLoading(true);
    setError(null);
    try {
      const result = await Notification.requestPermission();
      setPermission(result);
      if (result !== "granted") { setError("Permission denied"); return false; }
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to request permission");
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [isSupported, setIsLoading, setError, setPermission]);

  const subscribe = useCallback(async (): Promise<PushSubscription | null> => {
    if (!isSupported || permission !== "granted") return null;
    if (!VAPID_PUBLIC_KEY) { console.warn("[Push] VAPID public key not configured"); return null; }
    setIsLoading(true);
    setError(null);
    try {
      const registration = await navigator.serviceWorker.ready;
      const existing = await registration.pushManager.getSubscription();
      if (existing) { setSubscription(existing); return existing; }
      const newSub = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(VAPID_PUBLIC_KEY),
      });
      setSubscription(newSub);
      await sendSubscriptionToServer(newSub);
      return newSub;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to subscribe");
      return null;
    } finally {
      setIsLoading(false);
    }
  }, [isSupported, permission, setIsLoading, setError, setSubscription]);

  const unsubscribe = useCallback(async (): Promise<boolean> => {
    if (!subscription) return true;
    setIsLoading(true);
    setError(null);
    try {
      await subscription.unsubscribe();
      setSubscription(null);
      await removeSubscriptionFromServer(subscription);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to unsubscribe");
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [subscription, setIsLoading, setError, setSubscription]);

  const updatePreferences = useCallback(
    async (newPreferences: Partial<NotificationPreferences>) => {
      setPreferences(newPreferences);
      if (subscription) {
        await updatePreferencesOnServer(subscription, { ...preferences, ...newPreferences });
      }
    },
    [preferences, subscription, setPreferences]
  );

  return { permission, subscription, preferences, isSupported, isLoading, error,
    requestPermission, subscribe, unsubscribe, updatePreferences };
}

interface PushNotificationManagerProps {
  autoSubscribe?: boolean;
  children?: React.ReactNode;
}

export function PushNotificationManager({ autoSubscribe = false, children }: PushNotificationManagerProps) {
  const { isSupported, permission, subscribe } = usePushNotifications();
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (initialized || !isSupported) return;
    const init = async () => {
      setInitialized(true);
      if (!autoSubscribe) return;
      const registration = await navigator.serviceWorker.ready;
      const existing = await registration.pushManager.getSubscription();
      if (existing) return;
      if (permission === "granted") await subscribe();
    };
    init();
  }, [initialized, isSupported, autoSubscribe, permission, subscribe]);

  return <>{children}</>;
}

export default PushNotificationManager;
