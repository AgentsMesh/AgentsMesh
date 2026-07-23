"use client";

import { useEffect, useMemo, useState } from "react";
import { getChannelState } from "@/lib/wasm-core";
import type { TransformedMessage } from "./types";

export interface ChannelEntryAnchor {
  firstUnreadId: number | null;
  resolved: boolean;
  needsUnreadSummary: boolean;
}

interface EntrySnapshot {
  channelId: number | null;
  lastReadId: number | null;
  unknownResolved: boolean;
}

function readCursor(channelId: number): number | null {
  const lastReadId = getChannelState().get_last_read_id(BigInt(channelId));
  return lastReadId >= 0 ? lastReadId : null;
}

function makeSnapshot(channelId: number): EntrySnapshot {
  const cursor = channelId ? readCursor(channelId) : null;
  return {
    channelId,
    lastReadId: cursor,
    unknownResolved: cursor != null || !channelId,
  };
}

export function useChannelEntryAnchor(
  channelId: number,
  messages: TransformedMessage[],
  unreadSummaryAttempted = false,
): ChannelEntryAnchor {
  const [snapshot, setSnapshot] = useState<EntrySnapshot>(() => makeSnapshot(channelId));

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) setSnapshot(makeSnapshot(channelId));
    });
    return () => { cancelled = true; };
  }, [channelId]);

  useEffect(() => {
    if (!channelId || snapshot.channelId !== channelId || snapshot.lastReadId != null || snapshot.unknownResolved) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      const cursor = readCursor(channelId);
      setSnapshot((current) => {
        if (current.channelId !== channelId || current.lastReadId != null || current.unknownResolved) return current;
        if (cursor != null) return { channelId, lastReadId: cursor, unknownResolved: true };
        return unreadSummaryAttempted ? { ...current, unknownResolved: true } : current;
      });
    });
    return () => { cancelled = true; };
    // Retry re-runs when unreadSummaryAttempted flips after fetchUnreadCounts;
    // that fetch writes the Rust cursor BEFORE resolving, so we read the fresh
    // value here without subscribing to the global _unreadTick (which would
    // re-run this on unread changes for unrelated channels).
  }, [channelId, snapshot.channelId, snapshot.lastReadId, snapshot.unknownResolved, unreadSummaryAttempted]);

  return useMemo(() => {
    if (!channelId) return { firstUnreadId: null, resolved: true, needsUnreadSummary: false };
    if (snapshot.channelId !== channelId) return { firstUnreadId: null, resolved: false, needsUnreadSummary: false };
    if (snapshot.lastReadId == null) {
      return {
        firstUnreadId: null,
        resolved: snapshot.unknownResolved,
        needsUnreadSummary: !snapshot.unknownResolved,
      };
    }
    const first = messages.find((m) => m.id > snapshot.lastReadId!);
    return {
      firstUnreadId: first ? first.id : null,
      resolved: true,
      needsUnreadSummary: false,
    };
  }, [channelId, messages, snapshot]);
}
