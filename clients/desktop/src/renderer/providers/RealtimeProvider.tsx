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
import {
  handlePodEvent, handleChannelEvent, handleInfraEvent,
  handleAutopilotEvent, handleLoopEvent,
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
  // Throttle reconnect fetch burst: WebSocket flap would otherwise re-trigger
  // every fetch per cycle; resulting setState storm linked to React #185.
  const lastReconnectFetchRef = useRef(0);

  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      if (event.type.startsWith("pod:")) { handlePodEvent(event); return; }
      if (event.type.startsWith("channel:")) { handleChannelEvent(event); return; }
      if (event.type.startsWith("autopilot:")) { handleAutopilotEvent(event); return; }
      if (event.type.startsWith("loop_run:")) {
        handleLoopEvent(event, loopDebounceRef, t, (title, desc) => {
          toast.warning(title, { description: desc, duration: 8000 });
        });
        return;
      }
      if (event.type.startsWith("runner:") || event.type.startsWith("ticket:") ||
          event.type.startsWith("mr:") || event.type.startsWith("pipeline:")) {
        handleInfraEvent(event);
        return;
      }
      onEvent?.(event);
    },
    [onEvent, t]
  );

  useAllEventsSubscription(handleEvent, [handleEvent]);

  useEffect(() => {
    const ref = loopDebounceRef;
    return () => { if (ref.current) clearTimeout(ref.current); };
  }, []);

  useEffect(() => {
    if (connectionState !== "connected") return;
    const now = Date.now();
    if (now - lastReconnectFetchRef.current < 1500) return;
    lastReconnectFetchRef.current = now;
    usePodStore.getState().fetchSidebarPods?.(usePodStore.getState().currentSidebarFilter);
    useRunnerStore.getState().fetchRunners?.();
    useTicketStore.getState().fetchTickets?.();
    useMeshStore.getState().fetchTopology?.();
    useAutopilotStore.getState().fetchAutopilotControllers?.();
    useLoopStore.getState().fetchLoops?.();
    useChannelMessageStore.getState().fetchUnreadCounts?.();
  }, [connectionState]);

  const value = useMemo<RealtimeContextValue>(
    () => ({ connectionState, reconnect }),
    [connectionState, reconnect]
  );

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}
