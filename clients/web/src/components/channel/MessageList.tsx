"use client";

import { useMemo, useCallback, useRef, useEffect, Fragment } from "react";
import { MessageSquare, ChevronDown, Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { MessageRow } from "./MessageRow";
import { UnreadDivider } from "./UnreadDivider";
import { computeGroupFlags } from "./message-grouping";
import { useMessageListScroll } from "./useMessageListScroll";
import { usePods } from "@/stores/pod";
import type { TransformedMessage } from "./types";
import type { MessageEditPayload } from "@/lib/viewModels/channelMessage";

interface MessageListProps {
  messages: TransformedMessage[];
  loading?: boolean;
  loadingMore?: boolean;
  hasMore?: boolean;
  error?: string | null;
  onLoadMore?: () => void;
  onRetry?: () => void;
  currentUserId?: number;
  channelId?: number;
  firstUnreadId?: number | null;
  roleByUserId?: Map<number, string>;
  onEditMessage?: (messageId: number, payload: MessageEditPayload) => Promise<void>;
  onDeleteMessage?: (messageId: number) => Promise<void>;
}

export function MessageList({
  messages, loading, loadingMore, hasMore, error,
  onLoadMore, onRetry, currentUserId, channelId, firstUnreadId, roleByUserId,
  onEditMessage, onDeleteMessage,
}: MessageListProps) {
  const t = useTranslations("channels.messages");
  const allPods = usePods();
  const {
    containerRef, bottomRef, isAtBottom, newMessageCount, mentionBelowId,
    handleScroll, scrollToBottom, scrollToMessage,
  } = useMessageListScroll({ messages, loading, loadingMore, channelId, firstUnreadId, currentUserId });

  const sentinelRef = useRef<HTMLDivElement>(null);
  const onLoadMoreRef = useRef(onLoadMore);
  useEffect(() => { onLoadMoreRef.current = onLoadMore; });

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !hasMore) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && !loadingMore) onLoadMoreRef.current?.();
      },
      { root: containerRef.current, rootMargin: "200px 0px 0px 0px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasMore, loadingMore, containerRef]);

  const formatDate = useCallback((dateString: string) => {
    const date = new Date(dateString);
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);
    if (date.toDateString() === today.toDateString()) return t("today");
    if (date.toDateString() === yesterday.toDateString()) return t("yesterday");
    return date.toLocaleDateString();
  }, [t]);

  const dateGroups = useMemo(() => {
    const result: { date: string; messages: TransformedMessage[] }[] = [];
    let currentDate = "";
    for (const msg of messages) {
      const msgDate = formatDate(msg.createdAt);
      if (msgDate !== currentDate) {
        currentDate = msgDate;
        result.push({ date: msgDate, messages: [msg] });
      } else {
        result[result.length - 1].messages.push(msg);
      }
    }
    return result;
  }, [messages, formatDate]);

  const groupFlags = useMemo(() => computeGroupFlags(messages), [messages]);

  return (
    <div className="relative min-h-0 flex-1">
      <div
        ref={containerRef}
        className="h-full overflow-y-auto py-2"
        onScroll={handleScroll}
        data-testid="channel-message-list"
      >
        {hasMore && <div ref={sentinelRef} className="h-1" />}
        {loadingMore && (
          <div className="flex justify-center py-3">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        )}

        {dateGroups.map((dateGroup) => (
          <div key={dateGroup.date}>
            <div className="flex justify-center py-2">
              <span className="text-[11px] text-muted-foreground">— {dateGroup.date} —</span>
            </div>
            {dateGroup.messages.map((message) => (
              <Fragment key={message.id}>
                {firstUnreadId === message.id && <UnreadDivider />}
                <MessageRow
                  message={message}
                  allPods={allPods}
                  currentUserId={currentUserId}
                  channelId={channelId}
                  isFirstInGroup={groupFlags.get(message.id) ?? true}
                  role={message.user ? roleByUserId?.get(message.user.id) : undefined}
                  onEditMessage={onEditMessage}
                  onDeleteMessage={onDeleteMessage}
                />
              </Fragment>
            ))}
          </div>
        ))}

        {error && !loading && messages.length === 0 && (
          <div className="flex h-full flex-col items-center justify-center text-muted-foreground">
            <MessageSquare className="mb-4 h-12 w-12 opacity-30" />
            <p className="text-sm text-destructive">{error}</p>
            {onRetry && (
              <button className="mt-2 text-xs text-primary hover:underline" onClick={onRetry}>
                {t("loadOlder")}
              </button>
            )}
          </div>
        )}

        {messages.length === 0 && !loading && !error && (
          <div className="flex h-full flex-col items-center justify-center text-muted-foreground">
            <MessageSquare className="mb-4 h-12 w-12 opacity-30" />
            <p className="text-sm">{t("noMessages")}</p>
            <p className="mt-1 text-xs">{t("startConversation")}</p>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {mentionBelowId != null && messages.some((m) => m.id === mentionBelowId) && (
        <button
          aria-label={t("mentionBelow")}
          className="absolute bottom-16 right-4 z-10 flex items-center gap-1 rounded-full bg-destructive px-3 py-1.5 text-[11px] font-medium text-destructive-foreground shadow-md transition-colors hover:bg-destructive/90"
          onClick={() => scrollToMessage(mentionBelowId)}
        >
          {t("mentionBelow")}
          <ChevronDown className="h-3.5 w-3.5" />
        </button>
      )}

      {!isAtBottom && messages.length > 0 && (
        <button
          aria-label={t("loadOlder")}
          className="absolute bottom-4 right-4 z-10 flex items-center gap-1 rounded-full bg-primary p-2 text-primary-foreground shadow-lg transition-colors hover:bg-primary/90"
          onClick={scrollToBottom}
        >
          <ChevronDown className="h-4 w-4" />
          {newMessageCount > 0 && (
            <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-destructive px-1 text-xs text-destructive-foreground">
              {newMessageCount}
            </span>
          )}
        </button>
      )}
    </div>
  );
}

export default MessageList;
