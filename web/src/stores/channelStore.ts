import { create } from "zustand";
import { useMemo } from "react";
import { getErrorMessage } from "@/lib/utils";
import { getChannelService, getApiClient } from "@/lib/wasm-core";

export interface Channel {
  id: number; organization_id?: number; name: string; description?: string; document?: string;
  is_archived: boolean;
  visibility?: "public" | "private";
  is_member?: boolean;
  member_count: number;
  created_at?: string; updated_at?: string;
  repository?: { id: number; name: string };
  ticket?: { id: number; slug: string; title: string };
  pods?: Array<{ pod_key: string; alias?: string; status: string; agent?: { name: string } }>;
}

const svc = () => getChannelService();
const bump = () => useChannelStore.setState((s) => ({ _tick: s._tick + 1 }));

export function useChannels(): Channel[] {
  const tick = useChannelStore((s) => s._tick);
  return useMemo(() => JSON.parse(svc().channels_json()) as Channel[], [tick]);
}

export interface ChannelLastMessage {
  sender_name: string;
  content_preview: string;
  message_type?: string;
  timestamp: string;
}

/** Read the cached last-message preview for a channel (from WASM `last_messages` map). */
export function getLastMessage(channelId: number): ChannelLastMessage | null {
  const raw = svc().get_last_message_json(BigInt(channelId));
  if (!raw) return null;
  try {
    return typeof raw === "string" ? (JSON.parse(raw) as ChannelLastMessage) : (raw as ChannelLastMessage);
  } catch {
    return null;
  }
}

export function useCurrentChannel(): Channel | null {
  const tick = useChannelStore((s) => s._tick);
  return useMemo(() => {
    const v = svc().current_channel_json();
    return v ? (typeof v === "string" ? JSON.parse(v) : v) : null;
  }, [tick]);
}

export interface ChannelMember {
  channel_id: number;
  user_id: number;
  role: string;
  is_muted: boolean;
  joined_at: string;
}

/** Members of a given channel. Rust ChannelService caches the list per channel
 *  in state; the hook re-reads whenever `_tick` bumps (fetch / invite / remove). */
export function useChannelMembers(channelId: number | null | undefined): ChannelMember[] {
  const tick = useChannelStore((s) => s._tick);
  return useMemo(() => {
    if (channelId == null) return [];
    try {
      return JSON.parse(svc().channel_members_json(BigInt(channelId))) as ChannelMember[];
    } catch {
      return [];
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, channelId]);
}

interface ChannelState {
  _tick: number; loading: boolean; channelLoading: boolean;
  error: string | null; selectedChannelId: number | null; searchQuery: string; showArchived: boolean;
  currentChannel: Channel | null;
  setSelectedChannelId: (id: number | null) => void; setSearchQuery: (q: string) => void; setShowArchived: (s: boolean) => void;
  fetchChannels: (f?: { includeArchived?: boolean }) => Promise<void>; fetchChannel: (id: number) => Promise<void>;
  createChannel: (d: {
    name: string; description?: string; document?: string;
    repositoryId?: number; ticketSlug?: string;
    visibility?: "public" | "private"; memberIds?: number[];
  }) => Promise<Channel>;
  updateChannel: (id: number, d: Partial<{ name: string; description: string; document: string }>) => Promise<Channel>;
  archiveChannel: (id: number) => Promise<void>; unarchiveChannel: (id: number) => Promise<void>;
  joinChannel: (channelId: number, podKey: string) => Promise<void>; leaveChannel: (channelId: number, podKey: string) => Promise<void>;
  joinUserChannel: (channelId: number) => Promise<void>;
  leaveUserChannel: (channelId: number) => Promise<void>;
  inviteMembers: (channelId: number, userIds: number[]) => Promise<void>;
  patchChannelMemberCount: (channelId: number, delta: number) => void;
  setCurrentChannel: (ch: Channel | null) => void; clearError: () => void;
}

// TODO(wasm): move these to dedicated ChannelService methods once the core crate
// adds invite/join/leave APIs. For now they use the shared ApiClient directly.
async function orgScopedPost(path: string, body?: unknown): Promise<unknown> {
  return await getApiClient().post(path, body ?? {});
}

export const useChannelStore = create<ChannelState>((set, get) => ({
  _tick: 0, loading: false, channelLoading: false,
  error: null, selectedChannelId: null, searchQuery: "", showArchived: false,
  currentChannel: null,

  setSelectedChannelId: (id) => {
    set({ selectedChannelId: id });
    if (id !== null) {
      svc().select_channel(BigInt(id));
      bump();
      get().fetchChannel(id);
    } else {
      svc().select_channel(undefined as unknown as bigint);
      bump();
    }
  },

  setSearchQuery: (query) => set({ searchQuery: query }),
  setShowArchived: (show) => set({ showArchived: show }),

  fetchChannels: async (filters) => {
    set({ error: null });
    try {
      await svc().fetch_channels(filters?.includeArchived);
      bump();
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch channels") }); }
  },

  fetchChannel: async (id) => {
    set({ channelLoading: true, error: null });
    try {
      await svc().fetch_channel(BigInt(id));
      set({ channelLoading: false, _tick: get()._tick + 1 });
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to fetch channel"), channelLoading: false }); }
  },

  createChannel: async (data) => {
    set({ error: null });
    try {
      const json = await svc().create_channel(JSON.stringify({
        name: data.name, description: data.description, document: data.document,
        repository_id: data.repositoryId, ticket_slug: data.ticketSlug,
        visibility: data.visibility, member_ids: data.memberIds,
      }));
      const channel = JSON.parse(json) as Channel;
      bump();
      return channel;
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to create channel") }); throw e; }
  },

  updateChannel: async (id, data) => {
    try {
      const json = await svc().update_channel(BigInt(id), JSON.stringify(data));
      const channel = JSON.parse(json) as Channel;
      bump();
      return channel;
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to update channel") }); throw e; }
  },

  archiveChannel: async (id) => {
    try { await svc().archive_channel(BigInt(id)); bump(); }
    catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to archive channel") }); throw e; }
  },

  unarchiveChannel: async (id) => {
    try { await svc().unarchive_channel(BigInt(id)); bump(); }
    catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to unarchive channel") }); throw e; }
  },

  joinChannel: async (channelId, podKey) => {
    try { await svc().join_channel(BigInt(channelId), podKey); bump(); }
    catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to join channel") }); throw e; }
  },

  leaveChannel: async (channelId, podKey) => {
    try { await svc().leave_channel(BigInt(channelId), podKey); bump(); }
    catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to leave channel") }); throw e; }
  },

  joinUserChannel: async (channelId) => {
    try {
      await orgScopedPost(`/api/v1/channels/${channelId}/join`);
      get().patchChannelMemberCount(channelId, 1);
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to join channel") }); throw e; }
  },

  leaveUserChannel: async (channelId) => {
    try {
      await orgScopedPost(`/api/v1/channels/${channelId}/leave`);
      get().patchChannelMemberCount(channelId, -1);
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to leave channel") }); throw e; }
  },

  inviteMembers: async (channelId, userIds) => {
    try {
      await orgScopedPost(`/api/v1/channels/${channelId}/members`, { user_ids: userIds });
      get().patchChannelMemberCount(channelId, userIds.length);
    } catch (e: unknown) { set({ error: getErrorMessage(e, "Failed to invite members") }); throw e; }
  },

  patchChannelMemberCount: (channelId, delta) => {
    const current = JSON.parse(svc().get_channel_json(BigInt(channelId)) as string) as Channel | null;
    if (!current) return;
    const next = { ...current, member_count: Math.max(0, current.member_count + delta) };
    svc().update_channel_local(BigInt(channelId), JSON.stringify(next));
    bump();
  },

  setCurrentChannel: (channel) => {
    svc().set_current_channel(channel ? BigInt(channel.id) : null);
    set({ currentChannel: channel });
    bump();
  },

  clearError: () => set({ error: null }),
}));
