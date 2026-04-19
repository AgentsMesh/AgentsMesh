"use client";

import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useTranslations } from "next-intl";
import { MessageActions } from "./MessageActions";
import { StructuredContent } from "./StructuredContent";
import { buildMessageContent, extractMentionMap } from "./message-content-builder";
import type { TransformedMessage } from "./types";
import type { MessageContent } from "@/lib/api/channel-message-types";

interface MessageBubbleProps {
  message: TransformedMessage;
  isFirstInGroup: boolean;
  formatTime: (dateString: string) => string;
  currentUserId?: number;
  onEdit?: (messageId: number, content: MessageContent) => Promise<void>;
  onDelete?: (messageId: number) => Promise<void>;
}

export function MessageBubble({
  message, isFirstInGroup, formatTime, currentUserId, onEdit, onDelete,
}: MessageBubbleProps) {
  const t = useTranslations("channels.messages");
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState(message.body);

  const isOwnMessage = currentUserId != null && message.user?.id === currentUserId;

  const handleEditSubmit = useCallback(async () => {
    if (!onEdit || editContent.trim() === "" || editContent === message.body) {
      setEditing(false);
      return;
    }
    try {
      const mentionMap = extractMentionMap(message.content);
      const content = buildMessageContent(editContent.trim(), mentionMap);
      await onEdit(message.id, content);
      setEditing(false);
    } catch { /* Error handled by store */ }
  }, [onEdit, editContent, message.id, message.body, message.content]);

  const handleEditKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.nativeEvent.isComposing) return;
      if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); handleEditSubmit(); }
      if (e.key === "Escape") { setEditing(false); setEditContent(message.body); }
    },
    [handleEditSubmit, message.body]
  );

  return (
    <div className="group/msg relative">
      <MessageActions
        messageId={message.id}
        content={message.body}
        isOwnMessage={isOwnMessage}
        onEdit={onEdit ? () => Promise.resolve() : undefined}
        onDelete={onDelete}
        onStartEdit={() => { setEditing(true); setEditContent(message.body); }}
      />

      <div className="flex items-start gap-3">
        {!isFirstInGroup && (
          <span className="w-8 flex-shrink-0 text-[10px] text-muted-foreground opacity-0 group-hover/msg:opacity-100 transition-opacity pt-1 text-center tabular-nums">
            {formatTime(message.createdAt)}
          </span>
        )}

        <div className="flex-1 min-w-0">
          {editing ? (
            <div className="space-y-2">
              <Textarea
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                onKeyDown={handleEditKeyDown}
                className="min-h-[60px] text-sm"
                autoFocus
              />
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Button size="sm" variant="default" className="h-6 text-xs" onClick={handleEditSubmit}>
                  {t("save")}
                </Button>
                <Button size="sm" variant="ghost" className="h-6 text-xs"
                  onClick={() => { setEditing(false); setEditContent(message.body); }}>
                  {t("cancel")}
                </Button>
                <span>{t("editHint")}</span>
              </div>
            </div>
          ) : message.content ? (
            <div>
              <StructuredContent content={message.content}
                className="text-sm [&_p:first-child]:mt-0 [&_p:last-child]:mb-0" />
              {message.editedAt && (
                <span className="text-[10px] text-muted-foreground ml-1">({t("edited")})</span>
              )}
            </div>
          ) : (
            <div>
              <p className="text-sm">{message.body}</p>
              {message.editedAt && (
                <span className="text-[10px] text-muted-foreground ml-1">({t("edited")})</span>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default MessageBubble;
