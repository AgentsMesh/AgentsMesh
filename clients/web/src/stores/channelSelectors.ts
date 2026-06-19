import { useMemo } from "react";
import { fromBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedChannelsRequestSchema,
  ReplaceChannelMembersRequestSchema,
  InsertChannelRequestSchema,
} from "@proto/channel_state/v1/mutations_pb";
import {
  MessagePreviewSchema,
  type Channel as ProtoStateChannel,
  type ChannelMember as ProtoStateMember,
  type MessagePreview as ProtoStateMessagePreview,
} from "@proto/channel_state/v1/channel_state_pb";
import { getChannelState } from "@/lib/wasm-core";
import { useChannelStore } from "./channelStore";
import type { Channel, ChannelLastMessage, ChannelMember } from "./channelTypes";

const svc = () => getChannelState();

// state proto Channel → store Channel (snake_case + client-derived unread/
// mention/last_message). Replaces the serde-JSON channels_json read path —
// reading is now proto-bytes + this projection, zero JSON.
export function channelToCache(c: ProtoStateChannel): Channel {
  return {
    id: Number(c.id),
    organization_id: c.organizationId !== undefined ? Number(c.organizationId) : undefined,
    name: c.name,
    description: c.description,
    document: c.document,
    repository_id: c.repositoryId !== undefined ? Number(c.repositoryId) : undefined,
    ticket_id: c.ticketId !== undefined ? Number(c.ticketId) : undefined,
    ticket_slug: c.ticketSlug,
    created_by_pod: c.createdByPod,
    created_by_user_id: c.createdByUserId !== undefined ? Number(c.createdByUserId) : undefined,
    visibility: c.visibility as Channel["visibility"],
    is_archived: c.isArchived,
    is_member: c.isMember,
    member_count: c.memberCount !== undefined ? Number(c.memberCount) : 0,
    agent_count: c.agentCount !== undefined ? Number(c.agentCount) : undefined,
    created_at: c.createdAt,
    updated_at: c.updatedAt,
    unread_count: c.unreadCount,
    mention_count: c.mentionCount,
    last_message: c.lastMessage
      ? {
          sender_name: c.lastMessage.senderName,
          content_preview: c.lastMessage.contentPreview,
          message_type: c.lastMessage.messageType,
          timestamp: c.lastMessage.timestamp,
        }
      : undefined,
    last_activity_at: c.lastActivityAt,
  } as unknown as Channel;
}

export function useChannels(): Channel[] {
  const tick = useChannelStore((s) => s._tick);
  return useMemo(
    () => fromBinary(ReplaceCachedChannelsRequestSchema, svc().channels_bytes()).channels.map(channelToCache),
    [tick],
  );
}

function previewToCache(p: ProtoStateMessagePreview): ChannelLastMessage {
  return {
    sender_name: p.senderName,
    content_preview: p.contentPreview,
    message_type: p.messageType,
    timestamp: p.timestamp,
  } as ChannelLastMessage;
}

/** Read the cached last-message preview for a channel (from WASM `last_messages` map). */
export function getLastMessage(channelId: number): ChannelLastMessage | null {
  const bytes = svc().get_last_message_bytes(BigInt(channelId));
  if (bytes.length === 0) return null;
  return previewToCache(fromBinary(MessagePreviewSchema, bytes));
}

export function useCurrentChannel(): Channel | null {
  const tick = useChannelStore((s) => s._tick);
  return useMemo(() => {
    const bytes = svc().current_channel_bytes();
    if (bytes.length === 0) return null;
    const c = fromBinary(InsertChannelRequestSchema, bytes).channel;
    return c ? channelToCache(c) : null;
  }, [tick]);
}

/** Members of a given channel. Rust ChannelService caches the list per channel
 *  in state; the hook re-reads whenever `_tick` bumps (fetch / invite / remove). */
function memberToCache(m: ProtoStateMember): ChannelMember {
  return {
    channel_id: Number(m.channelId),
    user_id: Number(m.userId),
    role: m.role,
    is_muted: m.isMuted,
    joined_at: m.joinedAt,
  };
}

export function useChannelMembers(channelId: number | null | undefined): ChannelMember[] {
  const tick = useChannelStore((s) => s._tick);
  return useMemo(() => {
    if (channelId == null) return [];
    try {
      return fromBinary(ReplaceChannelMembersRequestSchema, svc().channel_members_bytes(BigInt(channelId)))
        .members.map(memberToCache);
    } catch {
      return [];
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, channelId]);
}

export function readChannel(id: number): Channel | null {
  const bytes = svc().get_channel_bytes(BigInt(id));
  if (bytes.length === 0) return null;
  const c = fromBinary(InsertChannelRequestSchema, bytes).channel;
  return c ? channelToCache(c) : null;
}
