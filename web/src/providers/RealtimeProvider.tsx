"use client";

import React, { createContext, useContext, useEffect, useCallback, useRef, useMemo } from "react";
import { useRealtimeConnection, useAllEventsSubscription } from "@/hooks/useRealtimeEvents";
import { usePodStore } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { useTicketStore } from "@/stores/ticket";
import { useMeshStore } from "@/stores/mesh";
import { useChannelMessageStore } from "@/stores/channel";
import { useAutopilotStore } from "@/stores/autopilot";
import { useLoopStore } from "@/stores/loop";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import {
  handlePodEvent, handleChannelEvent, handleInfraEvent,
  handleAutopilotEvent, handleLoopEvent,
} from "./realtimeEventHandlers";
import type { ConnectionState, RealtimeEvent, TerminalNotificationData, TaskCompletedData, NotificationPayloadData } from "@/lib/realtime";

interface RealtimeContextValue {
  connectionState: ConnectionState;
  reconnect: () => void;
}

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

export function useRealtime() {
  const context = useContext(RealtimeContext);
  if (!context) throw new Error("useRealtime must be used within RealtimeProvider");
  return context;
}

interface RealtimeProviderProps {
  children: React.ReactNode;
  onTerminalNotification?: (data: TerminalNotificationData) => void;
  onTaskCompleted?: (data: TaskCompletedData) => void;
  onBrowserNotification?: (data: { title: string; body: string; link?: string }) => void;
}

/**
 * RealtimeProvider manages the WebSocket connection and routes events to stores.
 * Event handling logic is in realtimeEventHandlers.ts for SRP.
 */
export function RealtimeProvider({
  children, onTerminalNotification, onTaskCompleted, onBrowserNotification,
}: RealtimeProviderProps) {
  const { connectionState, reconnect } = useRealtimeConnection();
  const t = useTranslations();
  const router = useRouter();
  const loopDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      // Pod events
      if (event.type.startsWith("pod:")) {
        handlePodEvent(event);
        return;
      }

      // Channel events
      if (event.type.startsWith("channel:")) {
        handleChannelEvent(event);
        return;
      }

      // Autopilot events
      if (event.type.startsWith("autopilot:")) {
        handleAutopilotEvent(event);
        return;
      }

      // Loop events
      if (event.type.startsWith("loop_run:")) {
        handleLoopEvent(event, loopDebounceRef, t, (title, desc) => {
          toast.warning(title, { description: desc, duration: 8000 });
        });
        return;
      }

      // Infrastructure events (runner, ticket, MR, pipeline)
      if (event.type.startsWith("runner:") || event.type.startsWith("ticket:") ||
          event.type.startsWith("mr:") || event.type.startsWith("pipeline:")) {
        handleInfraEvent(event);
        return;
      }

      // Notification events
      switch (event.type as string) {
        case "terminal:notification":
        case "pod:notification":
          onTerminalNotification?.(event.data as TerminalNotificationData);
          break;
        case "task:completed":
          onTaskCompleted?.(event.data as TaskCompletedData);
          break;
        case "notification": {
          const data = event.data as NotificationPayloadData;
          if (data.channels?.toast) {
            const toastFn = data.priority === "high" ? toast.warning : toast.info;
            toastFn(data.title, {
              description: data.body, duration: data.priority === "high" ? 8000 : 4000,
              ...(data.link ? { action: { label: "→", onClick: () => router.push(data.link!) } } : {}),
            });
          }
          if (data.channels?.browser) {
            onBrowserNotification?.({ title: data.title, body: data.body, link: data.link });
          }
          break;
        }
      }
    },
    [onTerminalNotification, onTaskCompleted, onBrowserNotification, t, router]
  );

  useAllEventsSubscription(handleEvent, [handleEvent]);

  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => { if (loopDebounceRef.current) clearTimeout(loopDebounceRef.current); };
  }, []);

  // Refresh data when reconnected
  useEffect(() => {
    if (connectionState === "connected") {
      usePodStore.getState().fetchSidebarPods?.(usePodStore.getState().currentSidebarFilter);
      useRunnerStore.getState().fetchRunners?.();
      useTicketStore.getState().fetchTickets?.();
      useMeshStore.getState().fetchTopology?.();
      useAutopilotStore.getState().fetchAutopilotControllers?.();
      useLoopStore.getState().fetchLoops?.();
      useChannelMessageStore.getState().fetchUnreadCounts?.();
    }
  }, [connectionState]);

  const value = useMemo<RealtimeContextValue>(() => ({ connectionState, reconnect }), [connectionState, reconnect]);

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}
