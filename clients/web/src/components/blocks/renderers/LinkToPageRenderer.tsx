"use client";

import React from "react";
import { ArrowUpRight } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { useBlock } from "@/stores/blockstore";

import { BlockChrome } from "../editor/BlockChrome";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function LinkToPageRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
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
    if (!targetID) return;
    const el = document.getElementById(`block-${targetID}`);
    el?.scrollIntoView({ behavior: "smooth", block: "center" });
    el?.classList.add("ring-2", "ring-primary");
    setTimeout(() => el?.classList.remove("ring-2", "ring-primary"), 1500);
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
