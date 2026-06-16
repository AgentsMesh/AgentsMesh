"use client";

import React from "react";
import { FileText } from "lucide-react";

import type { Block } from "@/lib/viewModels/blockstore";
import { pageDisplayMeta } from "@/lib/blockstore/pageDisplayMeta";
import { useSelectPage } from "@/lib/blockstore/useSelectPage";

import { BlockChrome } from "../editor/BlockChrome";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function SubPageLink({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const selectPage = useSelectPage();

  const { title, icon } = pageDisplayMeta(block);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  return (
    <BlockChrome
      className="pl-1"
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <button
        type="button"
        onClick={() => selectPage(block.id)}
        data-testid={`blocks-subpage-link-${block.id}`}
        className="inline-flex items-center gap-1.5 rounded px-1 py-0.5 text-left text-sm font-medium text-foreground hover:bg-muted/50"
      >
        <span aria-hidden="true" className="flex-shrink-0 text-muted-foreground">
          {icon ?? <FileText className="inline h-4 w-4" />}
        </span>
        <span className="truncate underline decoration-muted-foreground/40 underline-offset-2">
          {title}
        </span>
      </button>
    </BlockChrome>
  );
}
