// Channel read-state + member-flag RPC adapters: mark read/unread, unread
// counts, read receipts, mute, pin. Split from channelMessageConnect.ts to keep
// each file under the size limit; identical binary-in / binary-out transport.

import {
  MarkChannelReadRequestSchema,
  MarkChannelReadResponseSchema,
  MarkChannelUnreadRequestSchema,
  MarkChannelUnreadResponseSchema,
  GetChannelUnreadCountsRequestSchema,
  GetChannelUnreadCountsResponseSchema,
  GetMessageReadByRequestSchema,
  GetMessageReadByResponseSchema,
  MuteChannelRequestSchema,
  MuteChannelResponseSchema,
  PinChannelRequestSchema,
  PinChannelResponseSchema,
} from "@proto/channel/v1/channel_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getChannelService } from "@/lib/wasm-core";

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

export interface ChannelUnreadSummary {
  unread: Record<string, number>;
  mentions: Record<string, number>;
  lastRead: Record<string, number>;
  manuallyUnread: Record<string, boolean>;
}

export async function getChannelUnreadCounts(orgSlug: string): Promise<ChannelUnreadSummary> {
  const req = create(GetChannelUnreadCountsRequestSchema, { orgSlug });
  const bytes = toBinary(GetChannelUnreadCountsRequestSchema, req);
  const respBytes = await getChannelService().getChannelUnreadCountsConnect(bytes);
  const resp = fromBinary(GetChannelUnreadCountsResponseSchema, new Uint8Array(respBytes));
  const toNumberMap = (m: Record<string, bigint>): Record<string, number> => {
    const out: Record<string, number> = {};
    for (const [k, v] of Object.entries(m)) out[k] = Number(v);
    return out;
  };
  // lastRead is a message id coerced bigint→number to match the codebase-wide
  // number typing of message ids (TransformedMessage.id, the `id > lastRead`
  // divider comparison in useChannelEntryAnchor). A lone bigint here would make
  // those mixed-type comparisons throw. Safe until ids exceed 2^53.
  return {
    unread: toNumberMap(resp.unread),
    mentions: toNumberMap(resp.mentions),
    lastRead: toNumberMap(resp.lastRead),
    manuallyUnread: { ...resp.manuallyUnread },
  };
}

export async function markChannelUnread(orgSlug: string, channelId: number): Promise<void> {
  const req = create(MarkChannelUnreadRequestSchema, { orgSlug, channelId: BigInt(channelId) });
  const bytes = toBinary(MarkChannelUnreadRequestSchema, req);
  const respBytes = await getChannelService().markChannelUnreadConnect(bytes);
  fromBinary(MarkChannelUnreadResponseSchema, new Uint8Array(respBytes));
}

export async function getMessageReadBy(
  orgSlug: string, channelId: number, messageId: number,
): Promise<number[]> {
  const req = create(GetMessageReadByRequestSchema, {
    orgSlug, channelId: BigInt(channelId), messageId: BigInt(messageId),
  });
  const bytes = toBinary(GetMessageReadByRequestSchema, req);
  const respBytes = await getChannelService().getMessageReadByConnect(bytes);
  const resp = fromBinary(GetMessageReadByResponseSchema, new Uint8Array(respBytes));
  return resp.userIds.map((id) => Number(id));
}

export async function muteChannel(orgSlug: string, id: number, muted: boolean): Promise<void> {
  const req = create(MuteChannelRequestSchema, { orgSlug, id: BigInt(id), muted });
  const bytes = toBinary(MuteChannelRequestSchema, req);
  const respBytes = await getChannelService().muteChannelConnect(bytes);
  fromBinary(MuteChannelResponseSchema, new Uint8Array(respBytes));
}

export async function pinChannel(orgSlug: string, id: number, pinned: boolean): Promise<void> {
  const req = create(PinChannelRequestSchema, { orgSlug, id: BigInt(id), pinned });
  const bytes = toBinary(PinChannelRequestSchema, req);
  const respBytes = await getChannelService().pinChannelConnect(bytes);
  fromBinary(PinChannelResponseSchema, new Uint8Array(respBytes));
}
