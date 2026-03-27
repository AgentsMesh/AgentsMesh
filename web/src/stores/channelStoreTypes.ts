/** Channel entity from backend */
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
    alias?: string;
    status: string;
    agent?: {
      name: string;
    };
  }>;
}

/** Channel store state and actions */
export interface ChannelState {
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
