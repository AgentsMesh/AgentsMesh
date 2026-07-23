"use client";

import { useCallback, useEffect, useRef } from "react";

interface UseChannelEntryMarkReadOptions {
  channelId: number;
  lastMessageId: number | null;
  entryAnchorResolved: boolean;
  markRead: (channelId: number, messageId: number) => Promise<void>;
}

interface PendingMarkRead {
  channelId: number;
  messageId: number;
}

export function useChannelEntryMarkRead({
  channelId,
  lastMessageId,
  entryAnchorResolved,
  markRead,
}: UseChannelEntryMarkReadOptions) {
  const prevLastMsgIdRef = useRef<number | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingRef = useRef<PendingMarkRead | null>(null);
  // The 300ms debounce may elapse before the entry anchor is resolved; the
  // timer parks the pending here so we don't advance the read cursor before the
  // divider is frozen. A separate effect flushes it once resolved.
  const debounceElapsedRef = useRef(false);
  // Read the latest resolved state from the timer callback rather than its
  // captured closure value, which may be stale by the time it fires.
  const resolvedRef = useRef(entryAnchorResolved);
  useEffect(() => { resolvedRef.current = entryAnchorResolved; }, [entryAnchorResolved]);

  const flush = useCallback(() => {
    const pending = pendingRef.current;
    pendingRef.current = null;
    debounceElapsedRef.current = false;
    if (pending) markRead(pending.channelId, pending.messageId);
  }, [markRead]);

  useEffect(() => {
    return () => {
      // On unmount / channel switch, flush whatever is pending — even if the
      // anchor never resolved (unknown cursor + quick leave) — so the channel
      // still gets marked read.
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = null;
      flush();
    };
  }, [channelId, flush]);

  useEffect(() => {
    prevLastMsgIdRef.current = null;
    debounceElapsedRef.current = false;
  }, [channelId]);

  // Arm the pending mark-read as soon as a new last message appears — NOT gated
  // on the anchor, so a quick leave still flushes. Advancing the cursor is what
  // waits for resolution (see the timer callback + the resolve effect below).
  useEffect(() => {
    if (lastMessageId === null || lastMessageId === prevLastMsgIdRef.current) return;
    prevLastMsgIdRef.current = lastMessageId;
    if (timerRef.current) clearTimeout(timerRef.current);
    pendingRef.current = { channelId, messageId: lastMessageId };
    debounceElapsedRef.current = false;
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      debounceElapsedRef.current = true;
      if (resolvedRef.current) flush();
    }, 300);
  }, [channelId, lastMessageId, flush]);

  // If the debounce already elapsed while the anchor was still unresolved,
  // flush the moment it resolves.
  useEffect(() => {
    if (entryAnchorResolved && debounceElapsedRef.current) flush();
  }, [entryAnchorResolved, flush]);
}
