import { usePodStore } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { useTicketStore } from "@/stores/ticket";
import { useChannelStore, useChannelMessageStore } from "@/stores/channel";
import { useAuthStore } from "@/stores/auth";
import type {
  RealtimeEvent, RunnerStatusData, TicketStatusChangedData,
  ChannelMessageData, ChannelMessageEditedData, ChannelMessageDeletedData,
  ChannelMemberChangedData, MREventData, PipelineEventData,
} from "@/lib/realtime";

export { handlePodEvent } from "./realtimePodHandlers";
export { handleAutopilotEvent, handleLoopEvent } from "./realtimeFeatureHandlers";

export type DebounceRef = React.MutableRefObject<ReturnType<typeof setTimeout> | null>;

export function handleChannelEvent(event: RealtimeEvent, channelDebounceRef?: DebounceRef) {
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
      const currentUserId = useAuthStore.getState().user?.id;
      const viewingChannelId = useChannelStore.getState().selectedChannelId;
      const isSelf = currentUserId != null && data.sender_user_id === currentUserId;
      const isViewing = viewingChannelId === data.channel_id;
      if (!isSelf && !isViewing) {
        msgState.incrementUnread(data.channel_id);
      }
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
    case "channel:member_added":
    case "channel:member_removed": {
      const data = event.data as ChannelMemberChangedData;
      const chState = useChannelStore.getState();
      const delta = event.type === "channel:member_added" ? 1 : -1;
      chState.patchChannelMemberCount?.(data.channel_id, delta);
      if (chState.currentChannel?.id === data.channel_id && channelDebounceRef) {
        if (channelDebounceRef.current) clearTimeout(channelDebounceRef.current);
        channelDebounceRef.current = setTimeout(() => {
          channelDebounceRef.current = null;
          useChannelStore.getState().fetchChannel?.(data.channel_id);
        }, 300);
      }
      break;
    }
  }
}

export function handleInfraEvent(event: RealtimeEvent, ticketDebounceRef?: DebounceRef) {
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

      if (event.type === "ticket:status_changed" && data.slug && data.status) {
        ticketState.updateTicketStatusFromEvent(data.slug, data.status, data.previous_status);
      } else if (event.type === "ticket:deleted" && data.slug) {
        ticketState.removeTicketFromEvent(data.slug);
      }

      debouncedTicketRefetch(ticketDebounceRef);
      break;
    }
    case "mr:created":
    case "mr:updated":
    case "mr:merged":
    case "mr:closed": {
      const data = event.data as MREventData;
      if (data.ticket_slug) useTicketStore.getState().fetchTicket?.(data.ticket_slug);
      if (data.pod_id) {
        const pod = usePodStore.getState().pods.find(p => p.id === data.pod_id);
        if (pod) usePodStore.getState().fetchPod?.(pod.pod_key);
      }
      break;
    }
    case "pipeline:updated": {
      const data = event.data as PipelineEventData;
      if (data.ticket_slug) useTicketStore.getState().fetchTicket?.(data.ticket_slug);
      if (data.pod_id) {
        const pod = usePodStore.getState().pods.find(p => p.id === data.pod_id);
        if (pod) usePodStore.getState().fetchPod?.(pod.pod_key);
      }
      break;
    }
  }
}

function debouncedTicketRefetch(debounceRef: DebounceRef | undefined) {
  if (!debounceRef) {
    useTicketStore.getState().fetchTickets?.();
    return;
  }
  if (debounceRef.current) clearTimeout(debounceRef.current);
  debounceRef.current = setTimeout(() => {
    debounceRef.current = null;
    const s = useTicketStore.getState();
    s.fetchTickets?.();
    if (s.currentTicket?.slug) {
      s.fetchTicket?.(s.currentTicket.slug);
    }
  }, 500);
}
