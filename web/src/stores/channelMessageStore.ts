import { create } from "zustand";
import { channelApi } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";
import { useAuthStore } from "./auth";
import { getCache, updateCache } from "./channelMessageTypes";
import type { ChannelMessageState } from "./channelMessageTypes";

export { EMPTY_CACHE, type ChannelMessageCache } from "./channelMessageTypes";

/** Number of messages to fetch on initial channel load (kept small for fast first paint). */
export const INITIAL_MESSAGE_LIMIT = 20;

/** Number of messages to fetch when loading older history. */
export const LOAD_MORE_MESSAGE_LIMIT = 30;

export const useChannelMessageStore = create<ChannelMessageState>((set, get) => ({
  cache: {},
  unreadCounts: {},

  fetchMessages: async (channelId, limit = INITIAL_MESSAGE_LIMIT, beforeId) => {
    const isLoadMore = beforeId !== undefined;
    const current = getCache(get(), channelId);
    if (isLoadMore ? current.loadingMore : current.loading) return; // dedup guard

    set((state) =>
      updateCache(state, channelId, isLoadMore ? { loadingMore: true } : { loading: true, error: null })
    );

    try {
      const response = await channelApi.getMessages(channelId, limit, beforeId);
      const newMessages = response.messages || [];
      const hasMore = response.has_more ?? false;

      set((state) => {
        const existing = getCache(state, channelId);
        return updateCache(state, channelId, {
          messages: isLoadMore ? [...newMessages, ...existing.messages] : newMessages,
          hasMore,
          loading: false,
          loadingMore: false,
          error: null,
        });
      });
    } catch (error: unknown) {
      const msg = getErrorMessage(error, "Unknown error");
      console.error("Failed to fetch messages:", msg);
      set((state) => updateCache(state, channelId, {
        loading: false, loadingMore: false, error: isLoadMore ? null : msg,
      }));
    }
  },

  sendMessage: async (channelId, content, podKey) => {
    try {
      const response = await channelApi.sendMessage(channelId, content, podKey);
      const msg = response.message;

      // POST response may lack sender_user — backfill from auth store
      if (!msg.sender_user && msg.sender_user_id) {
        const authUser = useAuthStore.getState().user;
        if (authUser && authUser.id === msg.sender_user_id) {
          msg.sender_user = {
            id: authUser.id,
            username: authUser.username,
            name: authUser.name,
            avatar_url: authUser.avatar_url,
          };
        }
      }

      set((state) => {
        const existing = getCache(state, channelId);
        const idx = existing.messages.findIndex((m) => m.id === msg.id);
        // WebSocket event may arrive before POST response — dedup by replacing
        if (idx >= 0) {
          const updated = [...existing.messages];
          updated[idx] = msg;
          return updateCache(state, channelId, { messages: updated });
        }
        return updateCache(state, channelId, { messages: [...existing.messages, msg] });
      });
      return msg;
    } catch (error: unknown) {
      console.error("Failed to send message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  addMessage: (channelId, message) => {
    set((state) => {
      const existing = getCache(state, channelId);
      const idx = existing.messages.findIndex((m) => m.id === message.id);
      if (idx >= 0) {
        const prev = existing.messages[idx];
        if (!prev.sender_user && message.sender_user) {
          const updated = [...existing.messages];
          updated[idx] = message;
          return updateCache(state, channelId, { messages: updated });
        }
        return {};
      }
      return updateCache(state, channelId, { messages: [...existing.messages, message] });
    });
  },
  editMessage: async (channelId, messageId, content) => {
    try {
      const response = await channelApi.editMessage(channelId, messageId, content);
      set((state) => {
        const existing = getCache(state, channelId);
        return updateCache(state, channelId, {
          messages: existing.messages.map((m) =>
            m.id === messageId
              ? { ...m, body: response.message.body, content: response.message.content, mentions: response.message.mentions, edited_at: response.message.edited_at }
              : m
          ),
        });
      });
    } catch (error: unknown) {
      console.error("Failed to edit message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },
  deleteMessage: async (channelId, messageId) => {
    try {
      await channelApi.deleteMessage(channelId, messageId);
      set((state) => {
        const existing = getCache(state, channelId);
        return updateCache(state, channelId, {
          messages: existing.messages.filter((m) => m.id !== messageId),
        });
      });
    } catch (error: unknown) {
      console.error("Failed to delete message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },
  updateMessage: (channelId, data) => {
    set((state) => {
      const existing = getCache(state, channelId);
      return updateCache(state, channelId, {
        messages: existing.messages.map((m) =>
          m.id === data.id ? { ...m, body: data.body, content: data.content, mentions: data.mentions, edited_at: data.edited_at } : m
        ),
      });
    });
  },
  removeMessage: (channelId, messageId) => {
    set((state) => {
      const existing = getCache(state, channelId);
      return updateCache(state, channelId, {
        messages: existing.messages.filter((m) => m.id !== messageId),
      });
    });
  },
  fetchUnreadCounts: async () => {
    try {
      const response = await channelApi.getUnreadCounts();
      const counts: Record<number, number> = {};
      for (const [key, value] of Object.entries(response.unread || {})) {
        counts[Number(key)] = value;
      }
      set({ unreadCounts: counts });
    } catch (error: unknown) {
      console.error("Failed to fetch unread counts:", getErrorMessage(error, "Unknown error"));
    }
  },
  markRead: async (channelId, messageId) => {
    try {
      await channelApi.markRead(channelId, messageId);
      set((state) => {
        const counts = { ...state.unreadCounts };
        delete counts[channelId];
        return { unreadCounts: counts };
      });
    } catch (error: unknown) {
      console.error("Failed to mark channel as read:", getErrorMessage(error, "Unknown error"));
    }
  },

  muteChannel: async (channelId, muted) => {
    try {
      await channelApi.mute(channelId, muted);
    } catch (error: unknown) {
      console.error("Failed to update mute setting:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  incrementUnread: (channelId) => {
    set((state) => ({
      unreadCounts: {
        ...state.unreadCounts,
        [channelId]: (state.unreadCounts[channelId] || 0) + 1,
      },
    }));
  },

  clearChannelUnread: (channelId) => {
    set((state) => {
      if (!(channelId in state.unreadCounts)) return {};
      const counts = { ...state.unreadCounts };
      delete counts[channelId];
      return { unreadCounts: counts };
    });
  },

  totalUnreadCount: () => {
    return Object.values(get().unreadCounts).reduce((sum, c) => sum + c, 0);
  },
}));
