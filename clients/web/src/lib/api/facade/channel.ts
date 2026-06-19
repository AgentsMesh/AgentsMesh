import type { MessageContent, MessageMentions } from "@/lib/viewModels/channelMessage";
import { getChannelState } from "@/lib/wasm-core";
import { readCurrentOrg } from "@/stores/auth";
import { create as protoCreate, toBinary } from "@bufbuild/protobuf";
import {
  InsertChannelRequestSchema,
  ReplaceChannelPodsRequestSchema,
  ReplaceChannelMembersRequestSchema,
  RemoveChannelMemberRequestSchema,
} from "@proto/channel_state/v1/mutations_pb";
import {
  channelDataToProtoChannel,
  channelPodSummaryToProtoPod,
  channelMemberDataToProto,
} from "@/lib/api/channelProtoMap";
import {
  updateChannel as updateChannelConnect,
  archiveChannel as archiveChannelConnect,
  unarchiveChannel as unarchiveChannelConnect,
  searchChannelMessages as searchChannelMessagesConnect,
  listChannelPods as listChannelPodsConnect,
  listChannelPodsRaw,
  joinChannelPod as joinChannelPodConnect,
  leaveChannelPod as leaveChannelPodConnect,
  inviteChannelMembers as inviteChannelMembersConnect,
  removeChannelMember as removeChannelMemberConnect,
  listChannelMembers as listChannelMembersConnect,
  listChannelMembersRaw,
} from "./channelConnect";

export type { MessageContent, MessageMentions } from "@/lib/viewModels/channelMessage";
export type { InlineElement, Block } from "@/lib/viewModels/channelMessage";

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
  agent_count: number;
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

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

// channelApi: thin wrappers around channelConnect that also patch Rust SSOT so
// hooks reading `channel_*_json` selectors observe fresh data. list/get/create
// + message fetch live directly in stores (channelStore / channelMessageStore).
export const channelApi = {
  update: async (id: number, data: { name?: string; description?: string; document?: string }) => {
    const channel = await updateChannelConnect(orgSlug(), id, data);
    const req = protoCreate(InsertChannelRequestSchema, {
      channel: channelDataToProtoChannel(channel),
    });
    getChannelState().insert_channel(toBinary(InsertChannelRequestSchema, req));
    return { channel };
  },

  archive: async (id: number) => {
    await archiveChannelConnect(orgSlug(), id);
    return { message: "ok" };
  },

  unarchive: async (id: number) => {
    await unarchiveChannelConnect(orgSlug(), id);
    return { message: "ok" };
  },

  searchMessages: async (id: number, q: string, limit = 20) => {
    const messages = await searchChannelMessagesConnect(orgSlug(), id, q, limit);
    return { messages };
  },

  getPods: async (id: number) => {
    const respBytes = await listChannelPodsRaw(orgSlug(), id);
    // wire bytes → Rust apply_fetched_pods; useChannelPods reads channel_pods_json.
    getChannelState().apply_fetched_pods(BigInt(id), respBytes);
    const pods = JSON.parse(getChannelState().channel_pods_json(BigInt(id))) as ChannelPodSummary[];
    return { pods, total: pods.length };
  },

  joinPod: async (id: number, podKey: string) => {
    await joinChannelPodConnect(orgSlug(), id, podKey);
    return { message: "ok" };
  },

  leavePod: async (id: number, podKey: string) => {
    await leaveChannelPodConnect(orgSlug(), id, podKey);
    return { message: "ok" };
  },

  inviteMembers: async (id: number, userIds: number[]) => {
    await inviteChannelMembersConnect(orgSlug(), id, userIds);
    return { message: "ok" };
  },

  removeMember: async (id: number, userId: number) => {
    await removeChannelMemberConnect(orgSlug(), id, userId);
    // Mirror Rust wasm path: remove from cached members via proto-bytes
    // mutator so selector re-reads observe the absence immediately.
    const req = protoCreate(RemoveChannelMemberRequestSchema, {
      channelId: BigInt(id),
      userId: BigInt(userId),
    });
    getChannelState().remove_channel_member(toBinary(RemoveChannelMemberRequestSchema, req));
    return { message: "ok" };
  },

  listMembers: async (id: number) => {
    const respBytes = await listChannelMembersRaw(orgSlug(), id);
    getChannelState().apply_fetched_members(BigInt(id), respBytes);
    const members = JSON.parse(getChannelState().channel_members_json(BigInt(id))) as ChannelMemberData[];
    return { members, total: members.length };
  },
};
