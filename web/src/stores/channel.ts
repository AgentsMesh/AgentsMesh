export { useChannelStore, useChannels, useCurrentChannel, getLastMessage, type Channel, type ChannelLastMessage } from "./channelStore";
export {
  useChannelMessageStore,
  EMPTY_CACHE,
  useUnreadCounts,
  useUnreadCount,
  useTotalUnreadCount,
} from "./channelMessageStore";
export type { ChannelMessageCache, ChannelMessageState } from "./channelMessageTypes";

import { reconnectRegistry } from "@/lib/realtime";
import { useChannelMessageStore } from "./channelMessageStore";

reconnectRegistry.register({
  name: "channel:unread",
  fn: () => useChannelMessageStore.getState().fetchUnreadCounts?.(),
  priority: "low",
});
