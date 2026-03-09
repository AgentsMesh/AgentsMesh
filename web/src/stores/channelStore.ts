import { create } from "zustand";
import { channelApi } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";
import { useChannelMessageStore } from "./channelMessageStore";

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
  loading: boolean;
  channelLoading: boolean;
  error: string | null;

  // Channels Tab state
  selectedChannelId: number | null;
  searchQuery: string;
  showArchived: boolean;

  // Actions
  setSelectedChannelId: (id: number | null) => void;
  setSearchQuery: (query: string) => void;
  setShowArchived: (show: boolean) => void;
  fetchChannels: (filters?: { includeArchived?: boolean }) => Promise<void>;
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
    data: Partial<{ name: string; description: string; document: string }>
  ) => Promise<Channel>;
  archiveChannel: (id: number) => Promise<void>;
  unarchiveChannel: (id: number) => Promise<void>;
  joinChannel: (channelId: number, podKey: string) => Promise<void>;
  leaveChannel: (channelId: number, podKey: string) => Promise<void>;
  setCurrentChannel: (channel: Channel | null) => void;
  clearError: () => void;
}

export const useChannelStore = create<ChannelState>((set, get) => ({
  channels: [],
  currentChannel: null,
  loading: false,
  channelLoading: false,
  error: null,

  // Channels Tab state
  selectedChannelId: null,
  searchQuery: "",
  showArchived: false,

  setSelectedChannelId: (id) => {
    set({ selectedChannelId: id });
    if (id !== null) {
      get().fetchChannel(id);
      useChannelMessageStore.getState().fetchMessages(id);
      // Clear unread count for selected channel
      const msgStore = useChannelMessageStore.getState();
      const counts = { ...msgStore.unreadCounts };
      delete counts[id];
      useChannelMessageStore.setState({ unreadCounts: counts });
    } else {
      set({ currentChannel: null });
      useChannelMessageStore.setState({ messages: [] });
    }
  },

  setSearchQuery: (query) => set({ searchQuery: query }),
  setShowArchived: (show) => set({ showArchived: show }),

  fetchChannels: async (filters) => {
    set({ error: null });
    try {
      const apiFilters = filters
        ? { include_archived: filters.includeArchived }
        : undefined;
      const response = await channelApi.list(apiFilters);
      set({ channels: response.channels || [] });
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to fetch channels") });
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
      set({ error: getErrorMessage(error, "Failed to create channel") });
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

  joinChannel: async (channelId, podKey) => {
    try {
      await channelApi.joinPod(channelId, podKey);
      const response = await channelApi.get(channelId);
      set((state) => ({
        channels: state.channels.map((c) =>
          c.id === channelId ? response.channel : c
        ),
        currentChannel:
          state.currentChannel?.id === channelId
            ? response.channel
            : state.currentChannel,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to join channel") });
      throw error;
    }
  },

  leaveChannel: async (channelId, podKey) => {
    try {
      await channelApi.leavePod(channelId, podKey);
      const response = await channelApi.get(channelId);
      set((state) => ({
        channels: state.channels.map((c) =>
          c.id === channelId ? response.channel : c
        ),
        currentChannel:
          state.currentChannel?.id === channelId
            ? response.channel
            : state.currentChannel,
      }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to leave channel") });
      throw error;
    }
  },

  setCurrentChannel: (channel) => {
    set({ currentChannel: channel });
    useChannelMessageStore.setState({ messages: [] });
  },

  clearError: () => set({ error: null }),
}));
