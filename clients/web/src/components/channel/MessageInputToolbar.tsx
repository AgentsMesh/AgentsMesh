"use client";

import { Button } from "@/components/ui/button";
import { ArrowUp, AtSign, Paperclip, Loader2, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { FormatToolbarPopover } from "./FormatToolbarPopover";
import type { UseFileAttachmentResult } from "@/hooks/useFileAttachment";

interface MessageInputToolbarProps {
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
  value: string;
  onChange: (newValue: string) => void;
  onSend: () => void;
  disabled?: boolean;
  attachment: UseFileAttachmentResult;
  onMention?: () => void;
}

export function MessageInputToolbar({
  textareaRef,
  value,
  onChange,
  onSend,
  disabled,
  attachment,
  onMention,
}: MessageInputToolbarProps) {
  const t = useTranslations("channels.composer");
  const { inputRef, handleChange, pick, pending, key, name } = attachment;

  const insertAt = () => {
    if (onMention) {
      onMention();
      return;
    }
    const ta = textareaRef.current;
    if (!ta) return;
    const start = ta.selectionStart ?? value.length;
    const end = ta.selectionEnd ?? value.length;
    const newValue = value.slice(0, start) + "@" + value.slice(end);
    onChange(newValue);
    requestAnimationFrame(() => {
      ta.focus();
      const pos = start + 1;
      ta.setSelectionRange(pos, pos);
    });
  };

  const canSend = !disabled && (value.trim().length > 0 || !!key);

  return (
    <>
      {(name || pending) && (
        <AttachmentChip attachment={attachment} />
      )}
      <div className="flex items-center justify-between border-t border-border/60 px-2 py-1.5">
        <div className="flex items-center gap-0.5">
          <FormatToolbarPopover
            textareaRef={textareaRef}
            value={value}
            onChange={onChange}
          />
          <ToolbarButton label={t("mention")} onClick={insertAt} testId="toolbar-mention">
            <AtSign className="h-3.5 w-3.5" />
          </ToolbarButton>
          <ToolbarButton
            label={t("attach")}
            onClick={pick}
            disabled={pending}
            testId="toolbar-attach"
          >
            {pending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Paperclip className="h-3.5 w-3.5" />
            )}
          </ToolbarButton>
          <input
            ref={inputRef}
            type="file"
            className="hidden"
            onChange={handleChange}
            data-testid="message-attachment-input"
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-[11px] text-muted-foreground">⌘⏎ {t("send")}</span>
          <Button
            type="button"
            onClick={onSend}
            disabled={!canSend}
            size="icon"
            className="h-7 w-8 rounded-md"
            aria-label={t("send")}
          >
            <ArrowUp className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </>
  );
}

function AttachmentChip({ attachment }: { attachment: UseFileAttachmentResult }) {
  return (
    <div className="flex items-center gap-2 border-t border-border/60 px-3 py-1.5 text-xs">
      <Paperclip className="h-3 w-3 text-muted-foreground" />
      <span className="max-w-[240px] truncate text-foreground">{attachment.name}</span>
      {attachment.pending && (
        <span className="text-muted-foreground">{"⋯"}</span>
      )}
      {attachment.error && (
        <span className="text-destructive">{attachment.error}</span>
      )}
      {!attachment.pending && (
        <button
          type="button"
          onClick={attachment.clear}
          className="ml-auto inline-flex h-4 w-4 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label="remove"
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  );
}

function ToolbarButton({
  children,
  onClick,
  label,
  disabled,
  testId,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  label: string;
  disabled?: boolean;
  testId?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={label}
      aria-label={label}
      data-testid={testId}
      className="inline-flex h-6 w-6 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50"
    >
      {children}
    </button>
  );
}

export default MessageInputToolbar;
