"use client";

import React, { useEffect, useRef } from "react";

import { cn } from "@/lib/utils";

export interface SlashOption {
  id: string;
  label: string;
  hint?: string;
  onSelect: () => void | Promise<void>;
}

export interface SlashMenuProps {
  open: boolean;
  options: SlashOption[];
  onClose: () => void;
  anchorClassName?: string;
}

export function SlashMenu({ open, options, onClose, anchorClassName }: SlashMenuProps) {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (!ref.current) return;
      if (!ref.current.contains(e.target as Node)) onClose();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div
      ref={ref}
      className={cn(
        "z-50 min-w-[200px] rounded-md border border-border bg-popover p-1 shadow-md",
        anchorClassName,
      )}
    >
      {options.map((o) => (
        <button
          type="button"
          key={o.id}
          onClick={async () => {
            onClose();
            await o.onSelect();
          }}
          className="flex w-full items-center justify-between gap-2 rounded px-2 py-1.5 text-left text-sm hover:bg-accent"
        >
          <span>{o.label}</span>
          {o.hint && <span className="text-xs text-muted-foreground">{o.hint}</span>}
        </button>
      ))}
    </div>
  );
}
