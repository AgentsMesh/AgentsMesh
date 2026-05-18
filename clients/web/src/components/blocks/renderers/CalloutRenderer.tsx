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

// CalloutRenderer is a boxed attention block with an optional emoji prefix.
// Common usages: ℹ️ note, ⚠️ warning, 💡 tip. Nested content allowed.
export function CalloutRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const autoFocus = useAutoFocusIfPending(block.id);
  const text = readBlockText(block);
  const emoji = (block.data?.emoji as string | undefined) ?? "💡";

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
      <div className="flex gap-2 rounded-md border border-border bg-muted/40 p-3">
        <button
          type="button"
          onClick={() => {
            const next = window.prompt("Emoji:", emoji);
            if (next !== null) dispatch.updateBlockData(block.id, { emoji: next });
          }}
          className="flex h-5 w-5 shrink-0 items-center justify-center rounded hover:bg-muted"
          aria-label="Change emoji"
        >
          {emoji}
        </button>
        <div className="flex-1">
          <EditableText
            className="outline-none"
            placeholder="Write a callout…"
            value={text}
            autoFocus={autoFocus}
            onChange={(next) => dispatch.updateBlockData(block.id, { text: next }, { text: next })}
            onEnter={() => {
              void dispatch.insertSiblingAfter(block.id, BLOCK_TYPE_PARAGRAPH, { text: "" }, { text: "" });
            }}
            onBackspaceEmpty={handleDelete}
          />
        </div>
      </div>
      <NestChildren parentID={block.id} depth={depth} />
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}
