"use client";

import { useCallback, useMemo, useRef, useState, type KeyboardEvent, type RefObject } from "react";
import { useMentionCandidates, type MentionItem } from "@/hooks/useMentionCandidates";
import { getMentionQuery } from "./mention";
import type { MentionRefInput } from "@/lib/viewModels/channelMessage";

interface Options {
  channelId: number | null | undefined;
  content: string;
  setContent: (value: string) => void;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  containerRef: RefObject<HTMLDivElement | null>;
}

/**
 * @-mention autocomplete state machine for the composer: query detection,
 * candidate filtering, dropdown position, selection, and the keyboard nav that
 * the textarea's keydown delegates to. Extracted from MessageInput for SRP.
 */
export function useMentionAutocomplete({ channelId, content, setContent, textareaRef, containerRef }: Options) {
  const selectedMentionsRef = useRef<Map<string, MentionRefInput>>(new Map());
  const [visible, setVisible] = useState(false);
  const [query, setQuery] = useState("");
  const [startIndex, setStartIndex] = useState(-1);
  const [activeIndex, setActiveIndex] = useState(0);
  const [position, setPosition] = useState<{ top: number; left: number } | null>(null);

  const { candidates } = useMentionCandidates({ channelId: channelId ?? null, enabled: !!channelId });

  const filtered = useMemo(() => {
    return candidates.filter((item) => {
      if (!query) return true;
      const q = query.toLowerCase();
      return (
        item.displayName.toLowerCase().includes(q) ||
        item.mentionText.toLowerCase().includes(q) ||
        (item.description?.toLowerCase().includes(q) ?? false)
      );
    });
  }, [candidates, query]);

  const safeActiveIndex = Math.min(activeIndex, Math.max(filtered.length - 1, 0));

  const updatePosition = useCallback(() => {
    const textarea = textareaRef.current;
    const container = containerRef.current;
    if (!textarea || !container) return;
    const containerRect = container.getBoundingClientRect();
    const textareaRect = textarea.getBoundingClientRect();
    setPosition({ top: containerRect.bottom - textareaRect.top + 4, left: 0 });
  }, [textareaRef, containerRef]);

  const handleChange = useCallback(
    (value: string) => {
      setContent(value);
      const textarea = textareaRef.current;
      if (!textarea) return;
      const result = getMentionQuery(value, textarea.selectionStart);
      if (result && candidates.length > 0) {
        setQuery(result.query);
        setStartIndex(result.startIndex);
        setVisible(true);
        setActiveIndex(0);
        updatePosition();
      } else {
        setVisible(false);
      }
    },
    [candidates.length, updatePosition, setContent, textareaRef],
  );

  // Toolbar @-button path: the textarea selectionStart hasn't updated yet when
  // this fires from the click handler, so set mention state explicitly.
  const triggerMention = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    const start = textarea.selectionStart ?? content.length;
    const end = textarea.selectionEnd ?? content.length;
    setContent(content.slice(0, start) + "@" + content.slice(end));
    setStartIndex(start);
    setQuery("");
    setActiveIndex(0);
    setVisible(candidates.length > 0);
    updatePosition();
    requestAnimationFrame(() => {
      const ta = textareaRef.current;
      if (!ta) return;
      ta.focus();
      ta.setSelectionRange(start + 1, start + 1);
    });
  }, [content, candidates.length, updatePosition, setContent, textareaRef]);

  const handleSelect = useCallback(
    (item: MentionItem) => {
      const before = content.slice(0, startIndex);
      const after = content.slice(startIndex + 1 + query.length);
      const mentionText = `@${item.mentionText} `;

      const colonIdx = item.id.indexOf(":");
      if (colonIdx >= 0) {
        const entityType = item.id.slice(0, colonIdx);
        const entityKey = item.id.slice(colonIdx + 1);
        if (entityType === "user" || entityType === "pod") {
          selectedMentionsRef.current.set(item.mentionText, { entity_type: entityType, entity_key: entityKey });
        }
      }

      setContent(before + mentionText + after);
      setVisible(false);
      requestAnimationFrame(() => {
        const textarea = textareaRef.current;
        if (!textarea) return;
        textarea.focus();
        const pos = before.length + mentionText.length;
        textarea.setSelectionRange(pos, pos);
      });
    },
    [content, startIndex, query, setContent, textareaRef],
  );

  // Returns true when it consumed the event (caller should stop).
  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>): boolean => {
      if (!visible || filtered.length === 0) return false;
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((prev) => (prev < filtered.length - 1 ? prev + 1 : 0));
        return true;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((prev) => (prev > 0 ? prev - 1 : filtered.length - 1));
        return true;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        handleSelect(filtered[safeActiveIndex]);
        return true;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        setVisible(false);
        return true;
      }
      return false;
    },
    [visible, filtered, safeActiveIndex, handleSelect],
  );

  const getMentions = useCallback((): Record<string, MentionRefInput> => {
    const mentions: Record<string, MentionRefInput> = {};
    selectedMentionsRef.current.forEach((ref, display) => { mentions[display] = ref; });
    return mentions;
  }, []);

  const reset = useCallback(() => {
    setVisible(false);
    selectedMentionsRef.current.clear();
  }, []);

  return {
    filtered, safeActiveIndex, visible, position,
    handleChange, triggerMention, handleSelect, handleKeyDown, getMentions, reset,
  };
}
