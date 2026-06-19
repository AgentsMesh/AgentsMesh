// Channel message ops — list / search / send / edit / delete + read state.
// Split out of channelConnect.ts for SRP (file size limit).
//
// Shares `messageFromProto` with channelConnect.ts. Wire transport is
// identical: @bufbuild/protobuf binary in / binary out through the wasm
// ChannelService bridge.

import {
  ChannelMessageSchema,
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
} from "@proto/channel/v1/channel_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getChannelService } from "@/lib/wasm-core";
import { messageFromProto } from "./channelConnect";
import type { ChannelMessage } from "../facade/channel";
import type { MessageContent } from "@/lib/viewModels/channelMessage";

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

// Raw wire bytes for the fetch→state path: ListChannelMessagesResponse goes
// straight to Rust apply_fetched_messages (no TS messageFromProto/xToProto).
export async function listChannelMessagesRaw(
  orgSlug: string, channelId: number,
  opts: { beforeId?: number; limit?: number } = {},
): Promise<Uint8Array> {
  const req = create(ListChannelMessagesRequestSchema, {
    orgSlug, channelId: BigInt(channelId),
    beforeId: opts.beforeId !== undefined ? BigInt(opts.beforeId) : undefined,
    limit: opts.limit,
  });
  const bytes = toBinary(ListChannelMessagesRequestSchema, req);
  return new Uint8Array(await getChannelService().listChannelMessagesConnect(bytes));
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
