import { readCurrentUser, readCurrentOrg } from "@/stores/auth";
import { create } from "zustand";
import { create as protoCreate, toBinary } from "@bufbuild/protobuf";
import { getChannelState } from "@/lib/wasm-core";
import { getErrorMessage } from "@/lib/utils";
import {
  listChannelMessagesRaw,
  sendChannelMessage,
  editChannelMessage,
  deleteChannelMessage,
} from "@/lib/api/facade/channelConnect";
import { getCache, updateCache } from "./channelMessageTypes";
import type { ChannelMessageState } from "./channelMessageTypes";
import type { ChannelMessage } from "@/lib/api/facade/channel";
import {
  dispatchInsertMessage,
  dispatchIncomingMessage,
  dispatchMessageEdited,
} from "./channelMessageDispatch";
import { createReadStateActions } from "./channelReadStateActions";
import { ReplaceChannelUnreadCountsRequestSchema } from "@proto/channel_state/v1/mutations_pb";
import { registerOrgScopedReset } from "@/lib/org-scope/registry";

export { EMPTY_CACHE, type ChannelMessageCache } from "./channelMessageTypes";
export { readMessages } from "./channelMessageDispatch";

/** Number of messages to fetch on initial channel load. */
export const INITIAL_MESSAGE_LIMIT = 20;
/** Number of messages to fetch when loading older history. */
export const LOAD_MORE_MESSAGE_LIMIT = 30;

const svc = () => getChannelState();

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

const bumpMessages = () =>
  useChannelMessageStore.setState((s) => ({ _messagesTick: s._messagesTick + 1 }));

export const useChannelMessageStore = create<ChannelMessageState>((...a) => {
  const [set, get] = a;
  return {
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

    ...createReadStateActions(...a),
  };
});

export {
  useChannelMessages,
  useUnreadCounts,
  useUnreadCount,
  useMentionCounts,
  useManuallyUnread,
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
