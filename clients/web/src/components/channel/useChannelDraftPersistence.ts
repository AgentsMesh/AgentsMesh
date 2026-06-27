"use client";

import { useEffect, useRef } from "react";
import { getChannelDraft, setChannelDraft } from "@/stores/useChannelDraft";

/**
 * Per-channel composer draft: restore on entry, persist the outgoing channel's
 * text on switch, debounce-save while typing (survives reload), and save on
 * unmount. Client-local (localStorage via useChannelDraft); see [[useChannelDraft]].
 */
export function useChannelDraftPersistence(
  channelId: number | null | undefined,
  content: string,
  setContent: (text: string) => void,
) {
  const contentRef = useRef(content);
  contentRef.current = content;
  const loadedForRef = useRef<number | null>(null);

  useEffect(() => {
    const prev = loadedForRef.current;
    if (prev != null && prev !== channelId) setChannelDraft(prev, contentRef.current);
    setContent(channelId != null ? getChannelDraft(channelId) : "");
    loadedForRef.current = channelId ?? null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [channelId]);

  useEffect(() => {
    const ch = loadedForRef.current;
    if (ch == null) return;
    // Snapshot the text WITH its channel — reading contentRef at fire time can
    // pair the wrong channel with the wrong text across a fast switch (one
    // channel's draft bleeds into another).
    const text = content;
    const id = setTimeout(() => setChannelDraft(ch, text), 400);
    return () => clearTimeout(id);
  }, [content]);

  useEffect(
    () => () => {
      const ch = loadedForRef.current;
      // Persist a NON-EMPTY draft on unmount only — never delete here. Deletion
      // is the channelId-change save's job; guarding this stops React
      // StrictMode's mount→cleanup→mount from wiping the draft with the stale
      // pre-render empty contentRef.
      if (ch != null && contentRef.current.trim()) setChannelDraft(ch, contentRef.current);
    },
    [],
  );
}
