"use client";

import React, { createContext, useContext, useEffect, useCallback } from "react";
import { useRealtimeConnection, useAllEventsSubscription } from "@/hooks/useRealtimeEvents";
import { usePodStore } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { useTicketStore } from "@/stores/ticket";
import { useMeshStore } from "@/stores/mesh";
import { useWorkspaceStore } from "@/stores/workspace";
import { useChannelStore } from "@/stores/channel";
import { useAutopilotStore } from "@/stores/autopilot";
import type { ConnectionState, RealtimeEvent, PodStatusChangedData, PodCreatedData, RunnerStatusData, TicketStatusChangedData, TerminalNotificationData, TaskCompletedData, PodTitleChangedData, PodInitProgressData, ChannelMessageData, AutopilotStatusChangedData, AutopilotIterationData, AutopilotCreatedData, AutopilotTerminatedData, AutopilotThinkingData, MREventData, PipelineEventData } from "@/lib/realtime";

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

  // Get store actions
  const podStore = usePodStore();
  const runnerStore = useRunnerStore();
  const ticketStore = useTicketStore();
  const meshStore = useMeshStore();
  const workspaceStore = useWorkspaceStore();
  const channelStore = useChannelStore();
  const autopilotStore = useAutopilotStore();

  // Handle all events and route to appropriate stores
  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      switch (event.type) {
        // Pod events
        case "pod:created": {
          const data = event.data as PodCreatedData;
          // Refresh pods list to include the new pod
          podStore.fetchPods?.();
          // Also refresh Mesh topology since a new pod affects the mesh
          meshStore.fetchTopology?.();
          console.log("[Realtime] Pod created:", data.pod_key);
          break;
        }

        case "pod:status_changed": {
          const data = event.data as PodStatusChangedData;
          // Check if pod exists in store - if not, refresh the list
          const existingPod = podStore.pods.find(p => p.pod_key === data.pod_key);
          if (!existingPod) {
            // Pod not in list, might be newly created - refresh to get it
            podStore.fetchPods?.();
            console.log("[Realtime] Pod not found, refreshing list:", data.pod_key);
          } else if (podStore.updatePodStatus) {
            // Update existing pod status (including error details if present)
            podStore.updatePodStatus(
              data.pod_key,
              data.status as "running" | "initializing" | "failed" | "paused" | "terminated" | "error",
              data.agent_status,
              data.error_code,
              data.error_message
            );
          }
          // Also refresh Mesh topology since pod status affects the mesh
          meshStore.fetchTopology?.();
          console.log("[Realtime] Pod status changed:", data.pod_key, data.status);
          break;
        }

        case "pod:agent_status_changed": {
          const data = event.data as PodStatusChangedData;
          if (data.agent_status) {
            podStore.updateAgentStatus(data.pod_key, data.agent_status);
          }
          console.log("[Realtime] Pod agent status changed:", data.pod_key, data.agent_status);
          break;
        }

        case "pod:terminated": {
          const data = event.data as PodStatusChangedData;
          // Update pod status to terminated
          if (podStore.updatePodStatus) {
            podStore.updatePodStatus(data.pod_key, "terminated");
          }
          // Also refresh Mesh topology since termination removes the pod from mesh
          meshStore.fetchTopology?.();
          console.log("[Realtime] Pod terminated:", data.pod_key);
          break;
        }

        case "pod:title_changed": {
          const data = event.data as PodTitleChangedData;
          // Update terminal pane title in workspace store
          workspaceStore.updatePaneTitle(data.pod_key, data.title);
          // Also update pod title in podStore for sidebar display
          podStore.updatePodTitle(data.pod_key, data.title);
          // Also update node title in meshStore for mesh view display
          meshStore.updateNodeTitle(data.pod_key, data.title);
          console.log("[Realtime] Pod title changed:", data.pod_key, data.title);
          break;
        }

        case "pod:init_progress": {
          const data = event.data as PodInitProgressData;
          // Update pod init progress in podStore
          podStore.updatePodInitProgress(data.pod_key, data.phase, data.progress, data.message);
          console.log("[Realtime] Pod init progress:", data.pod_key, data.phase, data.progress);
          break;
        }

        // Runner events
        case "runner:online":
        case "runner:offline":
        case "runner:updated": {
          const data = event.data as RunnerStatusData;
          // Update runner status in store
          if (runnerStore.updateRunnerStatus) {
            runnerStore.updateRunnerStatus(
              data.runner_id,
              data.status as "online" | "offline" | "maintenance" | "busy"
            );
          }
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
          ticketStore.fetchTickets?.();
          console.log("[Realtime] Ticket event:", event.type, data.identifier);
          break;
        }

        // Channel events
        case "channel:message": {
          const data = event.data as ChannelMessageData;
          // Only add message if it belongs to the current channel
          const currentChannel = channelStore.currentChannel;
          if (currentChannel && currentChannel.id === data.channel_id) {
            channelStore.addMessage({
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
          autopilotStore.updateAutopilotControllerStatus(
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
          autopilotStore.addIteration(data.autopilot_controller_key, {
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
          autopilotStore.fetchAutopilotControllers?.();
          console.log("[Realtime] Autopilot created:", data.autopilot_controller_key);
          break;
        }

        case "autopilot:terminated": {
          const data = event.data as AutopilotTerminatedData;
          autopilotStore.removeAutopilotController(data.autopilot_controller_key);
          console.log("[Realtime] Autopilot terminated:", data.autopilot_controller_key, data.reason);
          break;
        }

        case "autopilot:thinking": {
          const data = event.data as AutopilotThinkingData;
          autopilotStore.updateThinking(data.autopilot_controller_key, data);
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
          if (data.ticket_id) {
            ticketStore.fetchTickets?.();
          }
          // Refresh pods if this MR is associated with a pod
          if (data.pod_id) {
            podStore.fetchPods?.();
          }
          console.log("[Realtime] MR event:", event.type, data.mr_iid, data.state);
          break;
        }

        // Pipeline events
        case "pipeline:updated": {
          const data = event.data as PipelineEventData;
          // Refresh tickets if this pipeline is associated with a ticket
          if (data.ticket_id) {
            ticketStore.fetchTickets?.();
          }
          // Refresh pods if this pipeline is associated with a pod
          if (data.pod_id) {
            podStore.fetchPods?.();
          }
          console.log("[Realtime] Pipeline event:", data.pipeline_id, data.pipeline_status);
          break;
        }

        default:
          console.log("[Realtime] Unknown event:", event.type);
      }
    },
    [podStore, runnerStore, ticketStore, meshStore, workspaceStore, channelStore, autopilotStore, onTerminalNotification, onTaskCompleted]
  );

  // Subscribe to all events
  useAllEventsSubscription(handleEvent, [handleEvent]);

  // Refresh data when reconnected
  useEffect(() => {
    if (connectionState === "connected") {
      // Refresh all stores after reconnection
      podStore.fetchPods?.();
      runnerStore.fetchRunners?.();
      ticketStore.fetchTickets?.();
      meshStore.fetchTopology?.();
      autopilotStore.fetchAutopilotControllers?.();
    }
    // Store objects are stable, only connectionState changes trigger refresh
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connectionState]);

  const value: RealtimeContextValue = {
    connectionState,
    reconnect,
  };

  return (
    <RealtimeContext.Provider value={value}>
      {children}
    </RealtimeContext.Provider>
  );
}
