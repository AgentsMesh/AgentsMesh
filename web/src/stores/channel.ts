/**
 * Barrel re-export for channel stores.
 *
 * Channel state is split into two focused stores:
 * - channelStore: channel CRUD, UI state (selectedChannelId, searchQuery, etc.)
 * - channelMessageStore: messages, unread counts, read state
 */
export { useChannelStore } from "./channelStore";
export type { Channel } from "./channelStoreTypes";
export { useChannelMessageStore, EMPTY_CACHE, type ChannelMessageCache } from "./channelMessageStore";
export type { ChannelMessageState } from "./channelMessageTypes";
