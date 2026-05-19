"use client";

import React from "react";
import { Copy, Trash, X } from "lucide-react";

import { useBlockstoreStore } from "@/stores/blockstore";

import { useBlockstoreDispatch } from "./useBlockstoreDispatch";

/**
 * SelectionActionBar is a floating toolbar that appears at the bottom of the
 * page whenever at least one block is selected. It batches delete / duplicate
 * over the full selection so multi-edits cost one ApplyOps per block.
 */
export function SelectionActionBar({ workspaceID }: { workspaceID: string }) {
  const selection = useBlockstoreStore((s) => s.selectedBlockIDs);
  const clear = useBlockstoreStore((s) => s.actions.clearSelection);
  const dispatch = useBlockstoreDispatch(workspaceID);

  if (selection.length === 0) return null;

  const handleDelete = async () => {
    const ids = [...selection];
    clear();
    for (const id of ids) {
      await dispatch.detachChild(id);
      await dispatch.removeBlock(id);
    }
  };

  const handleDuplicate = async () => {
    const ids = [...selection];
    for (const id of ids) await dispatch.duplicate(id);
    clear();
  };

  return (
    <div className="pointer-events-auto fixed bottom-6 left-1/2 z-50 flex -translate-x-1/2 items-center gap-2 rounded-full border border-border bg-popover px-3 py-1.5 shadow-lg">
      <span className="text-xs text-muted-foreground">
        {selection.length} selected
      </span>
      <button
        type="button"
        onClick={handleDuplicate}
        className="flex items-center gap-1 rounded px-2 py-1 text-xs hover:bg-accent"
      >
        <Copy className="h-3 w-3" />
        Duplicate
      </button>
      <button
        type="button"
        onClick={handleDelete}
        className="flex items-center gap-1 rounded px-2 py-1 text-xs text-destructive hover:bg-destructive/10"
      >
        <Trash className="h-3 w-3" />
        Delete
      </button>
      <button
        type="button"
        onClick={clear}
        aria-label="Clear selection"
        className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
      >
        <X className="h-3 w-3" />
      </button>
    </div>
  );
}
