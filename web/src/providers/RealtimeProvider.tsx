"use client";

import React, { createContext, useContext, useEffect, useCallback, useRef, useMemo } from "react";
import { useRealtimeConnection, useAllEventsSubscription } from "@/hooks/useRealtimeEvents";
import { usePodStore } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { useTicketStore } from "@/stores/ticket";
import { useMeshStore } from "@/stores/mesh";
import { useChannelStore } from "@/stores/channel";
import { useAutopilotStore } from "@/stores/autopilot";
import { useLoopStore } from "@/stores/loop";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import type { ConnectionState, RealtimeEvent, PodStatusChangedData, PodCreatedData, RunnerStatusData, TicketStatusChangedData, TerminalNotificationData, TaskCompletedData, PodTitleChangedData, PodInitProgressData, ChannelMessageData, AutopilotStatusChangedData, AutopilotIterationData, AutopilotCreatedData, AutopilotTerminatedData, AutopilotThinkingData, MREventData, PipelineEventData, LoopRunEventData, LoopRunWarningData } from "@/lib/realtime";

interface RealtimeContextValue {
  connectionState: ConnectionState;
  reconnect: () => void;
}

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

/**
 * Hook to access the realtime context
 */
export function useRealtime() {
  const context = useContext(RealtimeContext);
  if (!context) {
    throw new Error("useRealtime must be used within RealtimeProvider");
  }
  return context;
}

interface RealtimeProviderProps {
  children: React.ReactNode;
  /** Callback for terminal notifications (OSC 777) */
  onTerminalNotification?: (data: TerminalNotificationData) => void;
  /** Callback for task completed notifications */
  onTaskCompleted?: (data: TaskCompletedData) => void;
}

/**
 * RealtimeProvider manages the WebSocket connection and routes events to appropriate stores
 *
 * Should be placed at the dashboard layout level, after authentication is established
 */
export function RealtimeProvider({
  children,
  onTerminalNotification,
  onTaskCompleted,
}: RealtimeProviderProps) {
  const { connectionState, reconnect } = useRealtimeConnection();
  const t = useTranslations();

  // Debounce timer for loop events — rapid events (e.g. multiple runs completing
  // within seconds) are coalesced into a single API refresh cycle.
  const loopDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // NOTE: All store access uses getState() inside the event handler to avoid
  // subscribing RealtimeProvider to store state changes. Reactive subscriptions
  // would cause RealtimeProvider to re-render on every store update, creating
  // cascading re-renders that cause UI flicker (e.g., native <select> dropdowns
  // closing when the component tree re-renders).

  // Handle all events and route to appropriate stores
  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      switch (event.type) {
        // Pod events
        case "pod:created": {
          const data = event.data as PodCreatedData;
          // Fetch the individual pod to avoid resetting pagination state
          usePodStore.getState().fetchPod?.(data.pod_key);
          // Also refresh Mesh topology since a new pod affects the mesh
          useMeshStore.getState().fetchTopology?.();
          console.log("[Realtime] Pod created:", data.pod_key);
          break;
        }

        case "pod:status_changed": {
          const data = event.data as PodStatusChangedData;
          const podState = usePodStore.getState();
          // Check if pod exists in store - if not, fetch the individual pod
          const existingPod = podState.pods.find(p => p.pod_key === data.pod_key);
          if (!existingPod) {
            // Pod not in list, might be newly created - fetch individual pod
            podState.fetchPod?.(data.pod_key);
            console.log("[Realtime] Pod not found, fetching:", data.pod_key);
          } else if (podState.updatePodStatus) {
            // Update existing pod status (including error details if present)
            podState.updatePodStatus(
              data.pod_key,
              data.status as "running" | "initializing" | "failed" | "paused" | "terminated" | "error",
              data.agent_status,
              data.error_code,
              data.error_message
            );
          }
          // Also refresh Mesh topology since pod status affects the mesh
          useMeshStore.getState().fetchTopology?.();
          console.log("[Realtime] Pod status changed:", data.pod_key, data.status);
          break;
        }

        case "pod:agent_status_changed": {
          const data = event.data as PodStatusChangedData;
          if (data.agent_status) {
            usePodStore.getState().updateAgentStatus(data.pod_key, data.agent_status);
          }
          console.log("[Realtime] Pod agent status changed:", data.pod_key, data.agent_status);
          break;
        }

        case "pod:terminated": {
          const data = event.data as PodStatusChangedData;
          // Update pod status to terminated
          usePodStore.getState().updatePodStatus?.(data.pod_key, "terminated");
          // Also refresh Mesh topology since termination removes the pod from mesh
          useMeshStore.getState().fetchTopology?.();
          console.log("[Realtime] Pod terminated:", data.pod_key);
          break;
        }

        case "pod:title_changed": {
          const data = event.data as PodTitleChangedData;
          // Update pod title in podStore for sidebar display
          usePodStore.getState().updatePodTitle(data.pod_key, data.title);
          console.log("[Realtime] Pod title changed:", data.pod_key, data.title);
          break;
        }

        case "pod:init_progress": {
          const data = event.data as PodInitProgressData;
          // Update pod init progress in podStore
          usePodStore.getState().updatePodInitProgress(data.pod_key, data.phase, data.progress, data.message);
          console.log("[Realtime] Pod init progress:", data.pod_key, data.phase, data.progress);
          break;
        }

        // Runner events
        case "runner:online":
        case "runner:offline":
        case "runner:updated": {
          const data = event.data as RunnerStatusData;
          useRunnerStore.getState().updateRunnerStatus(
            data.runner_id,
            data.status as "online" | "offline" | "maintenance" | "busy"
          );
          console.log("[Realtime] Runner status:", data.runner_id, data.status);
          break;
        }

        // Ticket events
        case "ticket:created":
        case "ticket:updated":
        case "ticket:status_changed":
        case "ticket:moved":
        case "ticket:deleted": {
          const data = event.data as TicketStatusChangedData;
          // Refresh tickets list
          useTicketStore.getState().fetchTickets?.();
          console.log("[Realtime] Ticket event:", event.type, data.slug);
          break;
        }

        // Channel events
        case "channel:message": {
          const data = event.data as ChannelMessageData;
          const channelState = useChannelStore.getState();
          // Only add message if it belongs to the current channel
          const currentChannel = channelState.currentChannel;
          if (currentChannel && currentChannel.id === data.channel_id) {
            channelState.addMessage({
              id: data.id,
              channel_id: data.channel_id,
              sender_pod: data.sender_pod,
              sender_user_id: data.sender_user_id,
              message_type: data.message_type as "text" | "system" | "code" | "command",
              content: data.content,
              metadata: data.metadata,
              created_at: data.created_at,
            });
          }
          console.log("[Realtime] Channel message:", data.channel_id, data.id);
          break;
        }

        // Notification events
        case "terminal:notification": {
          const data = event.data as TerminalNotificationData;
          onTerminalNotification?.(data);
          console.log("[Realtime] Terminal notification:", data.title);
          break;
        }

        case "task:completed": {
          const data = event.data as TaskCompletedData;
          onTaskCompleted?.(data);
          console.log("[Realtime] Task completed:", data.pod_key, data.agent_status);
          break;
        }

        // AutopilotController events
        case "autopilot:status_changed": {
          const data = event.data as AutopilotStatusChangedData;
          useAutopilotStore.getState().updateAutopilotControllerStatus(
            data.autopilot_controller_key,
            data.phase,
            data.current_iteration,
            data.max_iterations,
            data.circuit_breaker_state,
            data.circuit_breaker_reason
          );
          console.log("[Realtime] Autopilot status changed:", data.autopilot_controller_key, data.phase);
          break;
        }

        case "autopilot:iteration": {
          const data = event.data as AutopilotIterationData;
          useAutopilotStore.getState().addIteration(data.autopilot_controller_key, {
            id: 0, // Will be assigned by server
            autopilot_controller_id: 0,
            iteration: data.iteration,
            phase: data.phase,
            summary: data.summary,
            files_changed: data.files_changed,
            duration_ms: data.duration_ms,
            created_at: new Date().toISOString(),
          });
          console.log("[Realtime] Autopilot iteration:", data.autopilot_controller_key, data.iteration);
          break;
        }

        case "autopilot:created": {
          const data = event.data as AutopilotCreatedData;
          // Refresh autopilot controllers list to include the new one
          useAutopilotStore.getState().fetchAutopilotControllers?.();
          console.log("[Realtime] Autopilot created:", data.autopilot_controller_key);
          break;
        }

        case "autopilot:terminated": {
          const data = event.data as AutopilotTerminatedData;
          useAutopilotStore.getState().removeAutopilotController(data.autopilot_controller_key);
          console.log("[Realtime] Autopilot terminated:", data.autopilot_controller_key, data.reason);
          break;
        }

        case "autopilot:thinking": {
          const data = event.data as AutopilotThinkingData;
          useAutopilotStore.getState().updateThinking(data.autopilot_controller_key, data);
          console.log("[Realtime] Autopilot thinking:", data.autopilot_controller_key, data.decision_type);
          break;
        }

        // MergeRequest events
        case "mr:created":
        case "mr:updated":
        case "mr:merged":
        case "mr:closed": {
          const data = event.data as MREventData;
          // Refresh tickets if this MR is associated with a ticket
          if (data.ticket_slug || data.ticket_id) {
            useTicketStore.getState().fetchTickets?.();
          }
          // Refresh pods if this MR is associated with a pod
          if (data.pod_id) {
            usePodStore.getState().fetchPods?.();
          }
          console.log("[Realtime] MR event:", event.type, data.mr_iid, data.state);
          break;
        }

        // Pipeline events
        case "pipeline:updated": {
          const data = event.data as PipelineEventData;
          // Refresh tickets if this pipeline is associated with a ticket
          if (data.ticket_slug || data.ticket_id) {
            useTicketStore.getState().fetchTickets?.();
          }
          // Refresh pods if this pipeline is associated with a pod
          if (data.pod_id) {
            usePodStore.getState().fetchPods?.();
          }
          console.log("[Realtime] Pipeline event:", data.pipeline_id, data.pipeline_status);
          break;
        }

        // Loop run events — debounced to coalesce rapid events into a single refresh
        case "loop_run:started":
        case "loop_run:completed":
        case "loop_run:failed": {
          const data = event.data as LoopRunEventData;
          // Clear any pending debounce timer and set a new one
          if (loopDebounceRef.current) {
            clearTimeout(loopDebounceRef.current);
          }
          loopDebounceRef.current = setTimeout(() => {
            loopDebounceRef.current = null;
            const currentLoopState = useLoopStore.getState();
            currentLoopState.fetchLoops?.();
            if (currentLoopState.currentLoop?.id === data.loop_id) {
              currentLoopState.fetchLoop?.(currentLoopState.currentLoop.slug);
              useLoopStore.setState({ runsOffset: 0 });
              currentLoopState.fetchRuns?.(currentLoopState.currentLoop.slug, { limit: 20, offset: 0 });
            }
          }, 500);
          console.log("[Realtime] Loop run event (debounced):", event.type, data.run_id, data.status);
          break;
        }

        // Loop run warning events (e.g., sandbox resume degradation)
        case "loop_run:warning": {
          const data = event.data as LoopRunWarningData;
          toast.warning(t("loops.runWarningTitle", { runNumber: data.run_number }), {
            description: data.detail || data.warning,
            duration: 8000,
          });
          console.log("[Realtime] Loop run warning:", data.warning, data.detail);
          break;
        }

        default:
          console.log("[Realtime] Unknown event:", event.type);
      }
    },
    // Only external callbacks and translation function are dependencies.
    // All stores are accessed via getState() to avoid reactive subscriptions.
    [onTerminalNotification, onTaskCompleted, t]
  );

  // Subscribe to all events
  useAllEventsSubscription(handleEvent, [handleEvent]);

  // Cleanup debounce timer on unmount to prevent stale state updates
  useEffect(() => {
    return () => {
      if (loopDebounceRef.current) {
        clearTimeout(loopDebounceRef.current);
      }
    };
  }, []);

  // Refresh data when reconnected
  useEffect(() => {
    if (connectionState === "connected") {
      // Refresh all stores after reconnection (using getState to avoid subscriptions)
      usePodStore.getState().fetchSidebarPods?.(usePodStore.getState().currentSidebarFilter);
      useRunnerStore.getState().fetchRunners?.();
      useTicketStore.getState().fetchTickets?.();
      useMeshStore.getState().fetchTopology?.();
      useAutopilotStore.getState().fetchAutopilotControllers?.();
      useLoopStore.getState().fetchLoops?.();
    }
  }, [connectionState]);

  // Memoize context value to prevent unnecessary consumer re-renders
  const value = useMemo<RealtimeContextValue>(() => ({
    connectionState,
    reconnect,
  }), [connectionState, reconnect]);

  return (
    <RealtimeContext.Provider value={value}>
      {children}
    </RealtimeContext.Provider>
  );
}
