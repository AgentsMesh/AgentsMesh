"use client";

import { useState, useRef, useCallback, useMemo, KeyboardEvent } from "react";
import { Button } from "@/components/ui/button";
import { useTranslations } from "next-intl";
import { MentionDropdown } from "./MentionDropdown";
import { useMentionCandidates, type MentionItem } from "@/hooks/useMentionCandidates";

/** Pod mention info resolved at send time */
export interface MentionedPod {
  podKey: string;
  mentionText: string;
}

interface MessageInputProps {
  onSend: (content: string, mentionedPods?: MentionedPod[]) => void;
  disabled?: boolean;
  placeholder?: string;
  /** Channel ID for fetching mention candidates */
  channelId?: number | null;
}

/**
 * Parse @mentions from text and match against known pod candidates.
 * Returns deduplicated list of mentioned pods with their full pod keys.
 */
function parsePodMentions(
  text: string,
  candidates: MentionItem[]
): MentionedPod[] {
  const podCandidates = candidates.filter((c) => c.type === "pod");
  if (podCandidates.length === 0) return [];

  const mentionRegex = /@([\w.\-]+)/g;
  const result: MentionedPod[] = [];
  const seen = new Set<string>();

  let match;
  while ((match = mentionRegex.exec(text)) !== null) {
    const mentionText = match[1];
    const pod = podCandidates.find((p) => p.mentionText === mentionText);
    if (pod && !seen.has(pod.id)) {
      seen.add(pod.id);
      result.push({
        podKey: pod.id.replace("pod:", ""),
        mentionText: pod.mentionText,
      });
    }
  }

  return result;
}

/**
 * Extract the prompt text by stripping pod @mentions from the message.
 * User @mentions are preserved as they may be part of the natural language prompt.
 */
export function extractPromptFromMention(
  content: string,
  mentionedPods: MentionedPod[]
): string {
  let prompt = content;
  for (const pod of mentionedPods) {
    // Escape special regex chars in mentionText
    const escaped = pod.mentionText.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    prompt = prompt.replace(new RegExp(`@${escaped}\\s*`, "g"), "");
  }
  return prompt.trim();
}

/**
 * Build a context-aware prompt for a pod, wrapping the raw prompt with
 * channel origin and reply instruction.
 */
export function buildChannelPrompt(
  rawPrompt: string,
  channelName: string
): string {
  return [
    `Message from channel(#${channelName}): ${rawPrompt}`,
    "",
    "If you finish it, please reply to this channel.",
  ].join("\n");
}

/**
 * Extract the @ query at the cursor position.
 * Returns the query string (text after @) and its start index, or null if not in a mention.
 */
function getMentionQuery(
  text: string,
  cursorPos: number
): { query: string; startIndex: number } | null {
  // Search backwards from cursor for '@'
  const textBeforeCursor = text.slice(0, cursorPos);
  const atIndex = textBeforeCursor.lastIndexOf("@");

  if (atIndex === -1) return null;

  // '@' must be at the start of text or preceded by a whitespace/newline
  if (atIndex > 0 && !/\s/.test(textBeforeCursor[atIndex - 1])) return null;

  // Extract query: text between '@' and cursor (must not contain whitespace)
  const query = textBeforeCursor.slice(atIndex + 1);
  if (/\s/.test(query)) return null;

  return { query, startIndex: atIndex };
}

export function MessageInput({
  onSend,
  disabled,
  placeholder,
  channelId,
}: MessageInputProps) {
  const t = useTranslations();
  const defaultPlaceholder = placeholder || t("mesh.messageInput.placeholder");
  const [content, setContent] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Mention state
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

  // Filter candidates by query
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

  // Clamp active index to valid range
  const safeActiveIndex = Math.min(
    activeIndex,
    Math.max(filteredCandidates.length - 1, 0)
  );

  // Calculate dropdown position relative to container
  const updateDropdownPosition = useCallback(() => {
    const textarea = textareaRef.current;
    const container = containerRef.current;
    if (!textarea || !container) return;

    // Position dropdown above the textarea
    const containerRect = container.getBoundingClientRect();
    const textareaRect = textarea.getBoundingClientRect();

    setDropdownPosition({
      top: containerRect.bottom - textareaRect.top + 4,
      left: 0,
    });
  }, []);

  // Detect @ mention trigger on text change
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

  // Handle mention selection
  const handleMentionSelect = useCallback(
    (item: MentionItem) => {
      const before = content.slice(0, mentionStartIndex);
      const after = content.slice(
        mentionStartIndex + 1 + mentionQuery.length
      );
      const mentionText = `@${item.mentionText} `;
      const newContent = before + mentionText + after;

      setContent(newContent);
      setMentionVisible(false);

      // Restore focus and cursor position
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

    // Resolve @pod mentions from the message content
    const mentionedPods = parsePodMentions(trimmedContent, candidates);

    onSend(trimmedContent, mentionedPods.length > 0 ? mentionedPods : undefined);
    setContent("");
    setMentionVisible(false);

    // Reset textarea height
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    // Handle mention dropdown navigation
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

    // Send on Enter (without Shift)
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleInput = () => {
    // Auto-resize textarea
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = `${Math.min(
        textareaRef.current.scrollHeight,
        200
      )}px`;
    }
  };

  return (
    <div className="border-t p-4" ref={containerRef}>
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
            placeholder={defaultPlaceholder}
            disabled={disabled}
            className="w-full resize-none rounded-lg border bg-background px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 disabled:opacity-50 min-h-[44px] max-h-[200px]"
            rows={1}
          />
        </div>
        <Button
          onClick={handleSend}
          disabled={disabled || !content.trim()}
          size="icon"
          className="h-[44px] w-[44px]"
        >
          <svg
            className="w-5 h-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"
            />
          </svg>
        </Button>
      </div>
      <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
        <span>{t("mesh.messageInput.hint")}</span>
      </div>
    </div>
  );
}

export default MessageInput;
