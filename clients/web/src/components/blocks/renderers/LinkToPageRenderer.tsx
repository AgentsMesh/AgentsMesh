"use client";

import React from "react";
import { ArrowUpRight } from "lucide-react";

import type { Block } from "@/lib/viewModels/blockstore";
import { useJumpToBlock } from "@/lib/blockstore/useJumpToBlock";
import { useBlock } from "@/stores/blockstore";

import { BlockChrome } from "../editor/BlockChrome";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function LinkToPageRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const jumpToBlock = useJumpToBlock();
  const targetID = (block.data?.target_id as string | undefined) ?? "";
  const target = useBlock(targetID || null);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const title =
    (target?.data?.title as string | undefined) ||
    (target?.text ?? undefined) ||
    "Untitled page";

  const handleJump = () => {
    if (targetID) jumpToBlock(targetID);
  };

  return (
    <BlockChrome
      className=""
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <button
        type="button"
        onClick={handleJump}
        className="inline-flex items-center gap-1 rounded border border-border bg-muted/40 px-2 py-1 text-sm hover:bg-muted"
      >
        <ArrowUpRight className="h-3.5 w-3.5" />
        <span>{title}</span>
      </button>
    </BlockChrome>
  );
}
