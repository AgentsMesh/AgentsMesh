import { useMemo } from "react";
import { fromBinary } from "@bufbuild/protobuf";
import { ReplaceChannelUnreadCountsRequestSchema } from "@proto/channel_state/v1/mutations_pb";
import { getChannelState } from "@/lib/wasm-core";
import { useChannelMessageStore, readMessages } from "./channelMessageStore";
import type { ChannelMessage } from "@/lib/api/facade/channel";

const svc = () => getChannelState();

// ── Selectors: Rust is SSOT. These hooks subscribe to tick counters so
// components re-render when Rust state mutates — no parallel JS copy.

export interface ChannelMessagesView {
  messages: ChannelMessage[];
  hasMore: boolean;
}

const EMPTY_VIEW: ChannelMessagesView = { messages: [], hasMore: false };

export function useChannelMessages(channelId: number | null | undefined): ChannelMessagesView {
  const tick = useChannelMessageStore((s) => s._messagesTick);
  return useMemo(() => {
    if (channelId == null) return EMPTY_VIEW;
    return readMessages(channelId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, channelId]);
}

export function useUnreadCounts(): Record<number, number> {
  const tick = useChannelMessageStore((s) => s._unreadTick);
  return useMemo(() => {
    // Read side (B, zero-JSON): decode the state proto unread map.
    const req = fromBinary(ReplaceChannelUnreadCountsRequestSchema, svc().unread_counts_bytes());
    const out: Record<number, number> = {};
    for (const [k, v] of Object.entries(req.counts)) out[Number(k)] = v;
    return out;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick]);
}

export function useUnreadCount(channelId: number | null | undefined): number {
  const tick = useChannelMessageStore((s) => s._unreadTick);
  return useMemo(() => {
    if (channelId == null) return 0;
    return svc().get_unread_count(BigInt(channelId));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tick, channelId]);
}

export function useTotalUnreadCount(): number {
  const tick = useChannelMessageStore((s) => s._unreadTick);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => svc().total_unread_count(), [tick]);
}
