"use client";

import React, { useEffect, useRef } from "react";

import { cn } from "@/lib/utils";

export interface EditableTextProps {
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  className?: string;
  debounceMs?: number;
  onEnter?: () => void;
  onBackspaceEmpty?: () => void;
  onSlashOnEmpty?: () => void;
  autoFocus?: boolean;
}

export function EditableText({
  value,
  onChange,
  placeholder,
  className,
  debounceMs = 300,
  onEnter,
  onBackspaceEmpty,
  onSlashOnEmpty,
  autoFocus,
}: EditableTextProps) {
  const ref = useRef<HTMLDivElement | null>(null);
  const lastEmitted = useRef(value);
  const pendingTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;
    if (node.innerText !== value) {
      node.innerText = value;
      lastEmitted.current = value;
    }
  }, [value]);

  useEffect(() => {
    if (!autoFocus) return;
    const node = ref.current;
    if (!node) return;
    node.focus();
    placeCaretAtEnd(node);
  }, [autoFocus]);

  const flush = () => {
    const node = ref.current;
    if (!node) return;
    const next = node.innerText;
    if (next !== lastEmitted.current) {
      lastEmitted.current = next;
      onChange(next);
    }
  };

  const scheduleFlush = () => {
    if (pendingTimer.current) clearTimeout(pendingTimer.current);
    pendingTimer.current = setTimeout(flush, debounceMs);
  };

  useEffect(() => {
    return () => {
      if (pendingTimer.current) clearTimeout(pendingTimer.current);
    };
  }, []);

  const onKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === "Enter" && !e.shiftKey && onEnter) {
      e.preventDefault();
      flush();
      onEnter();
      return;
    }
    if (e.key === "Backspace" && onBackspaceEmpty) {
      const node = ref.current;
      if (node && node.innerText.length === 0) {
        e.preventDefault();
        onBackspaceEmpty();
      }
    }
    if (e.key === "/" && onSlashOnEmpty) {
      const node = ref.current;
      if (node && node.innerText.length === 0) {
        e.preventDefault();
        onSlashOnEmpty();
      }
    }
  };

  const isEmpty = value.length === 0;

  return (
    <div
      ref={ref}
      contentEditable
      suppressContentEditableWarning
      onInput={scheduleFlush}
      onBlur={flush}
      onKeyDown={onKeyDown}
      data-placeholder={placeholder}
      className={cn(
        "min-h-[1.25em] whitespace-pre-wrap break-words",
        isEmpty && "before:text-muted-foreground/60 before:content-[attr(data-placeholder)]",
        className,
      )}
    />
  );
}

function placeCaretAtEnd(node: HTMLElement) {
  const range = document.createRange();
  range.selectNodeContents(node);
  range.collapse(false);
  const sel = window.getSelection();
  if (!sel) return;
  sel.removeAllRanges();
  sel.addRange(range);
}
