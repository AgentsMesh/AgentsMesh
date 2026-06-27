"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import { prefersReducedMotion } from "@/lib/scroll-behavior";
import type { TransformedMessage } from "./types";

interface UseMessageListScrollOptions {
  messages: TransformedMessage[];
  loading?: boolean;
  loadingMore?: boolean;
  channelId?: number;
  firstUnreadId?: number | null;
  currentUserId?: number;
}

interface UseMessageListScrollReturn {
  containerRef: React.RefObject<HTMLDivElement | null>;
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

export function useMessageListScroll({
  messages,
  loading,
  loadingMore,
  channelId,
  firstUnreadId,
  currentUserId,
}: UseMessageListScrollOptions): UseMessageListScrollReturn {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const [isAtBottom, setIsAtBottom] = useState(true);
  const [newMessageCount, setNewMessageCount] = useState(0);
  const [mentionBelowId, setMentionBelowId] = useState<number | null>(null);

  const handleScroll = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
    setIsAtBottom(atBottom);
    if (atBottom) {
      setNewMessageCount(0);
      setMentionBelowId(null);
    }
  }, []);

  const prevStateRef = useRef<{ length: number; firstId?: number }>({ length: 0 });
  const wasLoadingMoreRef = useRef(false);
  const initialLoadDone = useRef(false);

  // Channel switch happens without remount (the panel isn't keyed by id), so
  // reset entry refs — otherwise the new channel inherits the previous
  // channel's "already loaded" flag and never anchors or jumps to bottom.
  // newMessageCount / isAtBottom self-correct on the first scroll event.
  useEffect(() => {
    initialLoadDone.current = false;
    wasLoadingMoreRef.current = false;
    prevStateRef.current = { length: 0 };
  }, [channelId]);

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
      // On entry, position at the first unread (divider) when there is one;
      // otherwise jump straight to the latest message. No early return — the
      // prevStateRef sync below must run so the first post-entry message is
      // counted (and scanned for an @me) instead of being swallowed.
      const anchor = firstUnreadId != null
        ? containerRef.current?.querySelector("[data-unread-anchor]")
        : null;
      if (anchor) {
        anchor.scrollIntoView({ block: "start", behavior: "instant" as ScrollBehavior });
      } else {
        bottomRef.current?.scrollIntoView({ behavior: "instant" as ScrollBehavior });
      }
    } else if (messages.length > prev.length && prev.length > 0) {
      if (isAtBottom) {
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
  }, [messages, loadingMore, loading, isAtBottom, firstUnreadId, currentUserId]);

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
    containerRef, bottomRef, isAtBottom, newMessageCount, mentionBelowId,
    handleScroll, scrollToBottom, scrollToMessage,
  };
}
