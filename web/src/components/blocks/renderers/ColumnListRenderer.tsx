"use client";

import React from "react";

import type { Block, JSONMap } from "@/lib/api/blockstoreTypes";
import { BLOCK_TYPE_COLUMN } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";
import { useNestChildren, useRefs, useBlocks } from "@/stores/blockstore";

import { BlockRenderer, NestChildren } from "../BlockRenderer";
import { BlockChrome } from "../editor/BlockChrome";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

// ColumnListRenderer lays out its direct children (columns) horizontally.
// Each column owns its own nest tree, rendered vertically inside.
// Widths default to equal share; individual columns can override via
// data.width (0..1 fractional).
export function ColumnListRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const refIDs = useNestChildren(block.id);
  const refs = useRefs();
  const blocks = useBlocks();

  const columns: Block[] = refIDs
    .map((rid: number) => refs[rid]?.to_id)
    .filter((id: string | undefined): id is string => Boolean(id))
    .map((id: string) => blocks[id])
    .filter((b): b is Block => Boolean(b) && b.type === BLOCK_TYPE_COLUMN);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const onAddColumn = async () => {
    await dispatch.insertChild(block.id, BLOCK_TYPE_COLUMN, {}, { text: "" });
  };

  return (
    <BlockChrome
      className=""
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <div className="flex gap-3">
        {columns.map((col) => (
          <ColumnSlot key={col.id} block={col} depth={depth} totalCount={columns.length} />
        ))}
        <button
          type="button"
          onClick={onAddColumn}
          className={cn(
            "flex items-center justify-center rounded border border-dashed border-border px-2 text-xs text-muted-foreground",
            "hover:border-foreground/40 hover:text-foreground",
          )}
        >
          + column
        </button>
      </div>
    </BlockChrome>
  );
}

// ColumnSlot renders one column at its fractional width (or 1/n if unset).
// Drops to `flex-1` so the browser picks widths when none are specified.
function ColumnSlot({ block, depth, totalCount }: { block: Block; depth: number; totalCount: number }) {
  const width = (block.data as JSONMap | undefined)?.width as number | undefined;
  const style = width && width > 0 ? { flex: `${width} 1 0%` } : { flex: `1 1 ${100 / totalCount}%` };
  return (
    <div style={style} className="flex flex-col gap-1 rounded border border-border/40 p-2">
      <NestChildren parentID={block.id} depth={depth + 1} />
    </div>
  );
}

// ColumnRenderer is a standalone renderer for a bare `column` block that
// wasn't placed inside a column_list (orphan — rare). In normal flow the
// column_list's ColumnSlot handles layout; this is just a graceful fallback
// so UnknownBlock doesn't show up.
export function ColumnRenderer({ block, depth }: { block: Block; depth: number }) {
  return (
    <div className="rounded border border-border/40 p-2">
      <NestChildren parentID={block.id} depth={depth + 1} />
    </div>
  );
}

// Keep BlockRenderer import referenced (TypeScript tree-shake hint).
export { BlockRenderer };
