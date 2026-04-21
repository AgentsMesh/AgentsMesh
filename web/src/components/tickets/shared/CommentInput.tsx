"use client";

import { useState, useRef, useCallback } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Send, X, Loader2 } from "lucide-react";
import { useCurrentUser, useAuthStore } from "@/stores/auth";
import { MentionPopover } from "./MentionPopover";

interface CommentInputProps {
  onSubmit: (
    content: string,
    mentions: Array<{ user_id: number; username: string }>
  ) => Promise<void>;
  replyTo?: { id: number; username: string };
  onCancelReply?: () => void;
  placeholder?: string;
  initialContent?: string;
  onCancel?: () => void;
}

function UserAvatar({ src, name }: { src?: string; name: string }) {
  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={src}
        alt=""
        className="w-8 h-8 rounded-full shrink-0 ring-1 ring-border/30"
      />
    );
  }
  return (
    <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-xs font-semibold text-primary shrink-0 ring-1 ring-primary/15">
      {(name || "?")[0].toUpperCase()}
    </div>
  );
}

export function CommentInput({
  onSubmit,
  replyTo,
  onCancelReply,
  placeholder,
  initialContent,
  onCancel,
}: CommentInputProps) {
  const t = useTranslations();
  const user = useCurrentUser();
  const isEditMode = initialContent !== undefined;
  const [content, setContent] = useState(initialContent || "");
  const [submitting, setSubmitting] = useState(false);
  const [mentions, setMentions] = useState<
    Array<{ user_id: number; username: string }>
  >([]);

  const [mentionVisible, setMentionVisible] = useState(false);
  const [mentionQuery, setMentionQuery] = useState("");
  const [mentionPosition, setMentionPosition] = useState({ top: 0, left: 0 });
  const [mentionStartIndex, setMentionStartIndex] = useState(-1);

  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    const cursorPos = e.target.selectionStart || 0;
    setContent(value);

    const textBeforeCursor = value.slice(0, cursorPos);
    const atIndex = textBeforeCursor.lastIndexOf("@");

    if (atIndex >= 0) {
      const charBeforeAt = atIndex > 0 ? textBeforeCursor[atIndex - 1] : " ";
      const textAfterAt = textBeforeCursor.slice(atIndex + 1);

      if (
        (charBeforeAt === " " || charBeforeAt === "\n" || atIndex === 0) &&
        !textAfterAt.includes(" ")
      ) {
        setMentionQuery(textAfterAt);
        setMentionStartIndex(atIndex);
        setMentionVisible(true);

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

      setMentions((prev) => {
        if (prev.some((m) => m.username === username)) return prev;
        return [...prev, { user_id: 0, username }];
      });

      setTimeout(() => {
        if (textareaRef.current) {
          const newPos = mentionStartIndex + username.length + 2;
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
    if (mentionVisible) return;

    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  if (isEditMode) {
    return (
      <div className="relative">
        <div className="relative flex items-end gap-2 border border-border rounded-lg bg-card p-2">
          <textarea
            ref={textareaRef}
            value={content}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
            placeholder={placeholder || t("tickets.detail.addComment")}
            rows={2}
            className="flex-1 resize-none bg-transparent text-sm placeholder:text-muted-foreground/50 focus:outline-none min-h-[60px] max-h-[120px] py-1.5 px-2"
            style={{ overflow: "hidden" }}
            onInput={(e) => {
              const target = e.target as HTMLTextAreaElement;
              target.style.height = "auto";
              target.style.height = Math.min(target.scrollHeight, 120) + "px";
            }}
          />
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
              {submitting ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                t("tickets.detail.submit")
              )}
            </Button>
          </div>

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

  return (
    <div className="relative">
      {/* Reply indicator */}
      {replyTo && (
        <div className="flex items-center gap-2 px-3 py-1.5 mb-1 text-xs text-muted-foreground bg-muted/30 rounded-t-xl border border-b-0 border-border/50">
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
        className={`flex items-start gap-3 rounded-xl border border-border/50 bg-card shadow-sm p-3 ${
          replyTo ? "rounded-t-none" : ""
        }`}
      >
        <UserAvatar
          src={user?.avatar_url}
          name={user?.name || user?.username || "?"}
        />

        <div className="flex-1 min-w-0 relative">
          <textarea
            ref={textareaRef}
            value={content}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
            placeholder={placeholder || t("tickets.detail.addComment")}
            rows={1}
            className="w-full resize-none bg-transparent text-sm placeholder:text-muted-foreground/50 focus:outline-none min-h-[36px] max-h-[120px] py-1.5"
            style={{ height: "auto", overflow: "hidden" }}
            onInput={(e) => {
              const target = e.target as HTMLTextAreaElement;
              target.style.height = "auto";
              target.style.height = Math.min(target.scrollHeight, 120) + "px";
            }}
          />
          <div className="flex items-center justify-between mt-1">
            <span className="text-[10px] text-muted-foreground/40">
              Enter to send &middot; Shift+Enter for new line
            </span>
            <Button
              size="sm"
              onClick={handleSubmit}
              disabled={!content.trim() || submitting}
              className="shrink-0 h-7 px-3 text-xs gap-1.5"
            >
              {submitting ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <>
                  <Send className="w-3.5 h-3.5" />
                  {t("tickets.detail.submit")}
                </>
              )}
            </Button>
          </div>

          <MentionPopover
            visible={mentionVisible}
            query={mentionQuery}
            position={mentionPosition}
            onSelect={handleMentionSelect}
            onClose={() => setMentionVisible(false)}
          />
        </div>
      </div>
    </div>
  );
}

export default CommentInput;
