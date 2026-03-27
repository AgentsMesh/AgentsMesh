"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface NotificationPreferences {
  podStatus: boolean;
  ticketAssigned: boolean;
  ticketUpdated: boolean;
  runnerOffline: boolean;
}

export interface PushNotificationState {
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
      preferences: { podStatus: true, ticketAssigned: true, ticketUpdated: true, runnerOffline: true },
      isSupported: false,
      isLoading: false,
      error: null,
      setPermission: (permission) => set({ permission }),
      setSubscription: (subscription) => set({ subscription }),
      setPreferences: (preferences) =>
        set((state) => ({ preferences: { ...state.preferences, ...preferences } })),
      setIsSupported: (isSupported) => set({ isSupported }),
      setIsLoading: (isLoading) => set({ isLoading }),
      setError: (error) => set({ error }),
    }),
    {
      name: "agentsmesh-push-notifications",
      partialize: (state) => ({ preferences: state.preferences }),
    }
  )
);

// VAPID public key from environment
export const VAPID_PUBLIC_KEY = process.env.NEXT_PUBLIC_VAPID_PUBLIC_KEY || "";

// Convert base64 to Uint8Array for VAPID key
export function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
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

// API helpers
export async function sendSubscriptionToServer(subscription: PushSubscription) {
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

export async function removeSubscriptionFromServer(subscription: PushSubscription) {
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

export async function updatePreferencesOnServer(
  subscription: PushSubscription,
  preferences: NotificationPreferences
) {
  try {
    await fetch("/api/push/preferences", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ endpoint: subscription.endpoint, preferences }),
    });
  } catch (error) {
    console.error("[Push] Failed to update preferences on server:", error);
  }
}
