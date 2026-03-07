"use client";

import { useState, useCallback } from "react";
import { Markdown } from "@/components/ui/markdown";
import { Button } from "@/components/ui/button";
import { Copy, Check, MoreHorizontal } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useTranslations } from "next-intl";
import type { TransformedMessage } from "./types";

interface MessageBubbleProps {
  message: TransformedMessage;
  /** Whether this is the first message in a sender group (shows full header) */
  isFirstInGroup: boolean;
  formatTime: (dateString: string) => string;
}

/**
 * Single message content renderer with Discord-style hover action bar.
 * Shows copy + more actions on hover in the top-right corner.
 */
export function MessageBubble({
  message,
  isFirstInGroup,
  formatTime,
}: MessageBubbleProps) {
  const t = useTranslations("channels.messages");
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(message.content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API not available
    }
  }, [message.content]);

  const isCode = message.messageType === "code";
  const isCommand = message.messageType === "command";

  return (
    <div className="group/msg relative">
      {/* Hover action bar — top-right corner */}
      <div className="absolute -top-3 right-2 hidden group-hover/msg:flex items-center gap-0.5 bg-background border border-border rounded-md shadow-sm z-10 px-0.5">
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6"
          onClick={handleCopy}
        >
          {copied ? (
            <Check className="w-3 h-3 text-green-500" />
          ) : (
            <Copy className="w-3 h-3" />
          )}
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="h-6 w-6">
              <MoreHorizontal className="w-3 h-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={handleCopy}>
              <Copy className="w-3.5 h-3.5 mr-2" />
              {t("copyMessage")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Message content */}
      <div className="flex items-start gap-3">
        {/* Time gutter — for non-first messages, show time on hover */}
        {!isFirstInGroup && (
          <span className="w-8 flex-shrink-0 text-[10px] text-muted-foreground opacity-0 group-hover/msg:opacity-100 transition-opacity pt-1 text-center tabular-nums">
            {formatTime(message.createdAt)}
          </span>
        )}

        <div className={`flex-1 min-w-0 ${isFirstInGroup ? "" : ""}`}>
          {isCode ? (
            <pre className="p-3 bg-muted rounded-md text-sm overflow-x-auto">
              <code>{message.content}</code>
            </pre>
          ) : isCommand ? (
            <div className="p-2 bg-muted rounded-md text-sm font-mono text-green-600 dark:text-green-400">
              $ {message.content}
            </div>
          ) : (
            <Markdown
              content={message.content}
              compact
              highlightMentions
              className="text-sm [&_p:first-child]:mt-0 [&_p:last-child]:mb-0"
            />
          )}

          {/* Metadata */}
          {message.metadata && Object.keys(message.metadata).length > 0 && (
            <div className="mt-1.5 text-xs text-muted-foreground">
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
    </div>
  );
}

export default MessageBubble;
