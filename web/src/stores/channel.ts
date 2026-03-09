/**
 * Barrel re-export for channel stores.
 *
 * Channel state is split into two focused stores:
 * - channelStore: channel CRUD, UI state (selectedChannelId, searchQuery, etc.)
 * - channelMessageStore: messages, unread counts, read state
 */
export { useChannelStore, type Channel } from "./channelStore";
export { useChannelMessageStore } from "./channelMessageStore";
