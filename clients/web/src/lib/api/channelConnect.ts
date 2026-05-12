// Connect-RPC adapter for proto.channel.v1.ChannelService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out — conventions §2.5), and
// decodes responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing web ChannelData / ChannelMessage shapes (snake_case +
// number) so call sites don't have to change. The proto types are camelCase
// + BigInt; the adapter does the mapping.
//
// content / mentions ride as opaque JSON strings on the proto wire
// (`content_json` / `mentions_json`). The adapter parses them back into the
// rich AST so call-site signatures match the legacy REST DTO.

import {
  ChannelSchema,
  ChannelMessageSchema,
  ListChannelsRequestSchema,
  ListChannelsResponseSchema,
  GetChannelRequestSchema,
  CreateChannelRequestSchema,
  UpdateChannelRequestSchema,
  ArchiveChannelRequestSchema,
  ArchiveChannelResponseSchema,
  UnarchiveChannelRequestSchema,
  UnarchiveChannelResponseSchema,
  GetChannelDocumentRequestSchema,
  GetChannelDocumentResponseSchema,
  UpdateChannelDocumentRequestSchema,
  UpdateChannelDocumentResponseSchema,
  ListChannelMessagesRequestSchema,
  ListChannelMessagesResponseSchema,
  SearchChannelMessagesRequestSchema,
  SearchChannelMessagesResponseSchema,
  SendChannelMessageRequestSchema,
  EditChannelMessageRequestSchema,
  DeleteChannelMessageRequestSchema,
  DeleteChannelMessageResponseSchema,
  MarkChannelReadRequestSchema,
  MarkChannelReadResponseSchema,
  GetChannelUnreadCountsRequestSchema,
  GetChannelUnreadCountsResponseSchema,
  MuteChannelRequestSchema,
  MuteChannelResponseSchema,
  ListChannelMembersRequestSchema,
  ListChannelMembersResponseSchema,
  JoinChannelRequestSchema,
  JoinChannelResponseSchema,
  LeaveChannelRequestSchema,
  LeaveChannelResponseSchema,
  InviteChannelMembersRequestSchema,
  InviteChannelMembersResponseSchema,
  RemoveChannelMemberRequestSchema,
  RemoveChannelMemberResponseSchema,
  ListChannelPodsRequestSchema,
  ListChannelPodsResponseSchema,
  JoinChannelPodRequestSchema,
  JoinChannelPodResponseSchema,
  LeaveChannelPodRequestSchema,
  LeaveChannelPodResponseSchema,
  type Channel as ProtoChannel,
  type ChannelMessage as ProtoChannelMessage,
  type ChannelMember as ProtoChannelMember,
  type ChannelPod as ProtoChannelPod,
} from "@proto/channel/v1/channel_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getChannelService } from "@/lib/wasm-core";
import type { ChannelData, ChannelMessage } from "./channel";
import type { MessageContent, MessageMentions } from "./channel-message-types";

export type {
  ChannelData,
  ChannelMessage,
} from "./channel";

// ---------------- Field mappers ----------------

export function channelFromProto(c: ProtoChannel): ChannelData {
  return {
    id: Number(c.id),
    organization_id: Number(c.organizationId),
    name: c.name,
    description: c.description,
    document: c.document,
    repository_id: c.repositoryId !== undefined ? Number(c.repositoryId) : undefined,
    ticket_id: c.ticketId !== undefined ? Number(c.ticketId) : undefined,
    ticket_slug: c.ticketSlug,
    created_by_pod: c.createdByPod,
    created_by_user_id: c.createdByUserId !== undefined ? Number(c.createdByUserId) : undefined,
    visibility: (c.visibility === "private" ? "private" : "public"),
    is_archived: c.isArchived,
    is_member: c.isMember,
    member_count: Number(c.memberCount),
    created_at: c.createdAt,
    updated_at: c.updatedAt,
  };
}

export function messageFromProto(m: ProtoChannelMessage): ChannelMessage {
  let content: MessageContent | undefined;
  let mentions: MessageMentions | undefined;
  if (m.contentJson) {
    try { content = JSON.parse(m.contentJson) as MessageContent; } catch { /* ignore */ }
  }
  if (m.mentionsJson) {
    try { mentions = JSON.parse(m.mentionsJson) as MessageMentions; } catch { /* ignore */ }
  }
  return {
    id: Number(m.id),
    channel_id: Number(m.channelId),
    sender_pod: m.senderPod,
    sender_user_id: m.senderUserId !== undefined ? Number(m.senderUserId) : undefined,
    message_type: m.messageType,
    body: m.body,
    content,
    mentions,
    reply_to: m.replyTo !== undefined ? Number(m.replyTo) : undefined,
    edited_at: m.editedAt,
    is_deleted: m.isDeleted,
    created_at: m.createdAt,
    sender_user: m.senderUser
      ? {
          id: Number(m.senderUser.id),
          username: m.senderUser.username,
          name: m.senderUser.name,
          avatar_url: m.senderUser.avatarUrl,
        }
      : undefined,
    sender_pod_info: m.senderPodInfo
      ? {
          pod_key: m.senderPodInfo.podKey,
          alias: m.senderPodInfo.alias,
          agent: m.senderPodInfo.agent ? { name: m.senderPodInfo.agent.name } : undefined,
        }
      : undefined,
  };
}

export interface ChannelMemberData {
  channel_id: number;
  user_id: number;
  role: string;
  is_muted: boolean;
  joined_at: string;
}

function memberFromProto(m: ProtoChannelMember): ChannelMemberData {
  return {
    channel_id: Number(m.channelId),
    user_id: Number(m.userId),
    role: m.role,
    is_muted: m.isMuted,
    joined_at: m.joinedAt,
  };
}

export interface ChannelPodSummary {
  id: number;
  pod_key: string;
  alias?: string;
  status: string;
  agent_status: string;
}

function podFromProto(p: ProtoChannelPod): ChannelPodSummary {
  return {
    id: Number(p.id),
    pod_key: p.podKey,
    alias: p.alias,
    status: p.status,
    agent_status: p.agentStatus,
  };
}

// ---------------- Channels ----------------

export async function listChannels(
  orgSlug: string,
  opts: { includeArchived?: boolean; repositoryId?: number; ticketSlug?: string; limit?: number; offset?: number } = {},
): Promise<{ items: ChannelData[]; total: number; limit: number; offset: number }> {
  const req = create(ListChannelsRequestSchema, {
    orgSlug,
    includeArchived: opts.includeArchived,
    repositoryId: opts.repositoryId !== undefined ? BigInt(opts.repositoryId) : undefined,
    ticketSlug: opts.ticketSlug,
    limit: opts.limit,
    offset: opts.offset,
  });
  const bytes = toBinary(ListChannelsRequestSchema, req);
  const respBytes = await getChannelService().listChannelsConnect(bytes);
  const resp = fromBinary(ListChannelsResponseSchema, new Uint8Array(respBytes));
  return {
    items: resp.items.map(channelFromProto),
    total: Number(resp.total),
    limit: resp.limit,
    offset: resp.offset,
  };
}

export async function getChannel(orgSlug: string, id: number): Promise<ChannelData> {
  const req = create(GetChannelRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(GetChannelRequestSchema, req);
  const respBytes = await getChannelService().getChannelConnect(bytes);
  return channelFromProto(fromBinary(ChannelSchema, new Uint8Array(respBytes)));
}

export async function createChannel(
  orgSlug: string,
  data: {
    name: string;
    description?: string;
    document?: string;
    repository_id?: number;
    ticket_slug?: string;
    visibility?: string;
    member_ids?: number[];
  },
): Promise<ChannelData> {
  const req = create(CreateChannelRequestSchema, {
    orgSlug,
    name: data.name,
    description: data.description,
    document: data.document,
    repositoryId: data.repository_id !== undefined ? BigInt(data.repository_id) : undefined,
    ticketSlug: data.ticket_slug,
    visibility: data.visibility,
    memberIds: (data.member_ids ?? []).map((n) => BigInt(n)),
  });
  const bytes = toBinary(CreateChannelRequestSchema, req);
  const respBytes = await getChannelService().createChannelConnect(bytes);
  return channelFromProto(fromBinary(ChannelSchema, new Uint8Array(respBytes)));
}

export async function updateChannel(
  orgSlug: string,
  id: number,
  data: { name?: string; description?: string; document?: string },
): Promise<ChannelData> {
  const req = create(UpdateChannelRequestSchema, {
    orgSlug, id: BigInt(id),
    name: data.name, description: data.description, document: data.document,
  });
  const bytes = toBinary(UpdateChannelRequestSchema, req);
  const respBytes = await getChannelService().updateChannelConnect(bytes);
  return channelFromProto(fromBinary(ChannelSchema, new Uint8Array(respBytes)));
}

export async function archiveChannel(orgSlug: string, id: number): Promise<string> {
  const req = create(ArchiveChannelRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(ArchiveChannelRequestSchema, req);
  const respBytes = await getChannelService().archiveChannelConnect(bytes);
  return fromBinary(ArchiveChannelResponseSchema, new Uint8Array(respBytes)).message;
}

export async function unarchiveChannel(orgSlug: string, id: number): Promise<string> {
  const req = create(UnarchiveChannelRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(UnarchiveChannelRequestSchema, req);
  const respBytes = await getChannelService().unarchiveChannelConnect(bytes);
  return fromBinary(UnarchiveChannelResponseSchema, new Uint8Array(respBytes)).message;
}

// ---------------- Document ----------------

export async function getChannelDocument(orgSlug: string, id: number): Promise<string> {
  const req = create(GetChannelDocumentRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(GetChannelDocumentRequestSchema, req);
  const respBytes = await getChannelService().getChannelDocumentConnect(bytes);
  return fromBinary(GetChannelDocumentResponseSchema, new Uint8Array(respBytes)).document;
}

export async function updateChannelDocument(orgSlug: string, id: number, document: string): Promise<string> {
  const req = create(UpdateChannelDocumentRequestSchema, { orgSlug, id: BigInt(id), document });
  const bytes = toBinary(UpdateChannelDocumentRequestSchema, req);
  const respBytes = await getChannelService().updateChannelDocumentConnect(bytes);
  return fromBinary(UpdateChannelDocumentResponseSchema, new Uint8Array(respBytes)).document;
}

// ---------------- Messages ----------------

export async function listChannelMessages(
  orgSlug: string, channelId: number,
  opts: { beforeId?: number; limit?: number } = {},
): Promise<{ items: ChannelMessage[]; has_more: boolean }> {
  const req = create(ListChannelMessagesRequestSchema, {
    orgSlug, channelId: BigInt(channelId),
    beforeId: opts.beforeId !== undefined ? BigInt(opts.beforeId) : undefined,
    limit: opts.limit,
  });
  const bytes = toBinary(ListChannelMessagesRequestSchema, req);
  const respBytes = await getChannelService().listChannelMessagesConnect(bytes);
  const resp = fromBinary(ListChannelMessagesResponseSchema, new Uint8Array(respBytes));
  return { items: resp.items.map(messageFromProto), has_more: resp.hasMore };
}

export async function searchChannelMessages(
  orgSlug: string, channelId: number, query: string, limit?: number,
): Promise<ChannelMessage[]> {
  const req = create(SearchChannelMessagesRequestSchema, {
    orgSlug, channelId: BigInt(channelId), query, limit,
  });
  const bytes = toBinary(SearchChannelMessagesRequestSchema, req);
  const respBytes = await getChannelService().searchChannelMessagesConnect(bytes);
  const resp = fromBinary(SearchChannelMessagesResponseSchema, new Uint8Array(respBytes));
  return resp.items.map(messageFromProto);
}

export interface SendChannelMessagePayload {
  source?: string;
  mentions?: Record<string, { entity_type: string; entity_key: string }>;
  content?: MessageContent;
  attachment_key?: string;
  pod_key?: string;
  reply_to?: number;
}

export async function sendChannelMessage(
  orgSlug: string, channelId: number, payload: SendChannelMessagePayload,
): Promise<ChannelMessage> {
  const protoMentions: Record<string, { entityType: string; entityKey: string }> = {};
  for (const [k, v] of Object.entries(payload.mentions ?? {})) {
    protoMentions[k] = { entityType: v.entity_type, entityKey: v.entity_key };
  }
  const req = create(SendChannelMessageRequestSchema, {
    orgSlug, channelId: BigInt(channelId),
    source: payload.source,
    mentions: protoMentions,
    contentJson: payload.content ? JSON.stringify(payload.content) : undefined,
    attachmentKey: payload.attachment_key,
    podKey: payload.pod_key,
    replyTo: payload.reply_to !== undefined ? BigInt(payload.reply_to) : undefined,
  });
  const bytes = toBinary(SendChannelMessageRequestSchema, req);
  const respBytes = await getChannelService().sendChannelMessageConnect(bytes);
  return messageFromProto(fromBinary(ChannelMessageSchema, new Uint8Array(respBytes)));
}

export async function editChannelMessage(
  orgSlug: string, channelId: number, messageId: number,
  payload: Omit<SendChannelMessagePayload, "pod_key" | "reply_to">,
): Promise<ChannelMessage> {
  const protoMentions: Record<string, { entityType: string; entityKey: string }> = {};
  for (const [k, v] of Object.entries(payload.mentions ?? {})) {
    protoMentions[k] = { entityType: v.entity_type, entityKey: v.entity_key };
  }
  const req = create(EditChannelMessageRequestSchema, {
    orgSlug, channelId: BigInt(channelId), messageId: BigInt(messageId),
    source: payload.source,
    mentions: protoMentions,
    contentJson: payload.content ? JSON.stringify(payload.content) : undefined,
    attachmentKey: payload.attachment_key,
  });
  const bytes = toBinary(EditChannelMessageRequestSchema, req);
  const respBytes = await getChannelService().editChannelMessageConnect(bytes);
  return messageFromProto(fromBinary(ChannelMessageSchema, new Uint8Array(respBytes)));
}

export async function deleteChannelMessage(
  orgSlug: string, channelId: number, messageId: number,
): Promise<void> {
  const req = create(DeleteChannelMessageRequestSchema, {
    orgSlug, channelId: BigInt(channelId), messageId: BigInt(messageId),
  });
  const bytes = toBinary(DeleteChannelMessageRequestSchema, req);
  const respBytes = await getChannelService().deleteChannelMessageConnect(bytes);
  fromBinary(DeleteChannelMessageResponseSchema, new Uint8Array(respBytes));
}

// ---------------- Read state ----------------

export async function markChannelRead(
  orgSlug: string, channelId: number, messageId: number,
): Promise<void> {
  const req = create(MarkChannelReadRequestSchema, {
    orgSlug, channelId: BigInt(channelId), messageId: BigInt(messageId),
  });
  const bytes = toBinary(MarkChannelReadRequestSchema, req);
  const respBytes = await getChannelService().markChannelReadConnect(bytes);
  fromBinary(MarkChannelReadResponseSchema, new Uint8Array(respBytes));
}

export async function getChannelUnreadCounts(orgSlug: string): Promise<Record<string, number>> {
  const req = create(GetChannelUnreadCountsRequestSchema, { orgSlug });
  const bytes = toBinary(GetChannelUnreadCountsRequestSchema, req);
  const respBytes = await getChannelService().getChannelUnreadCountsConnect(bytes);
  const resp = fromBinary(GetChannelUnreadCountsResponseSchema, new Uint8Array(respBytes));
  const out: Record<string, number> = {};
  for (const [k, v] of Object.entries(resp.unread)) {
    out[k] = Number(v);
  }
  return out;
}

export async function muteChannel(orgSlug: string, id: number, muted: boolean): Promise<void> {
  const req = create(MuteChannelRequestSchema, { orgSlug, id: BigInt(id), muted });
  const bytes = toBinary(MuteChannelRequestSchema, req);
  const respBytes = await getChannelService().muteChannelConnect(bytes);
  fromBinary(MuteChannelResponseSchema, new Uint8Array(respBytes));
}

// ---------------- Members ----------------

export async function listChannelMembers(
  orgSlug: string, id: number, opts: { limit?: number; offset?: number } = {},
): Promise<{ members: ChannelMemberData[]; total: number }> {
  const req = create(ListChannelMembersRequestSchema, {
    orgSlug, id: BigInt(id), limit: opts.limit, offset: opts.offset,
  });
  const bytes = toBinary(ListChannelMembersRequestSchema, req);
  const respBytes = await getChannelService().listChannelMembersConnect(bytes);
  const resp = fromBinary(ListChannelMembersResponseSchema, new Uint8Array(respBytes));
  return { members: resp.items.map(memberFromProto), total: Number(resp.total) };
}

export async function joinChannel(orgSlug: string, id: number): Promise<void> {
  const req = create(JoinChannelRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(JoinChannelRequestSchema, req);
  const respBytes = await getChannelService().joinChannelConnect(bytes);
  fromBinary(JoinChannelResponseSchema, new Uint8Array(respBytes));
}

export async function leaveChannel(orgSlug: string, id: number): Promise<void> {
  const req = create(LeaveChannelRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(LeaveChannelRequestSchema, req);
  const respBytes = await getChannelService().leaveChannelConnect(bytes);
  fromBinary(LeaveChannelResponseSchema, new Uint8Array(respBytes));
}

export async function inviteChannelMembers(
  orgSlug: string, id: number, userIds: number[],
): Promise<void> {
  const req = create(InviteChannelMembersRequestSchema, {
    orgSlug, id: BigInt(id), userIds: userIds.map((n) => BigInt(n)),
  });
  const bytes = toBinary(InviteChannelMembersRequestSchema, req);
  const respBytes = await getChannelService().inviteChannelMembersConnect(bytes);
  fromBinary(InviteChannelMembersResponseSchema, new Uint8Array(respBytes));
}

export async function removeChannelMember(
  orgSlug: string, id: number, userId: number,
): Promise<void> {
  const req = create(RemoveChannelMemberRequestSchema, {
    orgSlug, id: BigInt(id), userId: BigInt(userId),
  });
  const bytes = toBinary(RemoveChannelMemberRequestSchema, req);
  const respBytes = await getChannelService().removeChannelMemberConnect(bytes);
  fromBinary(RemoveChannelMemberResponseSchema, new Uint8Array(respBytes));
}

// ---------------- Channel pods ----------------

export async function listChannelPods(
  orgSlug: string, id: number,
): Promise<{ pods: ChannelPodSummary[]; total: number }> {
  const req = create(ListChannelPodsRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(ListChannelPodsRequestSchema, req);
  const respBytes = await getChannelService().listChannelPodsConnect(bytes);
  const resp = fromBinary(ListChannelPodsResponseSchema, new Uint8Array(respBytes));
  return { pods: resp.items.map(podFromProto), total: Number(resp.total) };
}

export async function joinChannelPod(
  orgSlug: string, id: number, podKey: string,
): Promise<void> {
  const req = create(JoinChannelPodRequestSchema, { orgSlug, id: BigInt(id), podKey });
  const bytes = toBinary(JoinChannelPodRequestSchema, req);
  const respBytes = await getChannelService().joinChannelPodConnect(bytes);
  fromBinary(JoinChannelPodResponseSchema, new Uint8Array(respBytes));
}

export async function leaveChannelPod(
  orgSlug: string, id: number, podKey: string,
): Promise<void> {
  const req = create(LeaveChannelPodRequestSchema, { orgSlug, id: BigInt(id), podKey });
  const bytes = toBinary(LeaveChannelPodRequestSchema, req);
  const respBytes = await getChannelService().leaveChannelPodConnect(bytes);
  fromBinary(LeaveChannelPodResponseSchema, new Uint8Array(respBytes));
}
