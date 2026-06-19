// renderer cache (snake_case JSON) → state proto bytes. Mirror image of the
// wasm `*_bytes()` readers so the shared web selectors decode desktop and web
// identically (fromBinary(StateSchema) → xToCache). Field names here MUST match
// what channelSelectors.channelToCache / channelMessageStore.messageToCache read.
import { create, toBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedChannelsRequestSchema,
  InsertChannelRequestSchema,
  ReplaceCachedChannelMessagesRequestSchema,
  ReplaceChannelMembersRequestSchema,
  ReplaceChannelPodsRequestSchema,
} from "@agentsmesh/proto/channel_state/v1/mutations_pb";
import { MessagePreviewSchema } from "@agentsmesh/proto/channel_state/v1/channel_state_pb";

type Obj = Record<string, unknown>;
const num = (v: unknown): number => (v as number);

function cacheToStateChannel(c: Obj) {
  return {
    id: BigInt(num(c.id)),
    organizationId: c.organization_id != null ? BigInt(num(c.organization_id)) : undefined,
    name: (c.name as string) ?? "", description: (c.description as string) ?? "",
    document: (c.document as string) ?? "",
    repositoryId: c.repository_id != null ? BigInt(num(c.repository_id)) : undefined,
    ticketId: c.ticket_id != null ? BigInt(num(c.ticket_id)) : undefined,
    ticketSlug: (c.ticket_slug as string) ?? "",
    visibility: (c.visibility as string) ?? "",
    isArchived: !!c.is_archived, isMember: !!c.is_member,
    memberCount: c.member_count != null ? BigInt(num(c.member_count)) : undefined,
    agentCount: c.agent_count != null ? BigInt(num(c.agent_count)) : undefined,
    createdByPod: (c.created_by_pod as string) ?? "",
    createdByUserId: c.created_by_user_id != null ? BigInt(num(c.created_by_user_id)) : undefined,
    createdAt: (c.created_at as string) ?? "", updatedAt: (c.updated_at as string) ?? "",
  };
}

function cacheToStateMessage(m: Obj) {
  const su = m.sender_user as Obj | undefined;
  const sp = m.sender_pod_info as Obj | undefined;
  const agent = sp?.agent as Obj | undefined;
  return {
    id: BigInt(num(m.id)), channelId: BigInt(num(m.channel_id)),
    senderPod: (m.sender_pod as string) ?? undefined,
    senderUserId: m.sender_user_id != null ? BigInt(num(m.sender_user_id)) : undefined,
    senderUser: su ? {
      id: BigInt(num(su.id)), username: (su.username as string) ?? "",
      name: (su.name as string) ?? "", avatarUrl: (su.avatar_url as string) ?? undefined,
    } : undefined,
    senderPodInfo: sp ? {
      podKey: (sp.pod_key as string) ?? "", alias: (sp.alias as string) ?? undefined,
      agent: agent ? { name: (agent.name as string) ?? "" } : undefined,
    } : undefined,
    messageType: (m.message_type as string) ?? "", body: (m.body as string) ?? "",
    contentJson: (m.content_json as string) ?? undefined,
    mentionsJson: (m.mentions_json as string) ?? undefined,
    replyTo: m.reply_to != null ? BigInt(num(m.reply_to)) : undefined,
    editedAt: (m.edited_at as string) ?? undefined,
    isDeleted: !!m.is_deleted, createdAt: (m.created_at as string) ?? "",
  };
}

function cacheToStateMember(m: Obj) {
  return {
    channelId: BigInt(num(m.channel_id)), userId: BigInt(num(m.user_id)),
    role: (m.role as string) ?? "", isMuted: !!m.is_muted, joinedAt: (m.joined_at as string) ?? "",
  };
}

function cacheToStatePod(p: Obj) {
  return {
    id: p.id != null ? BigInt(num(p.id)) : BigInt(0), podKey: (p.pod_key as string) ?? "",
    alias: (p.alias as string) ?? undefined, status: (p.status as string) ?? "",
    agentStatus: (p.agent_status as string) ?? "",
  };
}

export function channelsBytes(cacheJson: string): Uint8Array {
  const list = JSON.parse(cacheJson) as Obj[];
  return toBinary(ReplaceCachedChannelsRequestSchema,
    create(ReplaceCachedChannelsRequestSchema, { channels: list.map(cacheToStateChannel) }));
}

function findChannel(cacheJson: string, id: number): Obj | undefined {
  return (JSON.parse(cacheJson) as Obj[]).find((c) => num(c.id) === id);
}

export function channelBytes(cacheJson: string, id: number): Uint8Array {
  const ch = findChannel(cacheJson, id);
  if (!ch) return new Uint8Array();
  return toBinary(InsertChannelRequestSchema,
    create(InsertChannelRequestSchema, { channel: cacheToStateChannel(ch) }));
}

export function currentChannelBytes(cacheJson: string, currentId: number | null): Uint8Array {
  if (currentId == null) return new Uint8Array();
  return channelBytes(cacheJson, currentId);
}

export function messagesBytes(channelId: number, messages: Obj[], hasMore: boolean): Uint8Array {
  return toBinary(ReplaceCachedChannelMessagesRequestSchema,
    create(ReplaceCachedChannelMessagesRequestSchema, {
      channelId: BigInt(channelId), hasMore, messages: messages.map(cacheToStateMessage),
    }));
}

export function lastMessageBytes(messages: Obj[]): Uint8Array {
  if (messages.length === 0) return new Uint8Array();
  const m = messages[messages.length - 1];
  const su = m.sender_user as Obj | undefined;
  const sp = m.sender_pod_info as Obj | undefined;
  const body = typeof m.body === "string" ? m.body : "";
  return toBinary(MessagePreviewSchema, create(MessagePreviewSchema, {
    messageId: BigInt(num(m.id)),
    senderName: (su?.name as string) || (su?.username as string) || (sp?.alias as string) || (sp?.pod_key as string) || "",
    contentPreview: body.slice(0, 80),
    messageType: (m.message_type as string) || undefined,
    timestamp: (m.created_at as string) ?? "",
  }));
}

export function membersBytes(channelId: number, json: string): Uint8Array {
  const members = JSON.parse(json) as Obj[];
  return toBinary(ReplaceChannelMembersRequestSchema,
    create(ReplaceChannelMembersRequestSchema, { channelId: BigInt(channelId), members: members.map(cacheToStateMember) }));
}

export function podsBytes(channelId: number, json: string): Uint8Array {
  const pods = JSON.parse(json) as Obj[];
  return toBinary(ReplaceChannelPodsRequestSchema,
    create(ReplaceChannelPodsRequestSchema, { channelId: BigInt(channelId), pods: pods.map(cacheToStatePod) }));
}
