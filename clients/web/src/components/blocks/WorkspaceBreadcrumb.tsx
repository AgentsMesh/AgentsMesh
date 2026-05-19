"use client";

import React, { useEffect, useState } from "react";
import { ChevronDown } from "lucide-react";

import { blockstoreApi } from "@/lib/api/blockstoreApi";
import type { Workspace } from "@/lib/api/blockstoreTypes";
import { cn, getErrorMessage } from "@/lib/utils";
import { useBlockstoreStore } from "@/stores/blockstore";

export interface WorkspaceBreadcrumbProps {
  current: Workspace;
  onSelect: (ws: Workspace) => void;
}

export function WorkspaceBreadcrumb({ current, onSelect }: WorkspaceBreadcrumbProps) {
  const [open, setOpen] = useState(false);
  const [list, setList] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
    });
    blockstoreApi
      .listWorkspaces()
      .then((res) => {
        if (!cancelled) setList(res.workspaces ?? []);
      })
      .catch((e) => {
        if (!cancelled) setError(getErrorMessage(e, "Load workspaces failed"));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      const el = (e.target as HTMLElement).closest("[data-ws-breadcrumb]");
      if (!el) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  const pick = (ws: Workspace) => {
    setOpen(false);
    if (ws.id === current.id) return;
    useBlockstoreStore.getState().actions.reset();
    onSelect(ws);
  };

  return (
    <div data-ws-breadcrumb className="relative inline-block">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "flex items-center gap-1 rounded border border-border bg-background px-2 py-1 text-xs hover:bg-muted",
        )}
      >
        <span className="font-medium">{current.name}</span>
        <ChevronDown className="h-3 w-3 text-muted-foreground" />
      </button>
      {open && (
        <div className="absolute left-0 z-50 mt-1 w-60 rounded-md border border-border bg-popover shadow">
          <ul className="max-h-60 overflow-y-auto py-1">
            {loading && <li className="px-2 py-1 text-xs text-muted-foreground">Loading…</li>}
            {error && <li className="px-2 py-1 text-xs text-destructive">{error}</li>}
            {!loading &&
              list.map((w) => (
                <li key={w.id}>
                  <button
                    type="button"
                    onClick={() => pick(w)}
                    className={cn(
                      "flex w-full items-center gap-2 px-2 py-1.5 text-left text-sm hover:bg-muted",
                      w.id === current.id && "font-medium text-primary",
                    )}
                  >
                    <span className="truncate">{w.name}</span>
                    <span className="ml-auto text-xs text-muted-foreground">{w.slug}</span>
                  </button>
                </li>
              ))}
            {!loading && list.length === 0 && !error && (
              <li className="px-2 py-1 text-xs text-muted-foreground">Only default workspace exists.</li>
            )}
          </ul>
        </div>
      )}
    </div>
  );
}
