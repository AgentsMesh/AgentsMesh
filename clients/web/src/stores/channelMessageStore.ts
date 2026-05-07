import { readCurrentUser } from "@/stores/auth";
import { create } from "zustand";
import { useMemo } from "react";
import { getChannelService } from "@/lib/wasm-core";
import { getErrorMessage } from "@/lib/utils";
import { useAuthStore } from "./auth";
import { getCache, updateCache } from "./channelMessageTypes";
import type { ChannelMessageState } from "./channelMessageTypes";
import type { ChannelMessage } from "@/lib/api/channel";
import { registerOrgScopedReset } from "@/lib/org-scope/registry";

export { EMPTY_CACHE, type ChannelMessageCache } from "./channelMessageTypes";

/** Number of messages to fetch on initial channel load. */
export const INITIAL_MESSAGE_LIMIT = 20;
/** Number of messages to fetch when loading older history. */
export const LOAD_MORE_MESSAGE_LIMIT = 30;

const svc = () => getChannelService();

export function readMessages(channelId: number): { messages: ChannelMessage[]; hasMore: boolean } {
  const raw = svc().get_messages_json(BigInt(channelId));
  if (!raw) return { messages: [], hasMore: false };
  const parsed = typeof raw === "string" ? JSON.parse(raw) : (raw as { messages?: ChannelMessage[]; has_more?: boolean });
  return { messages: parsed.messages || [], hasMore: parsed.has_more ?? false };
}

const bumpMessages = () =>
  useChannelMessageStore.setState((s) => ({ _messagesTick: s._messagesTick + 1 }));

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
      await svc().fetch_messages(
        BigInt(channelId),
        limit,
        beforeId !== undefined ? BigInt(beforeId) : undefined,
      );
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
      const req: Record<string, unknown> = { source: payload.source };
      if (payload.mentions && Object.keys(payload.mentions).length > 0) req.mentions = payload.mentions;
      if (payload.attachment_key) req.attachment_key = payload.attachment_key;
      if (podKey) req.pod_key = podKey;
      const json = await svc().send_message(BigInt(channelId), JSON.stringify(req));
      const msg = JSON.parse(json) as ChannelMessage;

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

      bumpMessages();
      return msg;
    } catch (error: unknown) {
      console.error("Failed to send message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  addMessage: (_channelId, message) => {
    svc().on_new_message(JSON.stringify(message));
    bumpMessages();
  },

  editMessage: async (channelId, messageId, payload) => {
    try {
      const req: Record<string, unknown> = { source: payload.source };
      if (payload.mentions && Object.keys(payload.mentions).length > 0) req.mentions = payload.mentions;
      await svc().edit_message(BigInt(channelId), BigInt(messageId), JSON.stringify(req));
      bumpMessages();
    } catch (error: unknown) {
      console.error("Failed to edit message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  deleteMessage: async (channelId, messageId) => {
    try {
      await svc().delete_message(BigInt(channelId), BigInt(messageId));
      bumpMessages();
    } catch (error: unknown) {
      console.error("Failed to delete message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  updateMessage: (channelId, data) => {
    svc().update_message_local(BigInt(channelId), JSON.stringify(data));
    bumpMessages();
  },

  removeMessage: (channelId, messageId) => {
    svc().remove_message_local(BigInt(channelId), BigInt(messageId));
    bumpMessages();
  },

  fetchUnreadCounts: async () => {
    try {
      await svc().fetch_unread_counts();
      set((s) => ({ _unreadTick: s._unreadTick + 1 }));
    } catch (error: unknown) {
      console.error("Failed to fetch unread counts:", getErrorMessage(error, "Unknown error"));
    }
  },

  markRead: async (channelId, messageId) => {
    try {
      await svc().mark_read(BigInt(channelId), BigInt(messageId));
      set((s) => ({ _unreadTick: s._unreadTick + 1 }));
    } catch (error: unknown) {
      console.error("Failed to mark channel as read:", getErrorMessage(error, "Unknown error"));
    }
  },

  muteChannel: async (channelId, muted) => {
    try {
      await svc().mute_channel(BigInt(channelId), muted);
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

// ── Selectors: Rust is SSOT. These hooks subscribe to tick counters so
// components re-render when Rust state mutates — no parallel JS copy.

export interface ChannelMessagesView {
  messages: ChannelMessage[];
  hasMore: boolean;
}

const EMPTY_VIEW: ChannelMessagesView = { messages: [], hasMore: false };

export function useChannelMessages(channelId: number | null | undefined): ChannelMessagesView {
  const tick = useChannelMessageStore((s) => s._messagesTick);
  return useMemo(() => {
    if (channelId == null) return EMPTY_VIEW;
    return readMessages(channelId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, channelId]);
}

export function useUnreadCounts(): Record<number, number> {
  const tick = useChannelMessageStore((s) => s._unreadTick);
  return useMemo(() => {
    try {
      return JSON.parse(svc().unread_counts_json()) as Record<number, number>;
    } catch {
      return {};
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick]);
}

export function useUnreadCount(channelId: number | null | undefined): number {
  const tick = useChannelMessageStore((s) => s._unreadTick);
  return useMemo(() => {
    if (channelId == null) return 0;
    return svc().get_unread_count(BigInt(channelId));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, channelId]);
}

export function useTotalUnreadCount(): number {
  const tick = useChannelMessageStore((s) => s._unreadTick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => svc().total_unread_count(), [tick]);
}

registerOrgScopedReset(() => {
  const ch = svc() as unknown as { set_unread_counts?: (json: string) => void };
  ch.set_unread_counts?.("{}");
  useChannelMessageStore.setState((s) => ({
    cache: {},
    _messagesTick: s._messagesTick + 1,
    _unreadTick: s._unreadTick + 1,
  }));
});
