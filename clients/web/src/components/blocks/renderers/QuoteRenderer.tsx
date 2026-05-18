"use client";

import React from "react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { BLOCK_TYPE_PARAGRAPH } from "@/lib/api/blockstoreTypes";

import { NestChildren } from "../BlockRenderer";
import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { EditableText } from "../editor/EditableText";
import { useAutoFocusIfPending } from "../editor/useAutoFocus";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";
import { readBlockText } from "./readBlockText";

// QuoteRenderer is a block-quote with a left border. Nested paragraphs and
// list items are allowed so users can write multi-line citations.
export function QuoteRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const autoFocus = useAutoFocusIfPending(block.id);
  const text = readBlockText(block);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  return (
    <BlockChrome
      className="flex flex-col gap-1"
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <div className="border-l-4 border-muted-foreground/40 pl-3 italic text-muted-foreground">
        <EditableText
          className="outline-none"
          placeholder="Quote…"
          value={text}
          autoFocus={autoFocus}
          onChange={(next) => dispatch.updateBlockData(block.id, { text: next }, { text: next })}
          onEnter={() => {
            void dispatch.insertSiblingAfter(block.id, BLOCK_TYPE_PARAGRAPH, { text: "" }, { text: "" });
          }}
          onBackspaceEmpty={handleDelete}
        />
      </div>
      <NestChildren parentID={block.id} depth={depth} />
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}
