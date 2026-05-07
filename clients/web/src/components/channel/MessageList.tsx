"use client";

import { useMemo, useCallback, useRef, useEffect } from "react";
import { MessageSquare, ChevronDown, Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { MessageBubble } from "./MessageBubble";
import { ToolCallCard } from "./ToolCallCard";
import { AttachmentCard } from "./AttachmentCard";
import { useMessageListScroll } from "./useMessageListScroll";
import { getPodDisplayName, getShortPodKey } from "@/lib/pod-display-name";
import { usePods, type Pod } from "@/stores/pod";
import { cn } from "@/lib/utils";
import type { TransformedMessage } from "./types";
import type { MessageEditPayload } from "@/lib/api/channel-message-types";

interface MessageListProps {
  messages: TransformedMessage[];
  loading?: boolean;
  loadingMore?: boolean;
  hasMore?: boolean;
  error?: string | null;
  onLoadMore?: () => void;
  onRetry?: () => void;
  currentUserId?: number;
  onEditMessage?: (messageId: number, payload: MessageEditPayload) => Promise<void>;
  onDeleteMessage?: (messageId: number) => Promise<void>;
}

const AVATAR_PALETTE = [
  "bg-sky-500", "bg-emerald-500", "bg-amber-500", "bg-violet-500",
  "bg-rose-500", "bg-indigo-500", "bg-teal-500", "bg-orange-500",
];

function paletteFor(seed: string | number): string {
  const s = String(seed);
  let hash = 0;
  for (let i = 0; i < s.length; i++) hash = (hash * 31 + s.charCodeAt(i)) >>> 0;
  return AVATAR_PALETTE[hash % AVATAR_PALETTE.length];
}

function getSenderName(msg: TransformedMessage, allPods: Pod[]): string {
  if (msg.pod) {
    const storePod = allPods.find((p) => p.pod_key === msg.pod!.podKey);
    return getPodDisplayName(storePod ?? {
      pod_key: msg.pod.podKey, alias: msg.pod.alias,
      agent: msg.pod.agent ? { name: msg.pod.agent.name } : undefined,
    });
  }
  if (msg.user) return msg.user.name || msg.user.username || "Unknown";
  return "Unknown";
}

function formatTime(dateString: string) {
  return new Date(dateString).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function MessageList({
  messages, loading, loadingMore, hasMore, error,
  onLoadMore, onRetry, currentUserId, onEditMessage, onDeleteMessage,
}: MessageListProps) {
  const t = useTranslations("channels.messages");
  const allPods = usePods();
  const {
    containerRef, bottomRef, isAtBottom, newMessageCount,
    handleScroll, scrollToBottom,
  } = useMessageListScroll({ messages, loading, loadingMore });

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

  const renderMessage = (message: TransformedMessage) => {
    if (message.messageType === "system") {
      return (
        <div key={message.id} data-message-id={message.id} className="flex justify-center py-2">
          <span className="text-[11px] text-muted-foreground">{message.body}</span>
        </div>
      );
    }

    const isPod = !!message.pod;
    const senderName = getSenderName(message, allPods);
    const letter = senderName.charAt(0).toUpperCase() || "?";
    const time = formatTime(message.createdAt);
    const avatarBg = paletteFor(
      isPod ? (message.pod?.podKey ?? "") : (message.user?.id ?? senderName),
    );
    const isToolCall = message.content?.kind === "tool_call";

    return (
      <div
        key={message.id}
        data-message-id={message.id}
        className="group/msg flex gap-3 px-6 py-1.5 hover:bg-muted/30"
      >
        {/* Avatar — circular for users, square 6px radius for pods */}
        <span
          className={cn(
            "flex h-7 w-7 flex-shrink-0 items-center justify-center text-xs font-semibold text-white",
            avatarBg,
            isPod ? "rounded-md font-mono" : "rounded-full",
          )}
        >
          {letter}
        </span>

        <div className="flex min-w-0 flex-1 flex-col gap-1">
          {/* Header row */}
          <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
            {isPod ? (
              <>
                <span className="font-mono text-[13px] font-semibold text-foreground">
                  {getShortPodKey(message.pod!.podKey)}
                </span>
                {message.pod?.agent?.name && (
                  <span className="rounded border border-border bg-muted px-1.5 py-[1px] font-mono text-[10px] text-muted-foreground">
                    {message.pod.agent.name}
                  </span>
                )}
              </>
            ) : (
              <span className="text-[13px] font-semibold text-foreground">{senderName}</span>
            )}
            <span>{time}</span>
          </div>

          {/* Body */}
          {isToolCall ? (
            <ToolCallCard content={message.content!} />
          ) : (
            <>
              <MessageBubble
                message={message}
                isFirstInGroup
                formatTime={formatTime}
                currentUserId={currentUserId}
                onEdit={onEditMessage}
                onDelete={onDeleteMessage}
              />
              {message.content?.attachment_key && (
                <AttachmentCard url={message.content.attachment_key} />
              )}
            </>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="relative min-h-0 flex-1">
      <div
        ref={containerRef}
        className="h-full overflow-y-auto py-2"
        onScroll={handleScroll}
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
            {dateGroup.messages.map(renderMessage)}
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

      {!isAtBottom && messages.length > 0 && (
        <button
          aria-label={t("loadOlder")}
          className="absolute bottom-4 right-4 z-10 flex items-center gap-1 rounded-full bg-primary p-2 text-primary-foreground shadow-lg transition-all hover:bg-primary/90"
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
