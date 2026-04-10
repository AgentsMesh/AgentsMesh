"use client";

import React, { createContext, useContext, useEffect, useCallback, useRef, useMemo } from "react";
import { useRealtimeConnection, useAllEventsSubscription } from "@/hooks/useRealtimeEvents";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { reconnectRegistry } from "@/lib/realtime";
import {
  handlePodEvent, handleChannelEvent, handleInfraEvent,
  handleAutopilotEvent, handleLoopEvent,
  type DebounceRef,
} from "./realtimeEventHandlers";
import type { ConnectionState, RealtimeEvent } from "@/lib/realtime";

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
  onEvent?: (event: RealtimeEvent) => void;
}

export function RealtimeProvider({ children, onEvent }: RealtimeProviderProps) {
  const { connectionState, reconnect } = useRealtimeConnection();
  const t = useTranslations();
  const loopDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const ticketDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const channelDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const podSidebarDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      if (event.type.startsWith("pod:")) { handlePodEvent(event, podSidebarDebounceRef); return; }
      if (event.type.startsWith("channel:")) { handleChannelEvent(event, channelDebounceRef); return; }
      if (event.type.startsWith("autopilot:")) { handleAutopilotEvent(event); return; }
      if (event.type.startsWith("loop_run:")) {
        handleLoopEvent(event, loopDebounceRef, t, (title, desc) => {
          toast.warning(title, { description: desc, duration: 8000 });
        });
        return;
      }
      if (event.type.startsWith("runner:") || event.type.startsWith("ticket:") ||
          event.type.startsWith("mr:") || event.type.startsWith("pipeline:")) {
        handleInfraEvent(event, ticketDebounceRef);
        return;
      }
      onEvent?.(event);
    },
    [onEvent, t]
  );

  useAllEventsSubscription(handleEvent, [handleEvent]);

  useEffect(() => {
    const refs: DebounceRef[] = [loopDebounceRef, ticketDebounceRef, channelDebounceRef, podSidebarDebounceRef];
    return () => { refs.forEach((r) => { if (r.current) clearTimeout(r.current); }); };
  }, []);

  useEffect(() => {
    if (connectionState !== "connected") return;
    const cancel = reconnectRegistry.execute();
    return cancel;
  }, [connectionState]);

  const value = useMemo<RealtimeContextValue>(() => ({ connectionState, reconnect }), [connectionState, reconnect]);

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}
