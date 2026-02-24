"use client";

import { useState, useRef, useCallback } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Send, X } from "lucide-react";
import { MentionPopover } from "./MentionPopover";

interface CommentInputProps {
  /** Called when the comment is submitted */
  onSubmit: (
    content: string,
    mentions: Array<{ user_id: number; username: string }>
  ) => Promise<void>;
  /** If replying to a comment, show the username being replied to */
  replyTo?: { id: number; username: string };
  /** Cancel reply mode */
  onCancelReply?: () => void;
  /** Placeholder text */
  placeholder?: string;
  /** Initial content for edit mode */
  initialContent?: string;
  /** Called when cancel is clicked in edit mode */
  onCancel?: () => void;
}

/**
 * Comment input component with @mention support.
 */
export function CommentInput({
  onSubmit,
  replyTo,
  onCancelReply,
  placeholder,
  initialContent,
  onCancel,
}: CommentInputProps) {
  const t = useTranslations();
  const isEditMode = initialContent !== undefined;
  const [content, setContent] = useState(initialContent || "");
  const [submitting, setSubmitting] = useState(false);
  const [mentions, setMentions] = useState<
    Array<{ user_id: number; username: string }>
  >([]);

  // Mention popover state
  const [mentionVisible, setMentionVisible] = useState(false);
  const [mentionQuery, setMentionQuery] = useState("");
  const [mentionPosition, setMentionPosition] = useState({ top: 0, left: 0 });
  const [mentionStartIndex, setMentionStartIndex] = useState(-1);

  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    const cursorPos = e.target.selectionStart || 0;
    setContent(value);

    // Detect @ trigger
    const textBeforeCursor = value.slice(0, cursorPos);
    const atIndex = textBeforeCursor.lastIndexOf("@");

    if (atIndex >= 0) {
      const charBeforeAt = atIndex > 0 ? textBeforeCursor[atIndex - 1] : " ";
      const textAfterAt = textBeforeCursor.slice(atIndex + 1);

      // Only trigger if @ is at start or preceded by whitespace, and no space after @
      if (
        (charBeforeAt === " " || charBeforeAt === "\n" || atIndex === 0) &&
        !textAfterAt.includes(" ")
      ) {
        setMentionQuery(textAfterAt);
        setMentionStartIndex(atIndex);
        setMentionVisible(true);

        // Position the popover near the textarea
        if (textareaRef.current) {
          const rect = textareaRef.current.getBoundingClientRect();
          setMentionPosition({
            top: rect.height + 4,
            left: 0,
          });
        }
        return;
      }
    }

    setMentionVisible(false);
  };

  const handleMentionSelect = useCallback(
    (username: string) => {
      if (mentionStartIndex < 0 || !textareaRef.current) return;

      const before = content.slice(0, mentionStartIndex);
      const cursorPos = textareaRef.current.selectionStart || content.length;
      const after = content.slice(cursorPos);

      const newContent = `${before}@${username} ${after}`;
      setContent(newContent);
      setMentionVisible(false);

      // Track mention (avoid duplicates)
      setMentions((prev) => {
        if (prev.some((m) => m.username === username)) return prev;
        // We don't have user_id here from the popover; it will be resolved by the parent via member list
        return [...prev, { user_id: 0, username }];
      });

      // Refocus textarea
      setTimeout(() => {
        if (textareaRef.current) {
          const newPos = mentionStartIndex + username.length + 2; // @username + space
          textareaRef.current.focus();
          textareaRef.current.setSelectionRange(newPos, newPos);
        }
      }, 0);
    },
    [content, mentionStartIndex]
  );

  const handleSubmit = async () => {
    const trimmed = content.trim();
    if (!trimmed || submitting) return;

    // Extract mentions from content
    const mentionRegex = /@(\w+)/g;
    const extractedMentions: Array<{ user_id: number; username: string }> = [];
    let match;
    while ((match = mentionRegex.exec(trimmed)) !== null) {
      const username = match[1];
      const existing = mentions.find((m) => m.username === username);
      extractedMentions.push({
        user_id: existing?.user_id || 0,
        username,
      });
    }

    setSubmitting(true);
    try {
      await onSubmit(trimmed, extractedMentions);
      if (!isEditMode) {
        setContent("");
        setMentions([]);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Don't handle Enter if mention popover is visible (popover handles it)
    if (mentionVisible) return;

    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div className="relative">
      {/* Reply indicator */}
      {replyTo && (
        <div className="flex items-center gap-2 px-3 py-1.5 mb-1 text-xs text-muted-foreground bg-muted/30 rounded-t-lg border border-b-0 border-border">
          <span>
            {t("tickets.detail.replyTo", { username: replyTo.username })}
          </span>
          <button
            type="button"
            onClick={onCancelReply}
            className="ml-auto hover:text-foreground transition-colors"
          >
            <X className="w-3 h-3" />
          </button>
        </div>
      )}

      <div
        className={`relative flex items-end gap-2 border border-border rounded-lg bg-card p-2 ${
          replyTo ? "rounded-t-none" : ""
        }`}
      >
        <textarea
          ref={textareaRef}
          value={content}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          placeholder={placeholder || t("tickets.detail.addComment")}
          rows={1}
          className="flex-1 resize-none bg-transparent text-sm placeholder:text-muted-foreground focus:outline-none min-h-[36px] max-h-[120px] py-1.5 px-2"
          style={{ height: "auto", overflow: "hidden" }}
          onInput={(e) => {
            const target = e.target as HTMLTextAreaElement;
            target.style.height = "auto";
            target.style.height = Math.min(target.scrollHeight, 120) + "px";
          }}
        />
        {isEditMode ? (
          <div className="flex gap-1.5 shrink-0">
            <Button
              size="sm"
              variant="outline"
              onClick={onCancel}
              className="h-8"
            >
              {t("tickets.detail.cancelReply")}
            </Button>
            <Button
              size="sm"
              onClick={handleSubmit}
              disabled={!content.trim() || submitting}
              className="h-8"
            >
              {t("tickets.detail.submit")}
            </Button>
          </div>
        ) : (
          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={!content.trim() || submitting}
            className="shrink-0 h-8 w-8 p-0"
          >
            <Send className="w-4 h-4" />
          </Button>
        )}

        {/* Mention Popover */}
        <MentionPopover
          visible={mentionVisible}
          query={mentionQuery}
          position={mentionPosition}
          onSelect={handleMentionSelect}
          onClose={() => setMentionVisible(false)}
        />
      </div>
    </div>
  );
}

export default CommentInput;
