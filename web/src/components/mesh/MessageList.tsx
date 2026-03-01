"use client";

import { useEffect, useRef } from "react";
import { Markdown } from "@/components/ui/markdown";

interface Message {
  id: number;
  content: string;
  messageType: "text" | "system" | "code" | "command";
  metadata?: Record<string, unknown>;
  createdAt: string;
  pod?: {
    podKey: string;
    agentType?: {
      name: string;
    };
  };
  user?: {
    id: number;
    username: string;
    name?: string;
    avatarUrl?: string;
  };
}

interface MessageListProps {
  messages: Message[];
  loading?: boolean;
  hasMore?: boolean;
  onLoadMore?: () => void;
}

export function MessageList({
  messages,
  loading,
  hasMore,
  onLoadMore,
}: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: "smooth" });
    }
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

    if (date.toDateString() === today.toDateString()) {
      return "Today";
    } else if (date.toDateString() === yesterday.toDateString()) {
      return "Yesterday";
    }
    return date.toLocaleDateString();
  };

  // Group messages by date
  const groupedMessages: { date: string; messages: Message[] }[] = [];
  let currentDate = "";

  messages.forEach((msg) => {
    const msgDate = formatDate(msg.createdAt);
    if (msgDate !== currentDate) {
      currentDate = msgDate;
      groupedMessages.push({ date: msgDate, messages: [msg] });
    } else {
      groupedMessages[groupedMessages.length - 1].messages.push(msg);
    }
  });

  const renderMessage = (message: Message) => {
    const isAgent = !!message.pod;
    const isSystem = message.messageType === "system";
    const isCode = message.messageType === "code";
    const isCommand = message.messageType === "command";

    if (isSystem) {
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
        className={`flex gap-3 py-2 ${isAgent ? "bg-muted/30 -mx-4 px-4" : ""}`}
      >
        {/* Avatar */}
        <div className="flex-shrink-0">
          {message.user?.avatarUrl ? (
            /* eslint-disable-next-line @next/next/no-img-element */
            <img
              src={message.user.avatarUrl}
              alt={message.user.username}
              className="w-8 h-8 rounded-full"
            />
          ) : isAgent ? (
            <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
              <svg
                className="w-4 h-4 text-primary-foreground"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
                />
              </svg>
            </div>
          ) : (
            <div className="w-8 h-8 rounded-full bg-muted flex items-center justify-center">
              <span className="text-sm font-medium">
                {(message.user?.name || message.user?.username || "?")[0].toUpperCase()}
              </span>
            </div>
          )}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <span className="font-medium text-sm">
              {isAgent
                ? message.pod?.agentType?.name || "Agent"
                : message.user?.name || message.user?.username || "Unknown"}
            </span>
            {isAgent && (
              <span className="text-xs text-muted-foreground">
                {message.pod?.podKey.slice(0, 8)}
              </span>
            )}
            <span className="text-xs text-muted-foreground">
              {formatTime(message.createdAt)}
            </span>
          </div>

          {isCode ? (
            <pre className="mt-1 p-3 bg-muted rounded-md text-sm overflow-x-auto">
              <code>{message.content}</code>
            </pre>
          ) : isCommand ? (
            <div className="mt-1 p-2 bg-muted rounded-md text-sm font-mono text-green-600 dark:text-green-400">
              $ {message.content}
            </div>
          ) : (
            <Markdown
              content={message.content}
              compact
              highlightMentions
              className="mt-1 text-sm [&_p:first-child]:mt-0 [&_p:last-child]:mb-0"
            />
          )}

          {/* Metadata */}
          {message.metadata && Object.keys(message.metadata).length > 0 && (
            <div className="mt-2 text-xs text-muted-foreground">
              <details>
                <summary className="cursor-pointer hover:text-foreground">
                  Metadata
                </summary>
                <pre className="mt-1 p-2 bg-muted rounded text-xs overflow-x-auto">
                  {JSON.stringify(message.metadata, null, 2)}
                </pre>
              </details>
            </div>
          )}
        </div>
      </div>
    );
  };

  return (
    <div ref={containerRef} className="flex-1 overflow-y-auto p-4">
      {/* Load More */}
      {hasMore && (
        <div className="text-center mb-4">
          <button
            className="text-sm text-primary hover:underline disabled:opacity-50"
            onClick={onLoadMore}
            disabled={loading}
          >
            {loading ? "Loading..." : "Load older messages"}
          </button>
        </div>
      )}

      {/* Messages */}
      {groupedMessages.map((group) => (
        <div key={group.date}>
          {/* Date Separator */}
          <div className="flex items-center gap-4 my-4">
            <div className="flex-1 border-t" />
            <span className="text-xs text-muted-foreground">{group.date}</span>
            <div className="flex-1 border-t" />
          </div>

          {/* Messages for this date */}
          {group.messages.map(renderMessage)}
        </div>
      ))}

      {/* Empty State */}
      {messages.length === 0 && !loading && (
        <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
          <svg
            className="w-12 h-12 mb-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
            />
          </svg>
          <p className="text-sm">No messages yet</p>
          <p className="text-xs mt-1">Start the conversation!</p>
        </div>
      )}

      {/* Scroll anchor */}
      <div ref={bottomRef} />
    </div>
  );
}

export default MessageList;
