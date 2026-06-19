// Connect-RPC adapter for proto.channel.v1.ChannelService — channel CRUD +
// document state + shared field mappers.
//
// Message ops live in `channelMessageConnect.ts`; member + pod ops in
// `channelMembersConnect.ts`. The proto wire layer is hidden behind the
// `facade/channelConnect.ts` re-export; business callers MUST NOT import
// from this file directly (enforced by no-restricted-imports).
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out — conventions §2.5), and
// decodes responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing web ChannelData / ChannelMessage shapes (snake_case +
// number) so call sites don't have to change. The proto types are camelCase
// + BigInt; the adapter does the mapping.

import {
  ChannelSchema,
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
  type Channel as ProtoChannel,
  type ChannelMessage as ProtoChannelMessage,
} from "@proto/channel/v1/channel_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getChannelService } from "@/lib/wasm-core";
import type { ChannelData, ChannelMessage } from "../facade/channel";
import type { MessageContent, MessageMentions } from "@/lib/viewModels/channelMessage";

export type { ChannelData, ChannelMessage } from "../facade/channel";

// ---------------- Field mappers (shared across split files) ----------------

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
    agent_count: Number(c.agentCount),
    created_at: c.createdAt,
    updated_at: c.updatedAt,
  };
}

// Exported for use by channelMessageConnect.ts.
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

// Raw wire bytes for the fetch→state path: the ListChannelsResponse goes
// straight to Rust apply_fetched_channels (no TS channelFromProto/xToProto).
export async function listChannelsRaw(
  orgSlug: string,
  opts: { includeArchived?: boolean; repositoryId?: number; ticketSlug?: string } = {},
): Promise<Uint8Array> {
  const req = create(ListChannelsRequestSchema, {
    orgSlug,
    includeArchived: opts.includeArchived,
    repositoryId: opts.repositoryId !== undefined ? BigInt(opts.repositoryId) : undefined,
    ticketSlug: opts.ticketSlug,
  });
  const bytes = toBinary(ListChannelsRequestSchema, req);
  return new Uint8Array(await getChannelService().listChannelsConnect(bytes));
}

export async function getChannel(orgSlug: string, id: number): Promise<ChannelData> {
  const req = create(GetChannelRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(GetChannelRequestSchema, req);
  const respBytes = await getChannelService().getChannelConnect(bytes);
  return channelFromProto(fromBinary(ChannelSchema, new Uint8Array(respBytes)));
}

// Raw wire bytes for the fetch→state path: response (Channel) →
// apply_fetched_channel (Rust upsert), no TS channelFromProto +
// channelToProtoChannel round-trip.
export async function getChannelRaw(orgSlug: string, id: number): Promise<Uint8Array> {
  const req = create(GetChannelRequestSchema, { orgSlug, id: BigInt(id) });
  return new Uint8Array(
    await getChannelService().getChannelConnect(toBinary(GetChannelRequestSchema, req)),
  );
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
