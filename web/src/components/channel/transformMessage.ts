import type { ChannelMessage } from "@/lib/api";
import type { TransformedMessage } from "./types";

export function transformMessage(msg: ChannelMessage): TransformedMessage {
  return {
    id: msg.id,
    body: msg.body,
    content: msg.content,
    messageType: msg.message_type,
    mentions: msg.mentions,
    editedAt: msg.edited_at,
    createdAt: msg.created_at,
    pod: msg.sender_pod_info
      ? {
          podKey: msg.sender_pod_info.pod_key,
          alias: msg.sender_pod_info.alias,
          agent: msg.sender_pod_info.agent
            ? { name: msg.sender_pod_info.agent.name }
            : undefined,
        }
      : msg.sender_pod
      ? { podKey: msg.sender_pod }
      : undefined,
    user: msg.sender_user
      ? {
          id: msg.sender_user.id,
          username: msg.sender_user.username,
          name: msg.sender_user.name,
          avatarUrl: msg.sender_user.avatar_url,
        }
      : undefined,
  };
}
