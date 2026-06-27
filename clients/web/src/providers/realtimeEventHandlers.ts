import { usePodStore } from "@/stores/pod";
import { useRunnerStore } from "@/stores/runner";
import { getTicketState, getPodState, parseWasmAny } from "@/lib/wasm-core";
import { useTicketStore } from "@/stores/ticket";
import { useChannelStore, useChannelMessageStore } from "@/stores/channel";
import { readCurrentUser } from "@/stores/auth";
import type { PodData } from "@/lib/api/facade/pod";
import {
  type RealtimeEvent,
  decodeEventData,
  ChannelMemberChangedEventDataSchema,
  MrEventDataSchema,
  PipelineEventDataSchema,
} from "@/lib/realtime";

export { handlePodEvent } from "./realtimePodHandlers";
export { handleAutopilotEvent, handleLoopEvent } from "./realtimeFeatureHandlers";
export { handleBlockstoreEvent } from "@/stores/blockstoreSubscribe";

export type DebounceRef = React.MutableRefObject<ReturnType<typeof setTimeout> | null>;

export function handleChannelEvent(event: RealtimeEvent, channelDebounceRef?: DebounceRef) {
  switch (event.type) {
    case "channel:message":
    case "channel:message_edited":
    case "channel:message_deleted": {
      // Rust Core (event_dispatch.rs → on_new_message / update_message /
      // remove_message) owns ALL state mutation here — message persistence,
      // preview, and the unread/mention business rules (self-message +
      // active-channel gating). The renderer only re-reads the result.
      bumpChannelStores();
      break;
    }
    case "channel:member_added":
    case "channel:member_removed": {
      // Member count is owned by Rust dispatch (patch_member_count). The
      // ONLY thing JS still does is the server-data refetch that Rust can't
      // synthesize: when *I* am added, a brand-new channel appears that
      // isn't in my cached list yet, so pull the list from the backend.
      const data = decodeEventData(ChannelMemberChangedEventDataSchema, event.data);
      const channelId = Number(data.channelId);
      const userId = Number(data.userId);
      const currentUserId = readCurrentUser()?.id;
      const chState = useChannelStore.getState();
      if (currentUserId != null && userId === currentUserId) {
        chState.fetchChannels?.({ includeArchived: true });
      } else if (chState.currentChannel?.id === channelId && channelDebounceRef) {
        if (channelDebounceRef.current) clearTimeout(channelDebounceRef.current);
        channelDebounceRef.current = setTimeout(() => {
          channelDebounceRef.current = null;
          useChannelStore.getState().fetchChannel?.(channelId);
        }, 300);
      }
      bumpChannelStores();
      break;
    }
  }
}

// Trigger React re-read of Rust-updated channel state. Rust dispatch has
// already mutated AppState by the time this runs (the dispatch hook fires
// before external handlers); these bumps are the only thing the renderer
// needs to do for channel realtime.
function bumpChannelStores() {
  useChannelStore.setState((s) => ({ _tick: s._tick + 1 }));
  // _unreadTick too: Rust on_new_message updated unread/mention counts, and the
  // sidebar badge selectors (useUnreadCounts / useMentionCounts) key on
  // _unreadTick — without this a realtime message never re-renders the badge.
  useChannelMessageStore.setState((s) => ({
    _messagesTick: s._messagesTick + 1,
    _unreadTick: s._unreadTick + 1,
  }));
}

export function handleInfraEvent(event: RealtimeEvent, ticketDebounceRef?: DebounceRef) {
  switch (event.type) {
    case "runner:online":
    case "runner:offline":
    case "runner:updated": {
      // Rust event_dispatch owns runner status (update_runner_status in
      // runtime.state, in all three lists); bump triggers the React selectors
      // to re-read. Desktop mirrors via the main-pushed runner snapshot.
      useRunnerStore.setState((s) => ({ _tick: s._tick + 1 }));
      break;
    }
    case "ticket:created":
    case "ticket:updated":
    case "ticket:status_changed":
    case "ticket:moved":
    case "ticket:deleted": {
      // Rust event_dispatch owns the status patch (update_ticket_status) and
      // deletion (remove_ticket) in runtime.state; bump triggers the React
      // selectors to re-read. The debounced refetch pulls full server data
      // (newly-created tickets / fields the partial event can't synthesize).
      useTicketStore.setState((s) => ({ _tick: s._tick + 1 }));
      debouncedTicketRefetch(ticketDebounceRef);
      break;
    }
    case "mr:created":
    case "mr:updated":
    case "mr:merged":
    case "mr:closed": {
      const data = decodeEventData(MrEventDataSchema, event.data);
      if (data.ticketSlug) useTicketStore.getState().fetchTicket?.(data.ticketSlug);
      if (data.podId != null) {
        const podId = Number(data.podId);
        const pods = JSON.parse(getPodState().pods_json()) as PodData[];
        const pod = pods.find((p) => p.id === podId);
        if (pod) usePodStore.getState().fetchPod?.(pod.pod_key);
      }
      break;
    }
    case "pipeline:updated": {
      const data = decodeEventData(PipelineEventDataSchema, event.data);
      if (data.ticketSlug) useTicketStore.getState().fetchTicket?.(data.ticketSlug);
      if (data.podId != null) {
        const podId = Number(data.podId);
        const pods = JSON.parse(getPodState().pods_json()) as PodData[];
        const pod = pods.find((p) => p.id === podId);
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
    const currentTicket = parseWasmAny<{ slug: string }>(getTicketState().current_ticket_json());
    if (currentTicket?.slug) {
      s.fetchTicket?.(currentTicket.slug);
    }
  }, 500);
}
