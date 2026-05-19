"use client";

import React, { useCallback, useEffect, useRef, useState } from "react";
import { AtSign } from "lucide-react";

import { userApi, type UserSummary } from "@/lib/api/user";
import { cn, getErrorMessage } from "@/lib/utils";

export interface UserPickerProps {
  value?: number;
  onPick: (user: UserSummary | null) => void;
  placeholder?: string;
  className?: string;
}

// UserPicker is a type-ahead popover backed by GET /users/search. Kept minimal
// — 2-char min query (server enforced), 300ms debounce, Esc to close, click
// outside to close. Callers (MentionRenderer, UserField) get both the id
// and the display name so they can persist either/both.
export function UserPicker({ value, onPick, placeholder = "Pick a user…", className }: UserPickerProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const doSearch = useCallback(async (q: string) => {
    const trimmed = q.trim();
    if (trimmed.length < 2) {
      setUsers([]);
      setError(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await userApi.search(trimmed, 10);
      setUsers(res.users ?? []);
    } catch (e) {
      setError(getErrorMessage(e, "Search failed"));
      setUsers([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!open) return;
    const timer = setTimeout(() => doSearch(query), 300);
    return () => clearTimeout(timer);
  }, [query, open, doSearch]);

  useEffect(() => {
    if (open) setTimeout(() => inputRef.current?.focus(), 0);
  }, [open]);

  return (
    <div ref={rootRef} className={cn("relative inline-block", className)}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1 rounded border border-border bg-background px-2 py-1 text-sm hover:bg-muted"
      >
        <AtSign className="h-3 w-3" />
        <span>{value ? `user#${value}` : placeholder}</span>
      </button>
      {open && (
        <div className="absolute z-50 mt-1 w-60 rounded-md border border-border bg-popover shadow">
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Type to search…"
            className="w-full border-b border-border bg-transparent px-2 py-1.5 text-sm outline-none"
          />
          <ul className="max-h-60 overflow-y-auto py-1">
            {error && <li className="px-2 py-1 text-xs text-destructive">{error}</li>}
            {loading && <li className="px-2 py-1 text-xs text-muted-foreground">Searching…</li>}
            {!loading && users.length === 0 && query.trim().length >= 2 && (
              <li className="px-2 py-1 text-xs text-muted-foreground">No matches.</li>
            )}
            {users.map((u) => (
              <li key={u.id}>
                <button
                  type="button"
                  onClick={() => {
                    onPick(u);
                    setOpen(false);
                  }}
                  className="flex w-full items-center gap-2 px-2 py-1.5 text-left text-sm hover:bg-muted"
                >
                  <AtSign className="h-3 w-3 shrink-0 text-muted-foreground" />
                  <span className="truncate">
                    {u.name || u.username}
                    {u.email && (
                      <span className="ml-1 text-xs text-muted-foreground">{u.email}</span>
                    )}
                  </span>
                </button>
              </li>
            ))}
          </ul>
          {value ? (
            <button
              type="button"
              onClick={() => {
                onPick(null);
                setOpen(false);
              }}
              className="w-full border-t border-border px-2 py-1 text-left text-xs text-destructive hover:bg-muted"
            >
              Clear
            </button>
          ) : null}
        </div>
      )}
    </div>
  );
}
