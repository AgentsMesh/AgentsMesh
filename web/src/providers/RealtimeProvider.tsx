"use client";

import React, { createContext, useContext, useEffect, useCallback } from "react";
import { useRealtimeConnection, useAllEventsSubscription } from "@/hooks/useRealtimeEvents";
import { usePodStore } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { useTicketStore } from "@/stores/ticket";
import { useMeshStore } from "@/stores/mesh";
import { useWorkspaceStore } from "@/stores/workspace";
import { useChannelStore } from "@/stores/channel";
import type { ConnectionState, RealtimeEvent, PodStatusChangedData, PodCreatedData, RunnerStatusData, TicketStatusChangedData, TerminalNotificationData, TaskCompletedData, PodTitleChangedData, ChannelMessageData } from "@/lib/realtime";

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
            // Update existing pod status
            podStore.updatePodStatus(
              data.pod_key,
              data.status as "running" | "initializing" | "failed" | "paused" | "terminated",
              data.agent_status
            );
          }
          // Also refresh Mesh topology since pod status affects the mesh
          meshStore.fetchTopology?.();
          console.log("[Realtime] Pod status changed:", data.pod_key, data.status);
          break;
        }

        case "pod:agent_status_changed": {
          const data = event.data as PodStatusChangedData;
          // Update agent status in store
          if (podStore.updatePodStatus) {
            podStore.updatePodStatus(
              data.pod_key,
              data.status as "running" | "initializing" | "failed" | "paused" | "terminated",
              data.agent_status
            );
          }
          // Also refresh Mesh topology since agent status affects the mesh display
          meshStore.fetchTopology?.();
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
          console.log("[Realtime] Pod title changed:", data.pod_key, data.title);
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

        default:
          console.log("[Realtime] Unknown event:", event.type);
      }
    },
    [podStore, runnerStore, ticketStore, meshStore, workspaceStore, channelStore, onTerminalNotification, onTaskCompleted]
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
