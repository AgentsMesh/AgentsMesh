"use client";

import { useEffect, useRef, useMemo } from "react";
import { MessageSquare, Bot } from "lucide-react";
import { useTranslations } from "next-intl";
import { MessageBubble } from "./MessageBubble";
import type { TransformedMessage } from "./types";

interface MessageListProps {
  messages: TransformedMessage[];
  loading?: boolean;
  hasMore?: boolean;
  onLoadMore?: () => void;
  /** Current user ID for showing edit/delete on own messages */
  currentUserId?: number;
  /** Callback to edit a message */
  onEditMessage?: (messageId: number, content: string) => Promise<void>;
  /** Callback to delete a message */
  onDeleteMessage?: (messageId: number) => Promise<void>;
}

function getSenderName(msg: TransformedMessage): string {
  if (msg.pod) return msg.pod.agentType?.name || "Agent";
  if (msg.user) return msg.user.name || msg.user.username || "Unknown";
  return "Unknown";
}

export function MessageList({ messages, loading, hasMore, onLoadMore, currentUserId, onEditMessage, onDeleteMessage }: MessageListProps) {
  const t = useTranslations("channels.messages");
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages.length]);

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    if (date.toDateString() === today.toDateString()) return t("today");
    if (date.toDateString() === yesterday.toDateString()) return t("yesterday");
    return date.toLocaleDateString();
  };

  // Group messages by date only
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messages]);

  const renderMessage = (message: TransformedMessage) => {
    const isAgent = !!message.pod;

    if (message.messageType === "system") {
      return (
        <div key={message.id} className="flex justify-center py-2">
          <span className="text-xs text-muted-foreground bg-muted px-3 py-1 rounded-full">
            {message.content}
          </span>
        </div>
      );
    }

    return (
      <div
        key={message.id}
        className={`flex gap-3 py-1.5 px-4 -mx-4 hover:bg-muted/20 transition-colors ${isAgent ? "bg-muted/30" : ""}`}
      >
        {/* Avatar */}
        <div className="flex-shrink-0 pt-0.5">
          {message.user?.avatarUrl ? (
            /* eslint-disable-next-line @next/next/no-img-element */
            <img
              src={message.user.avatarUrl}
              alt={message.user.username}
              className="w-8 h-8 rounded-full"
            />
          ) : isAgent ? (
            <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
              <Bot className="w-4 h-4 text-primary-foreground" />
            </div>
          ) : (
            <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center">
              <span className="text-sm font-medium">
                {(getSenderName(message) || "?")[0].toUpperCase()}
              </span>
            </div>
          )}
        </div>

        {/* Name + time + content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <span className="font-medium text-sm">{getSenderName(message)}</span>
            {isAgent && message.pod && (
              <span className="text-xs text-muted-foreground">
                {message.pod.podKey.slice(0, 8)}
              </span>
            )}
            <span className="text-xs text-muted-foreground">
              {formatTime(message.createdAt)}
            </span>
          </div>
          <MessageBubble
            message={message}
            isFirstInGroup
            formatTime={formatTime}
            currentUserId={currentUserId}
            onEdit={onEditMessage}
            onDelete={onDeleteMessage}
          />
        </div>
      </div>
    );
  };

  return (
    <div ref={containerRef} className="flex-1 overflow-y-auto px-4 py-2">
      {hasMore && (
        <div className="text-center mb-4">
          <button
            className="text-sm text-primary hover:underline disabled:opacity-50"
            onClick={onLoadMore}
            disabled={loading}
          >
            {loading ? t("loading") : t("loadOlder")}
          </button>
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

      {messages.length === 0 && !loading && (
        <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
          <MessageSquare className="w-12 h-12 mb-4 opacity-30" />
          <p className="text-sm">{t("noMessages")}</p>
          <p className="text-xs mt-1">{t("startConversation")}</p>
        </div>
      )}

      <div ref={bottomRef} />
    </div>
  );
}

export default MessageList;
