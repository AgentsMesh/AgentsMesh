import { create } from "zustand";
import { channelApi, ChannelMessage } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";
import { useAuthStore } from "./auth";

export interface Channel {
  id: number;
  organization_id: number;
  name: string;
  description?: string;
  document?: string;
  is_archived: boolean;
  created_at: string;
  updated_at: string;
  repository?: {
    id: number;
    name: string;
  };
  ticket?: {
    id: number;
    slug: string;
    title: string;
  };
  pods?: Array<{
    pod_key: string;
    status: string;
    agent_type?: {
      name: string;
    };
  }>;
}

interface ChannelState {
  // State
  channels: Channel[];
  currentChannel: Channel | null;
  messages: ChannelMessage[];
  loading: boolean;
  channelLoading: boolean;
  messagesLoading: boolean;
  error: string | null;

  // Channels Tab state
  selectedChannelId: number | null;
  searchQuery: string;
  showArchived: boolean;

  // Actions
  setSelectedChannelId: (id: number | null) => void;
  setSearchQuery: (query: string) => void;
  setShowArchived: (show: boolean) => void;
  fetchChannels: (filters?: {
    includeArchived?: boolean;
  }) => Promise<void>;
  fetchChannel: (id: number) => Promise<void>;
  createChannel: (data: {
    name: string;
    description?: string;
    document?: string;
    repositoryId?: number;
    ticketSlug?: string;
  }) => Promise<Channel>;
  updateChannel: (
    id: number,
    data: Partial<{
      name: string;
      description: string;
      document: string;
    }>
  ) => Promise<Channel>;
  archiveChannel: (id: number) => Promise<void>;
  unarchiveChannel: (id: number) => Promise<void>;
  fetchMessages: (channelId: number, limit?: number, offset?: number) => Promise<void>;
  sendMessage: (
    channelId: number,
    content: string,
    podKey?: string
  ) => Promise<ChannelMessage>;
  joinChannel: (channelId: number, podKey: string) => Promise<void>;
  leaveChannel: (channelId: number, podKey: string) => Promise<void>;
  setCurrentChannel: (channel: Channel | null) => void;
  addMessage: (message: ChannelMessage) => void;
  clearError: () => void;
}

export const useChannelStore = create<ChannelState>((set, get) => ({
  channels: [],
  currentChannel: null,
  messages: [],
  loading: false,
  channelLoading: false,
  messagesLoading: false,
  error: null,

  // Channels Tab state
  selectedChannelId: null,
  searchQuery: "",
  showArchived: false,

  setSelectedChannelId: (id) => {
    set({ selectedChannelId: id });
    if (id !== null) {
      get().fetchChannel(id);
      get().fetchMessages(id);
    } else {
      set({ currentChannel: null, messages: [] });
    }
  },

  setSearchQuery: (query) => set({ searchQuery: query }),
  setShowArchived: (show) => set({ showArchived: show }),

  fetchChannels: async (filters) => {
    set({ error: null });
    try {
      // Convert camelCase to snake_case for API
      const apiFilters = filters ? {
        include_archived: filters.includeArchived,
      } : undefined;
      const response = await channelApi.list(apiFilters);
      set({ channels: response.channels || [] });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch channels"),
      });
    }
  },

  fetchChannel: async (id) => {
    set({ channelLoading: true, error: null });
    try {
      const response = await channelApi.get(id);
      set({ currentChannel: response.channel, channelLoading: false });
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch channel"),
        channelLoading: false,
      });
    }
  },

  createChannel: async (data) => {
    set({ error: null });
    try {
      // Convert camelCase to snake_case for API
      const apiData = {
        name: data.name,
        description: data.description,
        document: data.document,
        repository_id: data.repositoryId,
        ticket_slug: data.ticketSlug,
      };
      const response = await channelApi.create(apiData);
      set((state) => ({
        channels: [response.channel, ...state.channels],
      }));
      return response.channel;
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to create channel"),
      });
      throw error;
    }
  },

  updateChannel: async (id, data) => {
    try {
      const response = await channelApi.update(id, data);
      set((state) => ({
        channels: state.channels.map((c) => (c.id === id ? response.channel : c)),
        currentChannel:
          state.currentChannel?.id === id ? response.channel : state.currentChannel,
      }));
      return response.channel;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to update channel") });
      throw error;
    }
  },

  archiveChannel: async (id) => {
    try {
      await channelApi.archive(id);
      set((state) => ({
        channels: state.channels.map((c) =>
          c.id === id ? { ...c, is_archived: true } : c
        ),
        currentChannel:
          state.currentChannel?.id === id
            ? { ...state.currentChannel, is_archived: true }
            : state.currentChannel,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to archive channel") });
      throw error;
    }
  },

  unarchiveChannel: async (id) => {
    try {
      await channelApi.unarchive(id);
      set((state) => ({
        channels: state.channels.map((c) =>
          c.id === id ? { ...c, is_archived: false } : c
        ),
        currentChannel:
          state.currentChannel?.id === id
            ? { ...state.currentChannel, is_archived: false }
            : state.currentChannel,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to unarchive channel") });
      throw error;
    }
  },

  fetchMessages: async (channelId, limit = 50, offset = 0) => {
    set({ messagesLoading: true, error: null });
    try {
      const response = await channelApi.getMessages(channelId, limit, offset);
      set((state) => ({
        messages:
          offset === 0
            ? response.messages || []
            : [...state.messages, ...(response.messages || [])],
        messagesLoading: false,
      }));
    } catch (error: unknown) {
      set({
        error: getErrorMessage(error, "Failed to fetch messages"),
        messagesLoading: false,
      });
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
      set({ error: getErrorMessage(error, "Failed to send message") });
      throw error;
    }
  },

  joinChannel: async (channelId, podKey) => {
    try {
      await channelApi.joinPod(channelId, podKey);
      // Refresh channel to get updated pod list
      const response = await channelApi.get(channelId);
      set((state) => ({
        channels: state.channels.map((c) => (c.id === channelId ? response.channel : c)),
        currentChannel:
          state.currentChannel?.id === channelId ? response.channel : state.currentChannel,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to join channel") });
      throw error;
    }
  },

  leaveChannel: async (channelId, podKey) => {
    try {
      await channelApi.leavePod(channelId, podKey);
      // Refresh channel to get updated pod list
      const response = await channelApi.get(channelId);
      set((state) => ({
        channels: state.channels.map((c) => (c.id === channelId ? response.channel : c)),
        currentChannel:
          state.currentChannel?.id === channelId ? response.channel : state.currentChannel,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to leave channel") });
      throw error;
    }
  },

  setCurrentChannel: (channel) => {
    set({ currentChannel: channel, messages: [] });
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
  },

  clearError: () => {
    set({ error: null });
  },
}));
