import type { ChannelMessage, MessageContent, MessageMentions, MessageSendPayload, MessageEditPayload } from "@/lib/api";

export interface ChannelMessageCache {
  loading: boolean;
  loadingMore: boolean;
  error: string | null;
}

export const EMPTY_CACHE: ChannelMessageCache = {
  loading: false,
  loadingMore: false,
  error: null,
};

export interface ChannelMessageState {
  cache: Record<number, ChannelMessageCache>;
  _messagesTick: number;
  _unreadTick: number;

  fetchMessages: (channelId: number, limit?: number, beforeId?: number) => Promise<void>;
  sendMessage: (
    channelId: number,
    payload: MessageSendPayload,
    podKey?: string,
  ) => Promise<ChannelMessage>;
  addMessage: (channelId: number, message: ChannelMessage) => void;
  editMessage: (channelId: number, messageId: number, payload: MessageEditPayload) => Promise<void>;
  deleteMessage: (channelId: number, messageId: number) => Promise<void>;
  updateMessage: (channelId: number, data: { id: number; body: string; content?: MessageContent; mentions?: MessageMentions; edited_at: string }) => void;
  removeMessage: (channelId: number, messageId: number) => void;

  fetchUnreadCounts: () => Promise<void>;
  markRead: (channelId: number, messageId: number) => Promise<void>;
  muteChannel: (channelId: number, muted: boolean) => Promise<void>;
  incrementUnread: (channelId: number) => void;
  clearChannelUnread: (channelId: number) => void;
}

export function getCache(state: ChannelMessageState, channelId: number): ChannelMessageCache {
  return state.cache[channelId] ?? EMPTY_CACHE;
}

const MAX_CACHED_CHANNELS = 20;

export function updateCache(
  state: ChannelMessageState,
  channelId: number,
  patch: Partial<ChannelMessageCache>
): { cache: Record<number, ChannelMessageCache> } {
  const newCache = {
    ...state.cache,
    [channelId]: { ...getCache(state, channelId), ...patch },
  };

  const keys = Object.keys(newCache).map(Number);
  if (keys.length > MAX_CACHED_CHANNELS) {
    const toEvict = keys
      .filter((k) => k !== channelId)
      .slice(0, keys.length - MAX_CACHED_CHANNELS);
    for (const k of toEvict) {
      delete newCache[k];
    }
  }

  return { cache: newCache };
}
