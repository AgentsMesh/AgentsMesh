import { create } from "zustand";
import { channelApi } from "@/lib/api";
import { getErrorMessage } from "@/lib/utils";
import { useChannelMessageStore } from "./channelMessageStore";
import type { Channel, ChannelState } from "./channelStoreTypes";

export const useChannelStore = create<ChannelState>((set, get) => {
  /** Update a channel in both lists and currentChannel */
  const patchChannel = (id: number, updater: (ch: Channel) => Channel) => {
    set((state) => ({
      channels: state.channels.map((c) => (c.id === id ? updater(c) : c)),
      currentChannel: state.currentChannel?.id === id ? updater(state.currentChannel) : state.currentChannel,
    }));
  };

  return {
  channels: [],
  currentChannel: null,
  loading: false,
  channelLoading: false,
  error: null,
  selectedChannelId: null,
  searchQuery: "",
  showArchived: false,

  setSelectedChannelId: (id) => {
    set({ selectedChannelId: id });
    if (id !== null) {
      get().fetchChannel(id);
      useChannelMessageStore.getState().clearChannelUnread(id);
    } else {
      set({ currentChannel: null });
    }
  },

  setSearchQuery: (query) => set({ searchQuery: query }),
  setShowArchived: (show) => set({ showArchived: show }),

  fetchChannels: async (filters) => {
    set({ error: null });
    try {
      const apiFilters = filters ? { include_archived: filters.includeArchived } : undefined;
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
      set({ error: getErrorMessage(error, "Failed to fetch channel"), channelLoading: false });
    }
  },

  createChannel: async (data) => {
    set({ error: null });
    try {
      const response = await channelApi.create({
        name: data.name, description: data.description, document: data.document,
        repository_id: data.repositoryId, ticket_slug: data.ticketSlug,
        visibility: data.visibility, member_ids: data.memberIds,
      });
      set((state) => ({ channels: [response.channel, ...state.channels] }));
      return response.channel;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to create channel") });
      throw error;
    }
  },

  updateChannel: async (id, data) => {
    try {
      const response = await channelApi.update(id, data);
      patchChannel(id, () => response.channel);
      return response.channel;
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to update channel") });
      throw error;
    }
  },

  archiveChannel: async (id) => {
    try {
      await channelApi.archive(id);
      patchChannel(id, (ch) => ({ ...ch, is_archived: true }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to archive channel") });
      throw error;
    }
  },

  unarchiveChannel: async (id) => {
    try {
      await channelApi.unarchive(id);
      patchChannel(id, (ch) => ({ ...ch, is_archived: false }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to unarchive channel") });
      throw error;
    }
  },

  joinChannel: async (channelId, podKey) => {
    try {
      await channelApi.joinPod(channelId, podKey);
      const response = await channelApi.get(channelId);
      patchChannel(channelId, () => response.channel);
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to join channel") });
      throw error;
    }
  },

  leaveChannel: async (channelId, podKey) => {
    try {
      await channelApi.leavePod(channelId, podKey);
      const response = await channelApi.get(channelId);
      patchChannel(channelId, () => response.channel);
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to leave channel") });
      throw error;
    }
  },

  patchChannelMemberCount: (channelId, delta) => {
    patchChannel(channelId, (ch) => ({
      ...ch,
      member_count: Math.max(0, ch.member_count + delta),
    }));
  },

  setCurrentChannel: (channel) => set({ currentChannel: channel }),
  clearError: () => set({ error: null }),

  joinUserChannel: async (channelId) => {
    try {
      await channelApi.joinChannel(channelId);
      patchChannel(channelId, (ch) => ({ ...ch, is_member: true, member_count: ch.member_count + 1 }));
      // Fetch messages now that user has access
      useChannelMessageStore.getState().fetchMessages(channelId);
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to join channel") });
      throw error;
    }
  },

  leaveUserChannel: async (channelId) => {
    try {
      await channelApi.leaveChannel(channelId);
      patchChannel(channelId, (ch) => ({ ...ch, is_member: false, member_count: Math.max(0, ch.member_count - 1) }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to leave channel") });
      throw error;
    }
  },

  inviteMembers: async (channelId, userIds) => {
    try {
      await channelApi.inviteMembers(channelId, userIds);
      patchChannel(channelId, (ch) => ({ ...ch, member_count: ch.member_count + userIds.length }));
    } catch (error: unknown) {
      set({ error: getErrorMessage(error, "Failed to invite members") });
      throw error;
    }
  },
}; });
