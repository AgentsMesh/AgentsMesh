"use client";

import { useEffect, useRef, useState } from "react";
import type { KeyboardEvent } from "react";

// Single-field popover shared by /rewind (numeric turn #) and /resume
// (session id). Enter submits a non-empty value; Escape / blur closes.
export function LoopalInputPopover({
  title,
  placeholder,
  numeric,
  onSubmit,
  onClose,
}: {
  title: string;
  placeholder: string;
  numeric?: boolean;
  onSubmit: (value: string) => void;
  onClose: () => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  const [val, setVal] = useState("");
  useEffect(() => {
    ref.current?.focus();
  }, []);

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault();
      if (val.trim()) onSubmit(val.trim());
    } else if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    }
  }

  return (
    <div className="absolute bottom-full left-3 mb-2 w-56 rounded-md border border-border bg-popover p-2 shadow-md">
      <div className="mb-1 text-[10px] uppercase text-muted-foreground">{title}</div>
      <input
        ref={ref}
        value={val}
        onChange={(e) => setVal(e.target.value)}
        onKeyDown={onKeyDown}
        onBlur={onClose}
        type={numeric ? "number" : "text"}
        placeholder={placeholder}
        className="w-full rounded border border-border bg-background px-2 py-1 text-xs outline-none focus:border-ring"
      />
    </div>
  );
}
