"use client";

import React, { createContext, useContext, useEffect, useCallback, useRef, useMemo } from "react";
import { useRealtimeConnection, useAllEventsSubscription } from "@/hooks/useRealtimeEvents";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { reconnectRegistry } from "@/lib/realtime";
import {
  handlePodEvent, handleChannelEvent, handleInfraEvent,
  handleAutopilotEvent, handleLoopEvent, handleBlockstoreEvent,
  type DebounceRef,
} from "./realtimeEventHandlers";
import type { ConnectionState, RealtimeEvent } from "@/lib/realtime";
import {
  getApiClient,
} from "@/lib/wasm-core";
import { usePodStore } from "@/stores/pod";
import { useTicketStore } from "@/stores/ticket";

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

/// Drain Rust-side pending side-effect queues. Rust SSOT dispatch
/// populates these synchronously during event handling; the tick
/// increment that follows is the React signal that something landed in
/// the queues. Called from a tick-bound useEffect below.
///
/// Returns a list of side-effects to perform on the JS side. We don't
/// run them inside the drain helper because i18n / sonner / browser
/// Notification API live in the React layer.
interface PendingSideEffects {
  toasts: Array<{ kind: string; title_key: string; title_params: unknown; description: string; duration_ms: number }>;
  notifications: Array<{ title: string; body: string; icon?: string; link?: string }>;
  refetchTicketSlugs: string[];
  refetchPodKeys: string[];
}

function drainPendingSideEffects(): PendingSideEffects {
  const client = getApiClient();
  if (!client) return { toasts: [], notifications: [], refetchTicketSlugs: [], refetchPodKeys: [] };
  // Each take_* call is atomic on the Rust side — second call returns
  // empty. Reading once per tick is the React-friendly pattern.
  try {
    return {
      toasts: JSON.parse(client.take_pending_toasts()),
      notifications: JSON.parse(client.take_pending_browser_notifications()),
      refetchTicketSlugs: JSON.parse(client.take_pending_refetch_ticket_slugs()),
      refetchPodKeys: JSON.parse(client.take_pending_refetch_pod_keys()),
    };
  } catch {
    return { toasts: [], notifications: [], refetchTicketSlugs: [], refetchPodKeys: [] };
  }
}

export function RealtimeProvider({ children, onEvent }: RealtimeProviderProps) {
  const { connectionState, reconnect } = useRealtimeConnection();
  const t = useTranslations();
  const loopDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const ticketDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const channelDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const podSidebarDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const tickPollRef = useRef<number>(0);

  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      // Rust SSOT dispatch already updated AppState by the time this
      // closure runs (the dispatch hook fires before external handlers).
      // The JS handlers below are kept for two reasons:
      //   1. Workspace pane lifecycle (removePaneByPodKey on pod:terminated)
      //   2. Sidebar refresh / org-scoped refetch on member events
      // Pure state writes (e.g. patchChannelMemberCount) are no-ops now
      // — Rust dispatch already applied them — but harmless to retry.
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
      if (event.type.startsWith("blockstore:")) { handleBlockstoreEvent(event); return; }
      onEvent?.(event);
    },
    [onEvent, t]
  );

  useAllEventsSubscription(handleEvent, [handleEvent]);

  // Drain Rust-side pending queues after every dispatched event. The
  // tick counter increments AFTER AppState mutation, so a positive
  // delta means there may be new pending items to consume.
  useAllEventsSubscription(
    useCallback(() => {
      const pending = drainPendingSideEffects();
      // Toasts (i18n key resolved here — Rust doesn't carry locale).
      for (const spec of pending.toasts) {
        const fn = spec.kind === "warning" ? toast.warning
          : spec.kind === "error" ? toast.error
          : spec.kind === "success" ? toast.success
          : toast.info;
        const params = (spec.title_params as Record<string, unknown>) ?? {};
        fn(t(spec.title_key as Parameters<typeof t>[0], params), {
          description: spec.description,
          duration: spec.duration_ms > 0 ? spec.duration_ms : undefined,
        });
      }
      // Browser notifications — left to platform onEvent (DashboardShell)
      // since it owns the permission state + router for click-through.
      for (const n of pending.notifications) {
        if (typeof window !== "undefined" && "Notification" in window) {
          if (Notification.permission === "granted") {
            new Notification(n.title, { body: n.body, icon: n.icon, tag: n.link ?? n.title });
          }
        }
      }
      // Refetches for indirect events (mr:* / pipeline:* / sparse pod:*).
      for (const slug of pending.refetchTicketSlugs) {
        void useTicketStore.getState().fetchTicket?.(slug);
      }
      for (const key of pending.refetchPodKeys) {
        void usePodStore.getState().fetchPod?.(key);
      }
    }, [t]),
    []
  );

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
