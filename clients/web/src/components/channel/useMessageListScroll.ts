"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { prefersReducedMotion } from "@/lib/scroll-behavior";
import { useMessageEntryPosition } from "./useMessageEntryPosition";
import type { TransformedMessage } from "./types";

interface UseMessageListScrollOptions {
  messages: TransformedMessage[];
  loading?: boolean;
  loadingMore?: boolean;
  channelId?: number;
  firstUnreadId?: number | null;
  entryAnchorResolved?: boolean;
  currentUserId?: number;
}

interface UseMessageListScrollReturn {
  containerRef: React.RefObject<HTMLDivElement | null>;
  contentRef: React.RefObject<HTMLDivElement | null>;
  bottomRef: React.RefObject<HTMLDivElement | null>;
  isAtBottom: boolean;
  newMessageCount: number;
  mentionBelowId: number | null;
  handleScroll: () => void;
  scrollToBottom: () => void;
  scrollToMessage: (id: number) => void;
}

function mentionsMe(m: TransformedMessage, userId?: number): boolean {
  if (!m.mentions) return false;
  if (m.mentions.channel) return true;
  return userId != null && (m.mentions.users?.includes(userId) ?? false);
}

function isScrolledToBottom(el: HTMLDivElement | null): boolean {
  if (!el) return true;
  return el.scrollHeight - el.scrollTop - el.clientHeight < 50;
}

export function useMessageListScroll({
  messages,
  loading,
  loadingMore,
  channelId,
  firstUnreadId,
  entryAnchorResolved = true,
  currentUserId,
}: UseMessageListScrollOptions): UseMessageListScrollReturn {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  const [isAtBottom, setIsAtBottom] = useState(true);
  const [newMessageCount, setNewMessageCount] = useState(0);
  const [mentionBelowId, setMentionBelowId] = useState<number | null>(null);

  // The append-effect must decide follow-to-bottom from the position held BEFORE
  // the new message committed. Appends fire no scroll event, so this ref (synced
  // from the state that scroll + entry-anchor keep current) holds exactly that
  // pre-append truth — a live DOM read there would measure the post-append gap
  // and strand a tall message.
  const isAtBottomRef = useRef(isAtBottom);
  useEffect(() => { isAtBottomRef.current = isAtBottom; }, [isAtBottom]);

  const handleScroll = useCallback(() => {
    const atBottom = isScrolledToBottom(containerRef.current);
    setIsAtBottom(atBottom);
    if (atBottom) {
      setNewMessageCount(0);
      setMentionBelowId(null);
    }
  }, []);

  const prevStateRef = useRef<{ length: number; firstId?: number }>({ length: 0 });
  const wasLoadingMoreRef = useRef(false);
  const initialLoadDone = useRef(false);

  useEffect(() => {
    initialLoadDone.current = false;
    wasLoadingMoreRef.current = false;
    prevStateRef.current = { length: 0 };
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      // Default to at-bottom so a streamed message follows into view; real scroll
      // events (handleScroll) flip it to false once the user scrolls up. Entry
      // positioning owns WHERE the viewport lands (divider vs bottom); it must not
      // also suppress follow-to-bottom, or new messages silently stop appearing.
      setIsAtBottom(true);
      setNewMessageCount(0);
      setMentionBelowId(null);
    });
    return () => { cancelled = true; };
  }, [channelId]);

  useMessageEntryPosition({
    channelId,
    messages,
    loading,
    loadingMore,
    firstUnreadId,
    entryAnchorResolved,
    containerRef,
    contentRef,
    bottomRef,
  });

  useEffect(() => {
    if (loadingMore) wasLoadingMoreRef.current = true;
  }, [loadingMore]);

  useEffect(() => {
    const prev = prevStateRef.current;
    const firstId = messages.length > 0 ? messages[0].id : undefined;

    if (messages.length === 0 && prev.length > 0) {
      initialLoadDone.current = false;
      wasLoadingMoreRef.current = false;
      prevStateRef.current = { length: 0 };
      return;
    }

    if (wasLoadingMoreRef.current && !loadingMore) {
      wasLoadingMoreRef.current = false;
      if (prev.firstId != null && containerRef.current) {
        const el = containerRef.current.querySelector(`[data-message-id="${prev.firstId}"]`);
        if (el) el.scrollIntoView({ block: "start" });
      }
    } else if (!initialLoadDone.current && messages.length > 0 && !loading) {
      initialLoadDone.current = true;
    } else if (messages.length > prev.length && prev.length > 0) {
      // Follow the new message only if the user was at the bottom BEFORE it
      // arrived (isAtBottomRef, kept current by scroll + entry-anchor). A live
      // DOM read here would measure the post-append gap and wrongly strand a tall
      // message; on an unread channel the divider anchor keeps this false so the
      // message shows the pill instead of yanking down.
      if (isAtBottomRef.current) {
        bottomRef.current?.scrollIntoView({
          behavior: prefersReducedMotion() ? "instant" : "smooth",
        });
      } else {
        setNewMessageCount((c) => c + (messages.length - prev.length));
        const hit = messages.slice(prev.length).find((m) => mentionsMe(m, currentUserId));
        if (hit) setMentionBelowId(hit.id);
      }
    }

    prevStateRef.current = { length: messages.length, firstId };
  }, [messages, loadingMore, loading, currentUserId]);

  const scrollToBottom = useCallback(() => {
    bottomRef.current?.scrollIntoView({
      behavior: prefersReducedMotion() ? "instant" : "smooth",
    });
    setNewMessageCount(0);
    setMentionBelowId(null);
  }, []);

  const scrollToMessage = useCallback((id: number) => {
    const el = containerRef.current?.querySelector(`[data-message-id="${id}"]`);
    el?.scrollIntoView({ block: "center", behavior: prefersReducedMotion() ? "instant" : "smooth" });
    setMentionBelowId(null);
  }, []);

  return {
    containerRef, contentRef, bottomRef, isAtBottom, newMessageCount, mentionBelowId,
    handleScroll, scrollToBottom, scrollToMessage,
  };
}
