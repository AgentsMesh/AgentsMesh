"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Bold, Italic, Strikethrough, Code, Link as LinkIcon, ChevronDown } from "lucide-react";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";

type WrapRule = {
  key: "bold" | "italic" | "strike" | "code" | "link";
  before: string;
  after: string;
  placeholder: string;
  cursorOffsetFromEnd?: number;
};

const RULES: WrapRule[] = [
  { key: "bold", before: "**", after: "**", placeholder: "text" },
  { key: "italic", before: "*", after: "*", placeholder: "text" },
  { key: "strike", before: "~~", after: "~~", placeholder: "text" },
  { key: "code", before: "`", after: "`", placeholder: "code" },
  { key: "link", before: "[", after: "](url)", placeholder: "text", cursorOffsetFromEnd: 4 },
];

interface FormatToolbarPopoverProps {
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
  value: string;
  onChange: (newValue: string) => void;
}

export function FormatToolbarPopover({ textareaRef, value, onChange }: FormatToolbarPopoverProps) {
  const t = useTranslations("channels.composer");
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const applyWrap = useCallback(
    (rule: WrapRule) => {
      const ta = textareaRef.current;
      if (!ta) return;
      const start = ta.selectionStart ?? value.length;
      const end = ta.selectionEnd ?? value.length;
      const selection = value.slice(start, end);
      const inner = selection || rule.placeholder;
      const insertion = `${rule.before}${inner}${rule.after}`;
      const newValue = value.slice(0, start) + insertion + value.slice(end);
      onChange(newValue);

      const cursorEnd = start + insertion.length - (rule.cursorOffsetFromEnd ?? 0);
      const cursorStart = selection
        ? cursorEnd
        : start + rule.before.length;
      requestAnimationFrame(() => {
        ta.focus();
        ta.setSelectionRange(cursorStart, cursorEnd);
      });
      setOpen(false);
    },
    [textareaRef, value, onChange],
  );

  return (
    <div ref={containerRef} className="relative inline-flex">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        title={t("formatBold")}
        aria-label={t("formatBold")}
        aria-expanded={open}
        data-testid="format-toolbar-trigger"
        className="inline-flex h-6 items-center gap-0.5 rounded px-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <Bold className="h-3.5 w-3.5" />
        <ChevronDown className="h-3 w-3 opacity-60" />
      </button>

      {open && (
        <div
          role="menu"
          data-testid="format-toolbar-menu"
          className={cn(
            "absolute z-50 bottom-full mb-1 left-0 flex items-center gap-0.5",
            "rounded-md border border-border bg-popover p-1 shadow-md",
            "animate-in fade-in-0 zoom-in-95",
          )}
        >
          <FormatButton testId="format-bold" icon={<Bold className="h-3.5 w-3.5" />} label={t("formatBold")} onClick={() => applyWrap(RULES[0])} />
          <FormatButton testId="format-italic" icon={<Italic className="h-3.5 w-3.5" />} label={t("formatItalic")} onClick={() => applyWrap(RULES[1])} />
          <FormatButton testId="format-strike" icon={<Strikethrough className="h-3.5 w-3.5" />} label={t("formatStrike")} onClick={() => applyWrap(RULES[2])} />
          <FormatButton testId="format-code" icon={<Code className="h-3.5 w-3.5" />} label={t("formatCode")} onClick={() => applyWrap(RULES[3])} />
          <FormatButton testId="format-link" icon={<LinkIcon className="h-3.5 w-3.5" />} label={t("formatLink")} onClick={() => applyWrap(RULES[4])} />
        </div>
      )}
    </div>
  );
}

function FormatButton({
  icon,
  label,
  onClick,
  testId,
}: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  testId?: string;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      onClick={onClick}
      title={label}
      aria-label={label}
      data-testid={testId}
      className="inline-flex h-6 w-6 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
    >
      {icon}
    </button>
  );
}

export default FormatToolbarPopover;
