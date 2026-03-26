import { usePodStore } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { useTicketStore } from "@/stores/ticket";
import { useChannelMessageStore } from "@/stores/channel";
import { useAutopilotStore } from "@/stores/autopilot";
import { useLoopStore } from "@/stores/loop";
import type {
  RealtimeEvent, RunnerStatusData, TicketStatusChangedData,
  ChannelMessageData, ChannelMessageEditedData, ChannelMessageDeletedData,
  AutopilotStatusChangedData, AutopilotIterationData,
  AutopilotTerminatedData, AutopilotThinkingData, MREventData, PipelineEventData,
  LoopRunEventData, LoopRunWarningData,
} from "@/lib/realtime";

export { handlePodEvent } from "./realtimePodHandlers";

export function handleChannelEvent(event: RealtimeEvent) {
  const msgState = useChannelMessageStore.getState();
  switch (event.type) {
    case "channel:message": {
      const data = event.data as ChannelMessageData;
      msgState.addMessage(data.channel_id, {
        id: data.id, channel_id: data.channel_id, sender_pod: data.sender_pod,
        sender_user_id: data.sender_user_id,
        message_type: data.message_type as "text" | "system" | "code" | "command",
        content: data.content, metadata: data.metadata, created_at: data.created_at,
        ...(data.sender_user_id && data.sender_name ? {
          sender_user: { id: data.sender_user_id, username: data.sender_name, name: data.sender_name },
        } : {}),
      });
      msgState.incrementUnread(data.channel_id);
      break;
    }
    case "channel:message_edited": {
      const data = event.data as ChannelMessageEditedData;
      msgState.updateMessage(data.channel_id, data);
      break;
    }
    case "channel:message_deleted": {
      const data = event.data as ChannelMessageDeletedData;
      msgState.removeMessage(data.channel_id, data.id);
      break;
    }
  }
}

export function handleInfraEvent(event: RealtimeEvent) {
  switch (event.type) {
    case "runner:online":
    case "runner:offline":
    case "runner:updated": {
      const data = event.data as RunnerStatusData;
      useRunnerStore.getState().updateRunnerStatus(data.runner_id, data.status as "online" | "offline" | "maintenance" | "busy");
      break;
    }
    case "ticket:created":
    case "ticket:updated":
    case "ticket:status_changed":
    case "ticket:moved":
    case "ticket:deleted": {
      const data = event.data as TicketStatusChangedData;
      const ticketState = useTicketStore.getState();
      ticketState.fetchTickets?.();
      if (event.type !== "ticket:deleted" && data.slug && ticketState.currentTicket?.slug === data.slug) {
        ticketState.fetchTicket?.(data.slug);
      }
      break;
    }
    case "mr:created":
    case "mr:updated":
    case "mr:merged":
    case "mr:closed": {
      const data = event.data as MREventData;
      if (data.ticket_slug || data.ticket_id) useTicketStore.getState().fetchTickets?.();
      if (data.pod_id) usePodStore.getState().fetchPods?.();
      break;
    }
    case "pipeline:updated": {
      const data = event.data as PipelineEventData;
      if (data.ticket_slug || data.ticket_id) useTicketStore.getState().fetchTickets?.();
      if (data.pod_id) usePodStore.getState().fetchPods?.();
      break;
    }
  }
}

export function handleAutopilotEvent(event: RealtimeEvent) {
  const store = useAutopilotStore.getState();
  switch (event.type) {
    case "autopilot:status_changed": {
      const data = event.data as AutopilotStatusChangedData;
      store.updateAutopilotControllerStatus(data.autopilot_controller_key, data.phase, data.current_iteration, data.max_iterations, data.circuit_breaker_state, data.circuit_breaker_reason);
      break;
    }
    case "autopilot:iteration": {
      const data = event.data as AutopilotIterationData;
      store.addIteration(data.autopilot_controller_key, {
        id: 0, autopilot_controller_id: 0, iteration: data.iteration, phase: data.phase,
        summary: data.summary, files_changed: data.files_changed, duration_ms: data.duration_ms,
        created_at: new Date().toISOString(),
      });
      break;
    }
    case "autopilot:created": {
      store.fetchAutopilotControllers?.();
      break;
    }
    case "autopilot:terminated": {
      const data = event.data as AutopilotTerminatedData;
      store.removeAutopilotController(data.autopilot_controller_key);
      break;
    }
    case "autopilot:thinking": {
      const data = event.data as AutopilotThinkingData;
      store.updateThinking(data.autopilot_controller_key, data);
      break;
    }
  }
}

export function handleLoopEvent(
  event: RealtimeEvent,
  debounceRef: React.MutableRefObject<ReturnType<typeof setTimeout> | null>,
  t: (key: string, params?: Record<string, string | number>) => string,
  showWarning: (title: string, description: string) => void
) {
  switch (event.type) {
    case "loop_run:started":
    case "loop_run:completed":
    case "loop_run:failed": {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        debounceRef.current = null;
        const s = useLoopStore.getState();
        s.fetchLoops?.();
        if (s.currentLoop?.id === (event.data as LoopRunEventData).loop_id) {
          s.fetchLoop?.(s.currentLoop.slug);
          useLoopStore.setState({ runsOffset: 0 });
          s.fetchRuns?.(s.currentLoop.slug, { limit: 20, offset: 0 });
        }
      }, 500);
      break;
    }
    case "loop_run:warning": {
      const data = event.data as LoopRunWarningData;
      showWarning(t("loops.runWarningTitle", { runNumber: data.run_number }), data.detail || data.warning);
      break;
    }
  }
}
