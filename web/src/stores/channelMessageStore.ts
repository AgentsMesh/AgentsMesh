import { create } from "zustand";
import { channelApi, ChannelMessage } from "@/lib/api";
import type { MentionPayload } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";
import { useAuthStore } from "./auth";
import { useChannelStore } from "./channelStore";

interface ChannelMessageState {
  // State
  messages: ChannelMessage[];
  messagesLoading: boolean;
  unreadCounts: Record<number, number>;

  // Message CRUD
  fetchMessages: (channelId: number, limit?: number, beforeId?: number) => Promise<void>;
  sendMessage: (
    channelId: number,
    content: string,
    podKey?: string,
    mentions?: MentionPayload[]
  ) => Promise<ChannelMessage>;
  addMessage: (message: ChannelMessage) => void;
  editMessage: (channelId: number, messageId: number, content: string) => Promise<void>;
  deleteMessage: (channelId: number, messageId: number) => Promise<void>;
  updateMessage: (data: { id: number; content: string; edited_at: string }) => void;
  removeMessage: (messageId: number) => void;

  // Unread / read state
  fetchUnreadCounts: () => Promise<void>;
  markRead: (channelId: number, messageId: number) => Promise<void>;
  muteChannel: (channelId: number, muted: boolean) => Promise<void>;
  incrementUnread: (channelId: number) => void;
}

export const useChannelMessageStore = create<ChannelMessageState>((set, get) => ({
  messages: [],
  messagesLoading: false,
  unreadCounts: {},

  fetchMessages: async (channelId, limit = 50, beforeId) => {
    set({ messagesLoading: true });
    try {
      const response = await channelApi.getMessages(channelId, limit, beforeId);
      set((state) => ({
        messages:
          beforeId === undefined
            ? response.messages || []
            : [...(response.messages || []), ...state.messages],
        messagesLoading: false,
      }));
    } catch (error: unknown) {
      console.error("Failed to fetch messages:", getErrorMessage(error, "Unknown error"));
      set({ messagesLoading: false });
    }
  },

  sendMessage: async (channelId, content, podKey, mentions) => {
    try {
      const response = await channelApi.sendMessage(channelId, content, podKey, undefined, mentions);
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
        const idx = state.messages.findIndex((m) => m.id === msg.id);
        if (idx >= 0) {
          const updated = [...state.messages];
          updated[idx] = msg;
          return { messages: updated };
        }
        return { messages: [...state.messages, msg] };
      });
      return msg;
    } catch (error: unknown) {
      console.error("Failed to send message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  addMessage: (message) => {
    set((state) => {
      const idx = state.messages.findIndex((m) => m.id === message.id);
      if (idx >= 0) {
        // Merge: prefer the version with richer sender info
        const existing = state.messages[idx];
        if (!existing.sender_user && message.sender_user) {
          const updated = [...state.messages];
          updated[idx] = message;
          return { messages: updated };
        }
        return {}; // Already have complete data, skip
      }
      return { messages: [...state.messages, message] };
    });
    // Auto mark-as-read for current channel when new message is added
    const currentChannel = useChannelStore.getState().currentChannel;
    if (currentChannel && currentChannel.id === message.channel_id) {
      channelApi.markRead(message.channel_id, message.id).catch(() => {});
    }
  },

  editMessage: async (channelId, messageId, content) => {
    try {
      const response = await channelApi.editMessage(channelId, messageId, content);
      set((state) => ({
        messages: state.messages.map((m) =>
          m.id === messageId
            ? { ...m, content: response.message.content, edited_at: response.message.edited_at }
            : m
        ),
      }));
    } catch (error: unknown) {
      console.error("Failed to edit message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  deleteMessage: async (channelId, messageId) => {
    try {
      await channelApi.deleteMessage(channelId, messageId);
      set((state) => ({
        messages: state.messages.filter((m) => m.id !== messageId),
      }));
    } catch (error: unknown) {
      console.error("Failed to delete message:", getErrorMessage(error, "Unknown error"));
      throw error;
    }
  },

  updateMessage: (data) => {
    set((state) => ({
      messages: state.messages.map((m) =>
        m.id === data.id
          ? { ...m, content: data.content, edited_at: data.edited_at }
          : m
      ),
    }));
  },

  removeMessage: (messageId) => {
    set((state) => ({
      messages: state.messages.filter((m) => m.id !== messageId),
    }));
  },

  fetchUnreadCounts: async () => {
    try {
      const response = await channelApi.getUnreadCounts();
      // Backend returns { unread: { "42": 3, "55": 17 } } with string keys
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
}));
