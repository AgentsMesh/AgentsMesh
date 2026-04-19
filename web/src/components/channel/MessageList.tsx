"use client";

import { useMemo, useCallback, useRef, useEffect } from "react";
import { MessageSquare, Bot, ChevronDown, Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { MessageBubble } from "./MessageBubble";
import { useMessageListScroll } from "./useMessageListScroll";
import { getPodDisplayName, getShortPodKey } from "@/lib/pod-display-name";
import { usePodStore, type Pod } from "@/stores/pod";
import type { TransformedMessage } from "./types";
import type { MessageContent } from "@/lib/api/channel-message-types";

interface MessageListProps {
  messages: TransformedMessage[];
  loading?: boolean;
  loadingMore?: boolean;
  hasMore?: boolean;
  error?: string | null;
  onLoadMore?: () => void;
  onRetry?: () => void;
  currentUserId?: number;
  onEditMessage?: (messageId: number, content: MessageContent) => Promise<void>;
  onDeleteMessage?: (messageId: number) => Promise<void>;
}

function getSenderName(msg: TransformedMessage, allPods: Pod[]): string {
  if (msg.pod) {
    const storePod = allPods.find((p) => p.pod_key === msg.pod!.podKey);
    return getPodDisplayName(storePod ?? {
      pod_key: msg.pod.podKey,
      alias: msg.pod.alias,
      agent: msg.pod.agent ? { name: msg.pod.agent.name } : undefined,
    });
  }
  if (msg.user) return msg.user.name || msg.user.username || "Unknown";
  return "Unknown";
}

function formatTime(dateString: string) {
  const date = new Date(dateString);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function MessageList({
  messages,
  loading,
  loadingMore,
  hasMore,
  error,
  onLoadMore,
  onRetry,
  currentUserId,
  onEditMessage,
  onDeleteMessage,
}: MessageListProps) {
  const t = useTranslations("channels.messages");
  const allPods = usePodStore((s) => s.pods);
  const {
    containerRef, bottomRef, isAtBottom, newMessageCount,
    handleScroll, scrollToBottom,
  } = useMessageListScroll({ messages, loading, loadingMore });

  // IntersectionObserver: auto-load older messages when sentinel enters viewport
  const sentinelRef = useRef<HTMLDivElement>(null);
  // Keep ref in sync with latest callback (intentionally no deps — runs every render)
  const onLoadMoreRef = useRef(onLoadMore);
  useEffect(() => { onLoadMoreRef.current = onLoadMore; });

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !hasMore) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && !loadingMore) {
          onLoadMoreRef.current?.();
        }
      },
      { root: containerRef.current, rootMargin: "200px 0px 0px 0px" }
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

  const renderMessage = (message: TransformedMessage) => {
    const isAgent = !!message.pod;

    if (message.messageType === "system") {
      return (
        <div key={message.id} data-message-id={message.id} className="flex justify-center py-2">
          <span className="text-xs text-muted-foreground bg-muted px-3 py-1 rounded-full">
            {message.body}
          </span>
        </div>
      );
    }

    return (
      <div
        key={message.id}
        data-message-id={message.id}
        className={`flex gap-3 py-1.5 px-4 -mx-4 hover:bg-muted/20 transition-colors ${isAgent ? "bg-muted/30" : ""}`}
      >
        <div className="flex-shrink-0 pt-0.5">
          {message.user?.avatarUrl ? (
            /* eslint-disable-next-line @next/next/no-img-element */
            <img src={message.user.avatarUrl} alt={message.user.username} className="w-8 h-8 rounded-full" />
          ) : isAgent ? (
            <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
              <Bot className="w-4 h-4 text-primary-foreground" />
            </div>
          ) : (
            <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center">
              <span className="text-sm font-medium">{(getSenderName(message, allPods) || "?")[0].toUpperCase()}</span>
            </div>
          )}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <span className="font-medium text-sm">{getSenderName(message, allPods)}</span>
            {isAgent && message.pod && (
              <span className="text-xs text-muted-foreground">{getShortPodKey(message.pod.podKey)}</span>
            )}
            <span className="text-xs text-muted-foreground">{formatTime(message.createdAt)}</span>
          </div>
          <MessageBubble
            message={message} isFirstInGroup formatTime={formatTime}
            currentUserId={currentUserId} onEdit={onEditMessage} onDelete={onDeleteMessage}
          />
        </div>
      </div>
    );
  };

  return (
    <div className="relative flex-1 min-h-0">
      <div ref={containerRef} className="h-full overflow-y-auto px-4 py-2" onScroll={handleScroll}>
        {/* Sentinel for IntersectionObserver auto-load */}
        {hasMore && <div ref={sentinelRef} className="h-1" />}
        {loadingMore && (
          <div className="flex justify-center py-3">
            <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
          </div>
        )}

        {dateGroups.map((dateGroup) => (
          <div key={dateGroup.date}>
            <div className="flex items-center gap-4 my-4">
              <div className="flex-1 border-t" />
              <span className="text-xs text-muted-foreground font-medium">{dateGroup.date}</span>
              <div className="flex-1 border-t" />
            </div>
            {dateGroup.messages.map(renderMessage)}
          </div>
        ))}

        {error && !loading && messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <MessageSquare className="w-12 h-12 mb-4 opacity-30" />
            <p className="text-sm text-destructive">{error}</p>
            {onRetry && (
              <button className="text-xs text-primary hover:underline mt-2" onClick={onRetry}>
                {t("loadOlder")}
              </button>
            )}
          </div>
        )}

        {messages.length === 0 && !loading && !error && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <MessageSquare className="w-12 h-12 mb-4 opacity-30" />
            <p className="text-sm">{t("noMessages")}</p>
            <p className="text-xs mt-1">{t("startConversation")}</p>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {!isAtBottom && messages.length > 0 && (
        <button
          aria-label={t("loadOlder")}
          className="absolute bottom-4 right-4 bg-primary text-primary-foreground
                     rounded-full p-2 shadow-lg hover:bg-primary/90 transition-all
                     flex items-center gap-1 z-10"
          onClick={scrollToBottom}
        >
          <ChevronDown className="w-4 h-4" />
          {newMessageCount > 0 && (
            <span className="bg-destructive text-destructive-foreground
                            text-xs rounded-full min-w-5 h-5 flex items-center justify-center px-1">
              {newMessageCount}
            </span>
          )}
        </button>
      )}
    </div>
  );
}

export default MessageList;
