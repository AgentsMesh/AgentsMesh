"use client";

import { useState, useRef, useCallback, useMemo, KeyboardEvent } from "react";
import { useTranslations } from "next-intl";
import { MentionDropdown } from "./MentionDropdown";
import { MessageInputToolbar } from "./MessageInputToolbar";
import { useFileAttachment } from "@/hooks/useFileAttachment";
import { useMentionCandidates, type MentionItem } from "@/hooks/useMentionCandidates";
import { getMentionQuery } from "./mention";
import { buildMessageContent } from "./message-content-builder";
import type { MessageContent } from "@/lib/api/channel-message-types";

interface MentionRef {
  entityType: string;
  entityKey: string;
}

interface MessageInputProps {
  onSend: (content: MessageContent) => void;
  disabled?: boolean;
  placeholder?: string;
  channelId?: number | null;
  channelName?: string;
}

export function MessageInput({
  onSend,
  disabled,
  placeholder,
  channelId,
  channelName,
}: MessageInputProps) {
  const t = useTranslations();
  const defaultPlaceholder = placeholder
    || (channelName
      ? t("channels.composer.placeholder", { channel: channelName })
      : t("mesh.messageInput.placeholder"));
  const [content, setContent] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const attachment = useFileAttachment();

  const selectedMentionsRef = useRef<Map<string, MentionRef>>(new Map());

  const [mentionVisible, setMentionVisible] = useState(false);
  const [mentionQuery, setMentionQuery] = useState("");
  const [mentionStartIndex, setMentionStartIndex] = useState(-1);
  const [activeIndex, setActiveIndex] = useState(0);
  const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number } | null>(null);

  const { candidates } = useMentionCandidates({
    channelId: channelId ?? null,
    enabled: !!channelId,
  });

  const filteredCandidates = useMemo(() => {
    return candidates.filter((item) => {
      if (!mentionQuery) return true;
      const q = mentionQuery.toLowerCase();
      return (
        item.displayName.toLowerCase().includes(q) ||
        item.mentionText.toLowerCase().includes(q) ||
        (item.description?.toLowerCase().includes(q) ?? false)
      );
    });
  }, [candidates, mentionQuery]);

  const safeActiveIndex = Math.min(activeIndex, Math.max(filteredCandidates.length - 1, 0));

  const updateDropdownPosition = useCallback(() => {
    const textarea = textareaRef.current;
    const container = containerRef.current;
    if (!textarea || !container) return;
    const containerRect = container.getBoundingClientRect();
    const textareaRect = textarea.getBoundingClientRect();
    setDropdownPosition({ top: containerRect.bottom - textareaRect.top + 4, left: 0 });
  }, []);

  const handleChange = useCallback(
    (value: string) => {
      setContent(value);
      const textarea = textareaRef.current;
      if (!textarea) return;
      const cursorPos = textarea.selectionStart;
      const result = getMentionQuery(value, cursorPos);
      if (result && candidates.length > 0) {
        setMentionQuery(result.query);
        setMentionStartIndex(result.startIndex);
        setMentionVisible(true);
        setActiveIndex(0);
        updateDropdownPosition();
      } else {
        setMentionVisible(false);
      }
    },
    [candidates.length, updateDropdownPosition],
  );

  /**
   * Toolbar @-button path. We can't rely on `handleChange` alone because the
   * textarea DOM `selectionStart` hasn't updated yet when handleChange runs
   * from within the click handler; that makes `getMentionQuery` miss the "@"
   * we just inserted. So we set mention state explicitly here.
   */
  const triggerMention = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    const start = textarea.selectionStart ?? content.length;
    const end = textarea.selectionEnd ?? content.length;
    const newContent = content.slice(0, start) + "@" + content.slice(end);
    setContent(newContent);
    setMentionStartIndex(start);
    setMentionQuery("");
    setActiveIndex(0);
    setMentionVisible(candidates.length > 0);
    updateDropdownPosition();
    requestAnimationFrame(() => {
      const ta = textareaRef.current;
      if (!ta) return;
      ta.focus();
      const pos = start + 1;
      ta.setSelectionRange(pos, pos);
    });
  }, [content, candidates.length, updateDropdownPosition]);

  const handleMentionSelect = useCallback(
    (item: MentionItem) => {
      const before = content.slice(0, mentionStartIndex);
      const after = content.slice(mentionStartIndex + 1 + mentionQuery.length);
      const mentionText = `@${item.mentionText} `;
      const newContent = before + mentionText + after;

      const colonIdx = item.id.indexOf(":");
      if (colonIdx >= 0) {
        const entityType = item.id.slice(0, colonIdx);
        const entityKey = item.id.slice(colonIdx + 1);
        if (entityType === "user" || entityType === "pod") {
          selectedMentionsRef.current.set(item.mentionText, { entityType, entityKey });
        }
      }

      setContent(newContent);
      setMentionVisible(false);

      requestAnimationFrame(() => {
        const textarea = textareaRef.current;
        if (textarea) {
          textarea.focus();
          const newCursorPos = before.length + mentionText.length;
          textarea.setSelectionRange(newCursorPos, newCursorPos);
        }
      });
    },
    [content, mentionStartIndex, mentionQuery],
  );

  const handleSend = () => {
    const trimmed = content.trim();
    if (!trimmed && !attachment.key) return;
    if (disabled) return;

    const messageContent = buildMessageContent(trimmed, selectedMentionsRef.current);
    if (attachment.key) messageContent.attachment_key = attachment.key;

    onSend(messageContent);
    setContent("");
    setMentionVisible(false);
    selectedMentionsRef.current.clear();
    attachment.clear();

    if (textareaRef.current) textareaRef.current.style.height = "auto";
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.nativeEvent.isComposing) return;

    if (mentionVisible && filteredCandidates.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((prev) => (prev < filteredCandidates.length - 1 ? prev + 1 : 0));
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((prev) => (prev > 0 ? prev - 1 : filteredCandidates.length - 1));
        return;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        handleMentionSelect(filteredCandidates[safeActiveIndex]);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        setMentionVisible(false);
        return;
      }
    }

    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleInput = () => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 200)}px`;
    }
  };

  return (
    <div className="border-t border-border px-4 py-3" ref={containerRef}>
      <div className="relative rounded-lg border border-border bg-background">
        <MentionDropdown
          items={filteredCandidates}
          activeIndex={safeActiveIndex}
          onSelect={handleMentionSelect}
          position={dropdownPosition}
          visible={mentionVisible}
        />

        <textarea
          ref={textareaRef}
          value={content}
          onChange={(e) => handleChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onInput={handleInput}
          placeholder={defaultPlaceholder}
          aria-label={defaultPlaceholder}
          disabled={disabled}
          className="block w-full resize-none border-0 bg-transparent px-3 pt-2.5 pb-1 text-[13px] focus:outline-none disabled:opacity-50 min-h-[42px] max-h-[200px]"
          rows={1}
          data-testid="message-input-textarea"
        />

        <MessageInputToolbar
          textareaRef={textareaRef}
          value={content}
          onChange={handleChange}
          onSend={handleSend}
          disabled={disabled}
          attachment={attachment}
          onMention={triggerMention}
        />
      </div>
    </div>
  );
}

export default MessageInput;
