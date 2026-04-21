"use client";

import React from "react";
import { Link as LinkIcon } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { useBlock } from "@/stores/blockstore";

import { BlockRenderer } from "../BlockRenderer";
import { BlockChrome } from "../editor/BlockChrome";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

// SyncedBlockRenderer mirrors another block's content inline. data.source_id
// points at the target block; this renderer resolves it from the store and
// delegates to BlockRenderer. Self-references short-circuit. Because both
// copies read the same store entry, edits to the source propagate to every
// mirror on the next WS push — no extra sync protocol needed.
export function SyncedBlockRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const sourceID = (block.data?.source_id as string | undefined) ?? "";
  const source = useBlock(sourceID || null);
  const selfRef = sourceID === block.id;

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const onPick = () => {
    const next = window.prompt("Source block id:", sourceID);
    if (next && next !== block.id) {
      dispatch.updateBlockData(block.id, { source_id: next });
    }
  };

  return (
    <BlockChrome
      className=""
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <div className="rounded-md border-l-4 border-primary/40 bg-primary/5 p-2">
        <div className="mb-1 flex items-center gap-1 text-[10px] uppercase tracking-wide text-muted-foreground">
          <LinkIcon className="h-3 w-3" />
          <span>synced</span>
          {selfRef && <span className="text-destructive">(self-reference blocked)</span>}
        </div>
        {selfRef ? null : source ? (
          <BlockRenderer blockID={sourceID} depth={depth + 1} />
        ) : (
          <button
            type="button"
            onClick={onPick}
            className="w-full rounded border border-dashed border-border bg-muted/30 p-2 text-xs text-muted-foreground hover:bg-muted/50"
          >
            {sourceID ? `Source not loaded: ${sourceID.slice(0, 8)}…` : "Link to a source block"}
          </button>
        )}
      </div>
    </BlockChrome>
  );
}
