"use client";

import React, { useMemo } from "react";

import { useBlocks } from "@/stores/blockstore";

import { BlockRenderer } from "./BlockRenderer";

export interface ListViewProps {
  workspaceID: string;
  blockType: string;
}

export function ListView({ workspaceID, blockType }: ListViewProps) {
  const blocks = useBlocks();
  const filtered = useMemo(
    () =>
      Object.values(blocks)
        .filter((b) => b.workspace_id === workspaceID && b.type === blockType && !b.deleted_at)
        .sort((a, b) => a.created_at.localeCompare(b.created_at)),
    [blocks, workspaceID, blockType],
  );

  if (filtered.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-muted-foreground/40 p-4 text-sm text-muted-foreground">
        No {blockType} blocks yet.
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {filtered.map((b) => (
        <BlockRenderer key={b.id} blockID={b.id} depth={0} />
      ))}
    </div>
  );
}
