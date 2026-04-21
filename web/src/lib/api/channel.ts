import type { MessageContent, MessageMentions } from "./channel-message-types";
import { getChannelService } from "@/lib/wasm-core";

export type { MessageContent, MessageMentions } from "./channel-message-types";
export type { InlineElement, Block } from "./channel-message-types";

export interface ChannelData {
  id: number;
  organization_id: number;
  name: string;
  description?: string;
  document?: string;
  repository_id?: number;
  ticket_id?: number;
  ticket_slug?: string;
  created_by_pod?: string;
  created_by_user_id?: number;
  visibility: "public" | "private";
  is_archived: boolean;
  is_member: boolean;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface ChannelMessage {
  id: number;
  channel_id: number;
  sender_pod?: string;
  sender_user_id?: number;
  message_type: string;
  body: string;
  content?: MessageContent;
  mentions?: MessageMentions;
  reply_to?: number;
  edited_at?: string;
  is_deleted?: boolean;
  created_at: string;
  sender_pod_info?: {
    pod_key: string;
    alias?: string;
    agent?: { name: string };
    ticket?: { slug?: string; title?: string };
    loop?: { name?: string };
    title?: string;
  };
  sender_user?: {
    id: number;
    username: string;
    name?: string;
    avatar_url?: string;
  };
}

// All channel operations route through Rust (WASM SSOT).
// list/get/create/getMessages/sendMessage are consumed directly by stores (channelStore/channelMessageStore)
// via svc().xxx — they are not re-exported here to avoid duplicate wrappers.
export const channelApi = {
  update: async (id: number, data: { name?: string; description?: string; document?: string }) => {
    const json = await getChannelService().update_channel(BigInt(id), JSON.stringify(data));
    return { channel: JSON.parse(json) as ChannelData };
  },

  archive: async (id: number) => {
    await getChannelService().archive_channel(BigInt(id));
    return { message: "ok" };
  },

  unarchive: async (id: number) => {
    await getChannelService().unarchive_channel(BigInt(id));
    return { message: "ok" };
  },

  searchMessages: async (id: number, q: string, limit = 20) => {
    const json = await getChannelService().search_channel_messages(BigInt(id), q, limit);
    return JSON.parse(json) as { messages: ChannelMessage[] };
  },

  getPods: async (id: number) => {
    const json = await getChannelService().get_channel_pods(BigInt(id));
    return JSON.parse(json) as {
      pods: Array<{ id: number; pod_key: string; alias?: string; status: string; agent_status: string }>;
      total: number;
    };
  },

  joinPod: async (id: number, podKey: string) => {
    await getChannelService().join_channel(BigInt(id), podKey);
    return { message: "ok" };
  },

  leavePod: async (id: number, podKey: string) => {
    await getChannelService().leave_channel(BigInt(id), podKey);
    return { message: "ok" };
  },

  inviteMembers: async (id: number, userIds: number[]) => {
    await getChannelService().invite_channel_members(BigInt(id), JSON.stringify(userIds));
    return { message: "ok" };
  },

  removeMember: async (id: number, userId: number) => {
    await getChannelService().remove_channel_member(BigInt(id), BigInt(userId));
    return { message: "ok" };
  },

  listMembers: async (id: number) => {
    const json = await getChannelService().fetch_channel_members(BigInt(id));
    return JSON.parse(json) as {
      members: Array<{ channel_id: number; user_id: number; role: string; is_muted: boolean; joined_at: string }>;
      total: number;
    };
  },
};
