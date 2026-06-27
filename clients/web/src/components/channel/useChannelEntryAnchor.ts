"use client";

import { useMemo, useRef } from "react";
import { getChannelState } from "@/lib/wasm-core";
import type { TransformedMessage } from "./types";

/**
 * Freeze the last-read cursor the first time a channel is opened — before the
 * 300ms markRead advances it — so the "new messages" boundary stays put while
 * the user reads. Returns the id of the first message after that cursor (the
 * divider + scroll anchor), or null when the channel is fully read.
 */
export function useChannelEntryAnchor(
  channelId: number,
  messages: TransformedMessage[],
): number | null {
  const snapRef = useRef<Map<number, number>>(new Map());
  // Re-snapshot on each fresh (re-)entry — drop a stale snapshot for this
  // channel so the divider reflects the CURRENT last-read cursor, not the one
  // frozen on a prior visit. Stays frozen while we remain in the channel.
  const enteredRef = useRef<number | null>(null);
  if (channelId !== enteredRef.current) {
    snapRef.current.delete(channelId);
    enteredRef.current = channelId;
  }
  if (channelId && !snapRef.current.has(channelId)) {
    snapRef.current.set(channelId, getChannelState().get_last_read_id(BigInt(channelId)));
  }
  const lastReadId = channelId ? snapRef.current.get(channelId) ?? -1 : -1;

  return useMemo(() => {
    // -1 = no known cursor (channel never reported by a summary fetch) → no
    // divider. A genuine 0 cursor ("read nothing yet") is kept: every message id
    // is > 0, so the divider correctly anchors at the first (all-unread) message.
    if (lastReadId < 0) return null;
    const first = messages.find((m) => m.id > lastReadId);
    return first ? first.id : null;
  }, [messages, lastReadId]);
}
