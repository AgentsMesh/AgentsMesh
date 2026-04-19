// Legacy barrel kept for backwards-compat imports.
// The authoritative types now live in ./channel (ChannelData, ChannelMessage)
// and ./channel-message-types (MessageContent, MessageMentions, …).
export type { ChannelData, ChannelMessage } from "./channel";
export type { MessageContent, MessageMentions, InlineElement, Block } from "./channel-message-types";

// Simple mention payload used by the WASM-based send-message path in the
// channel service wrappers. The structured `MessageMentions` above is the
// server-side schema; this one is the client hint.
export interface MentionPayload {
  type: "user" | "pod";
  id: string;
}
