import { readCurrentUser, readCurrentOrg } from "@/stores/auth";
import { create } from "zustand";
import { create as protoCreate, fromBinary, toBinary } from "@bufbuild/protobuf";
import { getChannelState } from "@/lib/wasm-core";
import { getErrorMessage } from "@/lib/utils";
import {
  listChannelMessagesRaw,
  sendChannelMessage,
  editChannelMessage,
  deleteChannelMessage,
  getChannelUnreadCounts,
  markChannelRead,
  muteChannel as muteChannelConnect,
} from "@/lib/api/facade/channelConnect";
import { getCache, updateCache } from "./channelMessageTypes";
import type { ChannelMessageState } from "./channelMessageTypes";
import type { ChannelMessage } from "@/lib/api/facade/channel";
import {
  fromWasmProjection,
  type WasmChannelMessage,
} from "./channelMessageWasmProjection";
import { channelMessageToProto } from "@/lib/api/channelProtoMap";
import {
  ReplaceCachedChannelMessagesRequestSchema,
  InsertChannelMessageRequestSchema,
  ApplyIncomingChannelMessageRequestSchema,
  ApplyChannelMessageEditedEventRequestSchema,
  ReplaceChannelUnreadCountsRequestSchema,
} from "@proto/channel_state/v1/mutations_pb";
import type { ChannelMessage as ProtoStateMessage } from "@proto/channel_state/v1/channel_state_pb";
import { registerOrgScopedReset } from "@/lib/org-scope/registry";

export { EMPTY_CACHE, type ChannelMessageCache } from "./channelMessageTypes";

/** Number of messages to fetch on initial channel load. */
export const INITIAL_MESSAGE_LIMIT = 20;
/** Number of messages to fetch when loading older history. */
export const LOAD_MORE_MESSAGE_LIMIT = 30;

const svc = () => getChannelState();

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

// state proto ChannelMessage → WasmChannelMessage (snake_case + sender nested +
// content_json/mentions_json string). Read side, zero-JSON; fromWasmProjection
// then parses the rich AST. Replaces the get_messages_json serde read.
function messageToCache(m: ProtoStateMessage): WasmChannelMessage {
  return {
    id: Number(m.id),
    channel_id: Number(m.channelId),
    sender_pod: m.senderPod,
    sender_user_id: m.senderUserId !== undefined && m.senderUserId !== BigInt(0) ? Number(m.senderUserId) : undefined,
    sender_user: m.senderUser ? {
      id: Number(m.senderUser.id), username: m.senderUser.username,
      name: m.senderUser.name, avatar_url: m.senderUser.avatarUrl,
    } : undefined,
    sender_pod_info: m.senderPodInfo ? {
      pod_key: m.senderPodInfo.podKey, alias: m.senderPodInfo.alias,
      agent: m.senderPodInfo.agent ? { name: m.senderPodInfo.agent.name } : undefined,
    } : undefined,
    message_type: m.messageType,
    body: m.body,
    content_json: m.contentJson || undefined,
    mentions_json: m.mentionsJson || undefined,
    reply_to: m.replyTo !== undefined && m.replyTo !== BigInt(0) ? Number(m.replyTo) : undefined,
    edited_at: m.editedAt || undefined,
    is_deleted: m.isDeleted,
    created_at: m.createdAt,
  } as WasmChannelMessage;
}

export function readMessages(channelId: number): { messages: ChannelMessage[]; hasMore: boolean } {
  const req = fromBinary(ReplaceCachedChannelMessagesRequestSchema, svc().get_messages_bytes(BigInt(channelId)));
  return { messages: req.messages.map(messageToCache).map(fromWasmProjection), hasMore: req.hasMore };
}

const bumpMessages = () =>
  useChannelMessageStore.setState((s) => ({ _messagesTick: s._messagesTick + 1 }));

function dispatchInsertMessage(channelId: number, message: ChannelMessage) {
  const req = protoCreate(InsertChannelMessageRequestSchema, {
    channelId: BigInt(channelId),
    message: channelMessageToProto(message),
  });
  svc().insert_channel_message(toBinary(InsertChannelMessageRequestSchema, req));
}

function dispatchIncomingMessage(message: ChannelMessage): boolean {
  const req = protoCreate(ApplyIncomingChannelMessageRequestSchema, {
    channelId: BigInt(message.channel_id),
    message: channelMessageToProto(message),
  });
  return svc().apply_incoming_channel_message(
    toBinary(ApplyIncomingChannelMessageRequestSchema, req),
  );
}

function dispatchMessageEdited(channelId: number, edit: {
  id: number; body: string; content?: ChannelMessage["content"]; mentions?: ChannelMessage["mentions"]; edited_at?: string;
}) {
  const req = protoCreate(ApplyChannelMessageEditedEventRequestSchema, {
    channelId: BigInt(channelId),
    messageId: BigInt(edit.id),
    body: edit.body,
    content: edit.content ? JSON.stringify(edit.content) : undefined,
    mentions: edit.mentions
      ? Object.fromEntries(
          Object.entries(edit.mentions).map(([k, v]) => [k, typeof v === "string" ? v : JSON.stringify(v)]),
        )
      : {},
    editedAt: edit.edited_at ?? "",
  });
  svc().apply_channel_message_edited_event(
    toBinary(ApplyChannelMessageEditedEventRequestSchema, req),
  );
}

export const useChannelMessageStore = create<ChannelMessageState>((set, get) => ({
  cache: {},
  _messagesTick: 0,
  _unreadTick: 0,

  fetchMessages: async (channelId, limit = INITIAL_MESSAGE_LIMIT, beforeId) => {
    const isLoadMore = beforeId !== undefined;
    const current = getCache(get(), channelId);
    if (isLoadMore ? current.loadingMore : current.loading) return;

    set((state) =>
      updateCache(state, channelId, isLoadMore ? { loadingMore: true } : { loading: true, error: null }),
    );

    try {
      const respBytes = await listChannelMessagesRaw(orgSlug(), channelId, { beforeId, limit });
      // wire bytes → Rust wire→state + set/prepend_messages business logic.
      if (isLoadMore) {
        svc().apply_fetched_messages_prepend(BigInt(channelId), respBytes);
      } else {
        svc().apply_fetched_messages(BigInt(channelId), respBytes);
      }
      set((state) => updateCache(state, channelId, {
        loading: false, loadingMore: false, error: null,
      }));
      bumpMessages();
    } catch (error: unknown) {
      const msg = getErrorMessage(error, "Unknown error");
      console.error("Failed to fetch messages:", msg);
      set((state) => updateCache(state, channelId, {
        loading: false, loadingMore: false, error: isLoadMore ? null : msg,
      }));
    }
  },

  sendMessage: async (channelId, payload, podKey) => {
    try {
      const msg = await sendChannelMessage(orgSlug(), channelId, {
        source: payload.source,
        mentions: payload.mentions && Object.keys(payload.mentions).length > 0 ? payload.mentions : undefined,
        attachment_key: payload.attachment_key,
        pod_key: podKey,
      });

      // POST response may lack sender_user — backfill from auth store.
      if (!msg.sender_user && msg.sender_user_id) {
        const authUser = readCurrentUser();
        if (authUser && authUser.id === msg.sender_user_id) {
          msg.sender_user = {
            id: authUser.id,
            username: authUser.username,
            name: authUser.name,
            avatar_url: authUser.avatar_url,
          };
        }
      }

      dispatchInsertMessage(channelId, msg);
      bumpMessages();
      return msg;
    } catch (error: unknown) {
      console.error("Failed to send message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  addMessage: (_channelId, message) => {
    dispatchIncomingMessage(message);
    bumpMessages();
  },

  editMessage: async (channelId, messageId, payload) => {
    try {
      const updated = await editChannelMessage(orgSlug(), channelId, messageId, {
        source: payload.source,
        mentions: payload.mentions && Object.keys(payload.mentions).length > 0 ? payload.mentions : undefined,
      });
      dispatchMessageEdited(channelId, {
        id: messageId, body: updated.body,
        content: updated.content, mentions: updated.mentions, edited_at: updated.edited_at,
      });
      bumpMessages();
    } catch (error: unknown) {
      console.error("Failed to edit message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  deleteMessage: async (channelId, messageId) => {
    try {
      await deleteChannelMessage(orgSlug(), channelId, messageId);
      svc().remove_message(BigInt(channelId), BigInt(messageId));
      bumpMessages();
    } catch (error: unknown) {
      console.error("Failed to delete message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  updateMessage: (channelId, data) => {
    dispatchMessageEdited(channelId, {
      id: data.id, body: data.body, content: data.content,
      mentions: data.mentions, edited_at: data.edited_at,
    });
    bumpMessages();
  },

  removeMessage: (channelId, messageId) => {
    svc().remove_message(BigInt(channelId), BigInt(messageId));
    bumpMessages();
  },

  fetchUnreadCounts: async () => {
    try {
      const unread = await getChannelUnreadCounts(orgSlug());
      const req = protoCreate(ReplaceChannelUnreadCountsRequestSchema, {
        counts: Object.fromEntries(Object.entries(unread).map(([k, v]) => [BigInt(k), v])) as unknown as { [k: string]: number },
      });
      svc().replace_channel_unread_counts(toBinary(ReplaceChannelUnreadCountsRequestSchema, req));
      set((s) => ({ _unreadTick: s._unreadTick + 1 }));
    } catch (error: unknown) {
      console.error("Failed to fetch unread counts:", getErrorMessage(error, "Unknown error"));
    }
  },

  markRead: async (channelId, messageId) => {
    try {
      await markChannelRead(orgSlug(), channelId, messageId);
      svc().clear_channel_unread(BigInt(channelId));
      set((s) => ({ _unreadTick: s._unreadTick + 1 }));
    } catch (error: unknown) {
      console.error("Failed to mark channel as read:", getErrorMessage(error, "Unknown error"));
    }
  },

  muteChannel: async (channelId, muted) => {
    try {
      await muteChannelConnect(orgSlug(), channelId, muted);
    } catch (error: unknown) {
      console.error("Failed to update mute setting:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  incrementUnread: (channelId) => {
    svc().increment_unread(BigInt(channelId));
    set((s) => ({ _unreadTick: s._unreadTick + 1 }));
  },

  clearChannelUnread: (channelId) => {
    svc().clear_channel_unread(BigInt(channelId));
    set((s) => ({ _unreadTick: s._unreadTick + 1 }));
  },
}));

export {
  useChannelMessages,
  useUnreadCounts,
  useUnreadCount,
  useTotalUnreadCount,
  type ChannelMessagesView,
} from "./channelMessageSelectors";

registerOrgScopedReset(() => {
  const emptyReq = protoCreate(ReplaceChannelUnreadCountsRequestSchema, { counts: {} });
  svc().replace_channel_unread_counts(toBinary(ReplaceChannelUnreadCountsRequestSchema, emptyReq));
  useChannelMessageStore.setState((s) => ({
    cache: {},
    _messagesTick: s._messagesTick + 1,
    _unreadTick: s._unreadTick + 1,
  }));
});
