"use client";

import { useEffect, useRef, useState } from "react";

interface Props {
  text: string;
  label?: string;
}

export function CopyButton({ text, label = "Copy" }: Props) {
  const [copied, setCopied] = useState(false);
  const resetTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => () => {
    if (resetTimer.current) clearTimeout(resetTimer.current);
  }, []);

  const handleClick = () => {
    navigator.clipboard.writeText(text).then(
      () => {
        setCopied(true);
        if (resetTimer.current) clearTimeout(resetTimer.current);
        resetTimer.current = setTimeout(() => setCopied(false), 1800);
      },
      () => {
        // fail silently — user can still select text
      },
    );
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      className="absolute top-3 right-3 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[10px] font-headline uppercase tracking-[0.18em] border border-white/10 bg-white/5 hover:bg-white/10 hover:border-[var(--azure-cyan)]/40 text-[var(--azure-text-muted)] hover:text-[var(--azure-cyan)] transition-colors"
      aria-label={copied ? "Copied" : label}
    >
      {copied ? "✓ Copied" : label}
    </button>
  );
}
