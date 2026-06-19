// proto → renderer cache shape. Accepts BOTH the wire proto (channel/v1, from
// fetch responses) and the state proto (channel_state/v1, from mutators) — the
// two share field names and differ only in scalar optionality, so one set of
// converters serves both. The renderer's synchronous readers parse the cache
// JSON expecting snake_case fields with content/mentions as opaque *_json
// strings (matches WasmChannelMessage).
import type {
  Channel as WireChannel,
  ChannelMessage as WireMessage,
  ChannelMember as WireMember,
  ChannelPod as WireChannelPod,
} from "@agentsmesh/proto/channel/v1/channel_pb";
import type {
  Channel as StateChannel,
  ChannelMessage as StateMessage,
  ChannelMember as StateMember,
} from "@agentsmesh/proto/channel_state/v1/channel_state_pb";
import type { Pod as StatePod } from "@agentsmesh/proto/pod/v1/pod_pb";

export function channelToCache(c: WireChannel | StateChannel): Record<string, unknown> {
  return {
    id: Number(c.id),
    organization_id: Number(c.organizationId),
    name: c.name,
    description: c.description,
    document: c.document,
    repository_id: c.repositoryId !== undefined ? Number(c.repositoryId) : undefined,
    ticket_id: c.ticketId !== undefined ? Number(c.ticketId) : undefined,
    ticket_slug: c.ticketSlug || undefined,
    visibility: c.visibility,
    is_archived: c.isArchived,
    is_member: c.isMember,
    member_count: Number(c.memberCount),
    agent_count: Number(c.agentCount),
    created_by_pod: c.createdByPod || undefined,
    created_by_user_id: c.createdByUserId !== undefined ? Number(c.createdByUserId) : undefined,
    created_at: c.createdAt,
    updated_at: c.updatedAt,
  };
}

export function messageToCache(m: WireMessage | StateMessage): Record<string, unknown> {
  return {
    id: Number(m.id),
    channel_id: Number(m.channelId),
    sender_pod: m.senderPod,
    sender_user_id: m.senderUserId !== undefined && m.senderUserId !== BigInt(0)
      ? Number(m.senderUserId) : undefined,
    sender_user: m.senderUser ? {
      id: Number(m.senderUser.id),
      username: m.senderUser.username,
      name: m.senderUser.name,
      avatar_url: m.senderUser.avatarUrl,
    } : undefined,
    sender_pod_info: m.senderPodInfo ? {
      pod_key: m.senderPodInfo.podKey,
      alias: m.senderPodInfo.alias,
      agent: m.senderPodInfo.agent ? { name: m.senderPodInfo.agent.name } : undefined,
    } : undefined,
    message_type: m.messageType,
    body: m.body,
    content_json: m.contentJson || undefined,
    mentions_json: m.mentionsJson || undefined,
    reply_to: m.replyTo !== undefined && m.replyTo !== BigInt(0)
      ? Number(m.replyTo) : undefined,
    edited_at: m.editedAt || undefined,
    is_deleted: m.isDeleted,
    created_at: m.createdAt,
  };
}

export function podToCache(p: WireChannelPod | StatePod): Record<string, unknown> {
  return {
    id: Number(p.id),
    pod_key: p.podKey,
    alias: p.alias,
    status: p.status,
    agent_status: p.agentStatus,
  };
}

export function memberToCache(m: WireMember | StateMember): Record<string, unknown> {
  return {
    channel_id: Number(m.channelId),
    user_id: Number(m.userId),
    role: m.role,
    is_muted: m.isMuted,
    joined_at: m.joinedAt,
  };
}
