"use client";

import { useEffect, useLayoutEffect, useRef } from "react";
import type { RefObject } from "react";
import type { TransformedMessage } from "./types";
import { clearEntryTimers, createEntryState, scrollToEntryAnchor } from "./entryAnchorScroll";

interface UseMessageEntryPositionOptions {
  channelId?: number;
  messages: TransformedMessage[];
  loading?: boolean;
  loadingMore?: boolean;
  firstUnreadId?: number | null;
  entryAnchorResolved?: boolean;
  containerRef: RefObject<HTMLDivElement | null>;
  contentRef: RefObject<HTMLDivElement | null>;
  bottomRef: RefObject<HTMLDivElement | null>;
  // Fires when an anchor lands, so the caller can sync at-bottom state to the
  // real landed position (a divider at scrollTop 0 emits no correcting scroll).
  onAnchor?: () => void;
}

// Entry "settles" once content height holds steady this long (a ResizeObserver
// quiescence signal), so late images re-anchor and fast layouts settle early.
const REFLOW_QUIESCE_MS = 250;

export function useMessageEntryPosition({
  channelId,
  messages,
  loading,
  loadingMore,
  firstUnreadId,
  entryAnchorResolved = true,
  containerRef,
  contentRef,
  bottomRef,
  onAnchor,
}: UseMessageEntryPositionOptions) {
  const firstId = messages[0]?.id ?? null;
  const lastId = messages[messages.length - 1]?.id ?? null;
  // Anchor identity omits the tail id / length on purpose: an appended message
  // must not re-fire entry positioning (that yanks back to the divider). Only a
  // window swap (firstId) or moved cursor (firstUnreadId) is a real re-anchor.
  const anchorKey = `${channelId ?? "none"}:${firstId ?? "none"}:${firstUnreadId ?? "read"}`;
  const stateRef = useRef(createEntryState());

  // Live inputs the ResizeObserver reads; synced in a layout effect so they are
  // fresh before the pre-paint observer/RAF fire (a passive effect lags a frame).
  const liveRef = useRef({ loading, loadingMore, hasMessages: messages.length > 0, firstUnreadId, entryAnchorResolved });
  useLayoutEffect(() => {
    liveRef.current = { loading, loadingMore, hasMessages: messages.length > 0, firstUnreadId, entryAnchorResolved };
  }, [loading, loadingMore, messages.length, firstUnreadId, entryAnchorResolved]);

  useEffect(() => {
    const state = stateRef.current;
    clearEntryTimers(state);
    Object.assign(state, createEntryState());
    return () => clearEntryTimers(state);
  }, [channelId]);

  useEffect(() => {
    if (!loading) return;
    const state = stateRef.current;
    clearEntryTimers(state);
    state.settled = false;
    state.lastAppliedKey = null;
  }, [loading]);

  // Intent is derived from real scroll position: any scroll that isn't our own
  // programmatic re-anchor (native scrollbar, keyboard, momentum) = user takeover.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const onScroll = () => {
      const state = stateRef.current;
      if (state.expectedScrollTop != null && Math.abs(container.scrollTop - state.expectedScrollTop) <= 2) {
        state.expectedScrollTop = null;
        return;
      }
      state.userInterrupted = true;
      state.settled = true;
      clearEntryTimers(state);
    };
    container.addEventListener("scroll", onScroll, { passive: true });
    return () => container.removeEventListener("scroll", onScroll);
  }, [channelId, containerRef]);

  useEffect(() => {
    const state = stateRef.current;
    if (!channelId || loading || loadingMore || messages.length === 0 || state.userInterrupted || state.settled) return;
    // Wait for the unread cursor to resolve before the first anchor, so an unknown
    // cursor doesn't land at the bottom and then jump to the divider on resolve.
    if (!entryAnchorResolved) return;
    if (state.lastAppliedKey === anchorKey) {
      // Already anchored this window; a grown tail = live traffic started — settle
      // so the stream flows / shows the pill instead of being dragged back.
      if (state.anchoredLastId != null && lastId !== state.anchoredLastId) state.settled = true;
      return;
    }
    // A load-more prepend changes firstId (→ anchorKey) but keeps the same tail;
    // adopt the new key WITHOUT re-anchoring — useMessageListScroll's load-more
    // restore owns that viewport, and re-anchoring would fight it.
    if (state.lastAppliedKey !== null && lastId === state.anchoredLastId) {
      state.lastAppliedKey = anchorKey;
      return;
    }

    const target = scrollToEntryAnchor(containerRef.current, bottomRef.current, firstUnreadId, state);
    if (!target) return;
    state.lastAppliedKey = anchorKey;
    state.anchoredLastId = lastId;
    state.target = target;
    onAnchor?.();
  }, [anchorKey, bottomRef, channelId, containerRef, entryAnchorResolved, firstUnreadId, lastId, loading, loadingMore, messages.length, onAnchor]);

  // Re-anchor on content growth (late images/embeds) and mark settled once the
  // reflow goes quiet — a debounced quiescence signal, reset on every resize.
  useEffect(() => {
    const content = contentRef.current;
    if (!content || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(() => {
      const state = stateRef.current;
      const live = liveRef.current;
      if (state.userInterrupted || live.loading || live.loadingMore || !live.hasMessages || !live.entryAnchorResolved) return;
      if (state.quiesceTimer) clearTimeout(state.quiesceTimer);
      state.quiesceTimer = setTimeout(() => {
        state.quiesceTimer = null;
        state.settled = true;
      }, REFLOW_QUIESCE_MS);
      if (state.resizeFrame != null) cancelAnimationFrame(state.resizeFrame);
      state.resizeFrame = requestAnimationFrame(() => {
        state.resizeFrame = null;
        if (state.userInterrupted || state.settled) return;
        const target = scrollToEntryAnchor(containerRef.current, bottomRef.current, liveRef.current.firstUnreadId, state);
        if (target) {
          state.target = target;
          onAnchor?.();
        }
      });
    });
    observer.observe(content);
    return () => {
      observer.disconnect();
      const state = stateRef.current;
      if (state.resizeFrame != null) cancelAnimationFrame(state.resizeFrame);
      state.resizeFrame = null;
    };
  }, [bottomRef, channelId, containerRef, contentRef, onAnchor]);
}
