// Proto-dispatch helpers: encode a ChannelMessage into the channel_state
// mutation envelopes and push it through the wasm ChannelState bridge. Pure
// functions over svc() — no store/tick coupling (callers bump separately).
// Split from channelMessageStore.ts for SRP (file size limit).

import { create as protoCreate, fromBinary, toBinary } from "@bufbuild/protobuf";
import { getChannelState } from "@/lib/wasm-core";
import { channelMessageToProto } from "@/lib/api/channelProtoMap";
import {
  ReplaceCachedChannelMessagesRequestSchema,
  InsertChannelMessageRequestSchema,
  ApplyIncomingChannelMessageRequestSchema,
  ApplyChannelMessageEditedEventRequestSchema,
} from "@proto/channel_state/v1/mutations_pb";
import type { ChannelMessage as ProtoStateMessage } from "@proto/channel_state/v1/channel_state_pb";
import { fromWasmProjection, type WasmChannelMessage } from "./channelMessageWasmProjection";
import type { ChannelMessage } from "@/lib/api/facade/channel";

const svc = () => getChannelState();

// state proto ChannelMessage → WasmChannelMessage (snake_case + sender nested +
// content_json/mentions_json string). Read side, zero-JSON; fromWasmProjection
// then parses the rich AST. Replaces the get_messages_json serde read.
function messageToCache(m: ProtoStateMessage): WasmChannelMessage {
  return {
    id: Number(m.id),
    channel_id: Number(m.channelId),
    sender_pod: m.senderPod,
    sender_user_id: m.senderUserId !== undefined && m.senderUserId !== BigInt(0) ? Number(m.senderUserId) : undefined,
    sender_user: m.senderUser ? {
      id: Number(m.senderUser.id), username: m.senderUser.username,
      name: m.senderUser.name, avatar_url: m.senderUser.avatarUrl,
    } : undefined,
    sender_pod_info: m.senderPodInfo ? {
      pod_key: m.senderPodInfo.podKey, alias: m.senderPodInfo.alias,
      agent: m.senderPodInfo.agent ? { name: m.senderPodInfo.agent.name } : undefined,
    } : undefined,
    message_type: m.messageType,
    body: m.body,
    content_json: m.contentJson || undefined,
    mentions_json: m.mentionsJson || undefined,
    reply_to: m.replyTo !== undefined && m.replyTo !== BigInt(0) ? Number(m.replyTo) : undefined,
    edited_at: m.editedAt || undefined,
    is_deleted: m.isDeleted,
    created_at: m.createdAt,
  } as WasmChannelMessage;
}

export function readMessages(channelId: number): { messages: ChannelMessage[]; hasMore: boolean } {
  const req = fromBinary(ReplaceCachedChannelMessagesRequestSchema, svc().get_messages_bytes(BigInt(channelId)));
  return { messages: req.messages.map(messageToCache).map(fromWasmProjection), hasMore: req.hasMore };
}

export function dispatchInsertMessage(channelId: number, message: ChannelMessage) {
  const req = protoCreate(InsertChannelMessageRequestSchema, {
    channelId: BigInt(channelId),
    message: channelMessageToProto(message),
  });
  svc().insert_channel_message(toBinary(InsertChannelMessageRequestSchema, req));
}

export function dispatchIncomingMessage(message: ChannelMessage): boolean {
  const req = protoCreate(ApplyIncomingChannelMessageRequestSchema, {
    channelId: BigInt(message.channel_id),
    message: channelMessageToProto(message),
  });
  return svc().apply_incoming_channel_message(
    toBinary(ApplyIncomingChannelMessageRequestSchema, req),
  );
}

export function dispatchMessageEdited(channelId: number, edit: {
  id: number; body: string; content?: ChannelMessage["content"]; mentions?: ChannelMessage["mentions"]; edited_at?: string;
}) {
  const req = protoCreate(ApplyChannelMessageEditedEventRequestSchema, {
    channelId: BigInt(channelId),
    messageId: BigInt(edit.id),
    body: edit.body,
    content: edit.content ? JSON.stringify(edit.content) : undefined,
    mentions: edit.mentions
      ? Object.fromEntries(
          Object.entries(edit.mentions).map(([k, v]) => [k, typeof v === "string" ? v : JSON.stringify(v)]),
        )
      : {},
    editedAt: edit.edited_at ?? "",
  });
  svc().apply_channel_message_edited_event(
    toBinary(ApplyChannelMessageEditedEventRequestSchema, req),
  );
}
