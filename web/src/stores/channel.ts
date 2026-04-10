export { useChannelStore } from "./channelStore";
export type { Channel } from "./channelStoreTypes";
export { useChannelMessageStore, EMPTY_CACHE, type ChannelMessageCache } from "./channelMessageStore";
export type { ChannelMessageState } from "./channelMessageTypes";

import { reconnectRegistry } from "@/lib/realtime";
import { useChannelMessageStore } from "./channelMessageStore";

reconnectRegistry.register({
  name: "channel:unread",
  fn: () => useChannelMessageStore.getState().fetchUnreadCounts?.(),
  priority: "low",
});
