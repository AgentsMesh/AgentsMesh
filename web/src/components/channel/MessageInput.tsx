"use client";

import { useState, useRef, useCallback, useMemo, KeyboardEvent } from "react";
import { Button } from "@/components/ui/button";
import { SendHorizontal } from "lucide-react";
import { useTranslations } from "next-intl";
import { MentionDropdown } from "./MentionDropdown";
import { useMentionCandidates, type MentionItem } from "@/hooks/useMentionCandidates";
import { getMentionQuery } from "./mention";
import type { MentionPayload } from "@/lib/api/channel";

interface MessageInputProps {
  onSend: (content: string, mentions?: MentionPayload[]) => void;
  disabled?: boolean;
  placeholder?: string;
  /** Channel ID for fetching mention candidates */
  channelId?: number | null;
}

/**
 * Parse a MentionItem.id ("user:123" | "pod:my-key") into a structured MentionPayload.
 */
function toMentionPayload(item: MentionItem): MentionPayload | null {
  const colonIdx = item.id.indexOf(":");
  if (colonIdx < 0) return null;
  const type = item.id.slice(0, colonIdx);
  const id = item.id.slice(colonIdx + 1);
  if (type !== "user" && type !== "pod") return null;
  return { type, id };
}

export function MessageInput({
  onSend,
  disabled,
  placeholder,
  channelId,
}: MessageInputProps) {
  const t = useTranslations();
  const defaultPlaceholder =
    placeholder || t("mesh.messageInput.placeholder");
  const [content, setContent] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Structured mention accumulator — collects payloads as user selects from dropdown
  const selectedMentionsRef = useRef<Map<string, MentionPayload>>(new Map());

  // Mention state (for @ autocomplete UI only; actual routing is server-side)
  const [mentionVisible, setMentionVisible] = useState(false);
  const [mentionQuery, setMentionQuery] = useState("");
  const [mentionStartIndex, setMentionStartIndex] = useState(-1);
  const [activeIndex, setActiveIndex] = useState(0);
  const [dropdownPosition, setDropdownPosition] = useState<{
    top: number;
    left: number;
  } | null>(null);

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

  const safeActiveIndex = Math.min(
    activeIndex,
    Math.max(filteredCandidates.length - 1, 0)
  );

  const updateDropdownPosition = useCallback(() => {
    const textarea = textareaRef.current;
    const container = containerRef.current;
    if (!textarea || !container) return;

    const containerRect = container.getBoundingClientRect();
    const textareaRect = textarea.getBoundingClientRect();

    setDropdownPosition({
      top: containerRect.bottom - textareaRect.top + 4,
      left: 0,
    });
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
    [candidates.length, updateDropdownPosition]
  );

  const handleMentionSelect = useCallback(
    (item: MentionItem) => {
      const before = content.slice(0, mentionStartIndex);
      const after = content.slice(mentionStartIndex + 1 + mentionQuery.length);
      const mentionText = `@${item.mentionText} `;
      const newContent = before + mentionText + after;

      // Collect structured mention data
      const payload = toMentionPayload(item);
      if (payload) {
        selectedMentionsRef.current.set(item.id, payload);
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
    [content, mentionStartIndex, mentionQuery]
  );

  const handleSend = () => {
    const trimmedContent = content.trim();
    if (!trimmedContent || disabled) return;

    const mentions = selectedMentionsRef.current.size > 0
      ? Array.from(selectedMentionsRef.current.values())
      : undefined;

    onSend(trimmedContent, mentions);
    setContent("");
    setMentionVisible(false);
    selectedMentionsRef.current.clear();

    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (mentionVisible && filteredCandidates.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((prev) =>
          prev < filteredCandidates.length - 1 ? prev + 1 : 0
        );
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((prev) =>
          prev > 0 ? prev - 1 : filteredCandidates.length - 1
        );
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
      textareaRef.current.style.height = `${Math.min(
        textareaRef.current.scrollHeight,
        200
      )}px`;
    }
  };

  return (
    <div className="border-t px-4 py-3" ref={containerRef}>
      <div className="flex items-end gap-2">
        <div className="flex-1 relative">
          {/* Mention dropdown */}
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
            placeholder={`${defaultPlaceholder}  (Enter ↵)`}
            disabled={disabled}
            className="block w-full resize-none rounded-xl border bg-muted/40 px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:bg-background disabled:opacity-50 min-h-[42px] max-h-[200px] transition-colors"
            rows={1}
          />
        </div>
        <Button
          onClick={handleSend}
          disabled={disabled || !content.trim()}
          size="icon"
          className="h-[42px] w-[42px] rounded-xl shrink-0"
        >
          <SendHorizontal className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}

export default MessageInput;
