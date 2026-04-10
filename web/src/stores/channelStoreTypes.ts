import type { ChannelData } from "@/lib/api/channel";

/** Channel entity in store — extends API response with rich associations */
export interface Channel extends ChannelData {
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
    alias?: string;
    status: string;
    agent?: {
      name: string;
    };
  }>;
}

/** Channel store state and actions */
export interface ChannelState {
  channels: Channel[];
  currentChannel: Channel | null;
  loading: boolean;
  channelLoading: boolean;
  error: string | null;

  selectedChannelId: number | null;
  searchQuery: string;
  showArchived: boolean;

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
    visibility?: "public" | "private";
    memberIds?: number[];
  }) => Promise<Channel>;
  updateChannel: (
    id: number,
    data: Partial<{ name: string; description: string; document: string }>
  ) => Promise<Channel>;
  archiveChannel: (id: number) => Promise<void>;
  unarchiveChannel: (id: number) => Promise<void>;
  joinChannel: (channelId: number, podKey: string) => Promise<void>;
  leaveChannel: (channelId: number, podKey: string) => Promise<void>;
  joinUserChannel: (channelId: number) => Promise<void>;
  leaveUserChannel: (channelId: number) => Promise<void>;
  inviteMembers: (channelId: number, userIds: number[]) => Promise<void>;
  patchChannelMemberCount: (channelId: number, delta: number) => void;
  setCurrentChannel: (channel: Channel | null) => void;
  clearError: () => void;
}
