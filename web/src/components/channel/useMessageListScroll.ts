"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import type { TransformedMessage } from "./types";

interface UseMessageListScrollOptions {
  messages: TransformedMessage[];
  loading?: boolean;
  loadingMore?: boolean;
}

interface UseMessageListScrollReturn {
  containerRef: React.RefObject<HTMLDivElement | null>;
  bottomRef: React.RefObject<HTMLDivElement | null>;
  isAtBottom: boolean;
  newMessageCount: number;
  handleScroll: () => void;
  scrollToBottom: () => void;
}

/**
 * IM-standard scroll behavior for message lists.
 * Handles: stick-to-bottom, scroll position restoration on load-more,
 * new message counting, and jump-to-latest.
 */
export function useMessageListScroll({
  messages,
  loading,
  loadingMore,
}: UseMessageListScrollOptions): UseMessageListScrollReturn {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const [isAtBottom, setIsAtBottom] = useState(true);
  const [newMessageCount, setNewMessageCount] = useState(0);

  // Reset count directly in scroll handler (avoids effect-based setState)
  const handleScroll = useCallback(() => {
    const el = containerRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
    setIsAtBottom(atBottom);
    if (atBottom) setNewMessageCount(0);
  }, []);

  const prevStateRef = useRef<{ length: number; firstId?: number }>({ length: 0 });
  const wasLoadingMoreRef = useRef(false);
  const initialLoadDone = useRef(false);

  useEffect(() => {
    if (loadingMore) wasLoadingMoreRef.current = true;
  }, [loadingMore]);

  useEffect(() => {
    const prev = prevStateRef.current;
    const firstId = messages.length > 0 ? messages[0].id : undefined;

    // Channel switch: messages became empty — reset all tracking state
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
      bottomRef.current?.scrollIntoView({ behavior: "instant" as ScrollBehavior });
    } else if (messages.length > prev.length && prev.length > 0) {
      if (isAtBottom) {
        bottomRef.current?.scrollIntoView({ behavior: "smooth" });
      } else {
        setNewMessageCount((c) => c + (messages.length - prev.length));
      }
    }

    prevStateRef.current = { length: messages.length, firstId };
  }, [messages, loadingMore, loading, isAtBottom]);

  const scrollToBottom = useCallback(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
    setNewMessageCount(0);
  }, []);

  return { containerRef, bottomRef, isAtBottom, newMessageCount, handleScroll, scrollToBottom };
}
