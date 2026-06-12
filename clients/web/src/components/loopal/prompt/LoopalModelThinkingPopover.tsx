"use client";

import { useEffect, useRef, useState } from "react";
import type { KeyboardEvent } from "react";
import { useTranslations } from "next-intl";
import { THINKING_OPTIONS, thinkingKey } from "./loopalThinking";

// /model + /thinking selector. Mirrors loopal's combined picker: thinking
// effort is the selectable axis (↑↓ + Enter); model is shown read-only since
// the backend exposes no switchable model list yet.
export function LoopalModelThinkingPopover({
  currentModel,
  currentThinking,
  onPick,
  onClose,
}: {
  currentModel: string | null;
  currentThinking: string | null;
  onPick: (config: string) => void;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  // Highlight the current thinking level so ↑↓+Enter confirms it rather than
  // silently switching to a hardcoded default; fall back to the first option.
  // Match via thinkingKey (parse-normalize both sides) — a raw config-string
  // equality would break on cross-language JSON key-order/whitespace drift.
  const [active, setActive] = useState(() => {
    const key = thinkingKey(currentThinking);
    return Math.max(0, THINKING_OPTIONS.findIndex((o) => o.key === key));
  });
  const t = useTranslations("loopal");
  useEffect(() => {
    ref.current?.focus();
  }, []);

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    const n = THINKING_OPTIONS.length;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((a) => (a + 1) % n);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) => (a - 1 + n) % n);
    } else if (e.key === "Enter") {
      e.preventDefault();
      onPick(THINKING_OPTIONS[active].config);
    } else if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    }
  }

  return (
    <div
      ref={ref}
      tabIndex={-1}
      onKeyDown={onKeyDown}
      onBlur={onClose}
      className="absolute bottom-full left-3 mb-2 w-56 rounded-md border border-border bg-popover p-1 shadow-md outline-none"
    >
      <div className="px-2 py-1 text-[10px] uppercase text-muted-foreground">
        {t("prompt.model")}{" "}
        {currentModel && <span className="font-mono normal-case text-foreground">· {currentModel}</span>}
      </div>
      <div className="px-2 pb-0.5 pt-2 text-[10px] uppercase text-muted-foreground">{t("prompt.thinking")}</div>
      {THINKING_OPTIONS.map((o, i) => (
        <button
          key={o.key}
          type="button"
          onMouseDown={(e) => {
            e.preventDefault();
            onPick(o.config);
          }}
          className={`flex w-full items-center rounded px-2 py-1 text-left text-xs ${
            i === active ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted/50"
          }`}
        >
          {t("thinking." + o.key)}
        </button>
      ))}
    </div>
  );
}
