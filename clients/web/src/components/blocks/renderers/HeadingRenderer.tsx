"use client";

import React from "react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { BLOCK_TYPE_PARAGRAPH } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";

import { NestChildren } from "../BlockRenderer";
import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { EditableText } from "../editor/EditableText";
import { useAutoFocusIfPending } from "../editor/useAutoFocus";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";
import { readBlockText } from "./readBlockText";

type HeadingLevel = 1 | 2 | 3;

// HeadingRenderer displays H1/H2/H3 based on data.level (default 1). Enter
// drops back to a paragraph below, matching the most common editor ergonomic.
export function HeadingRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const autoFocus = useAutoFocusIfPending(block.id);
  const text = readBlockText(block);
  const rawLevel = block.data?.level;
  const level: HeadingLevel = rawLevel === 2 || rawLevel === 3 ? rawLevel : 1;

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const levelStyle = cn(
    "outline-none font-semibold",
    level === 1 && "text-3xl",
    level === 2 && "text-2xl",
    level === 3 && "text-xl",
  );

  return (
    <BlockChrome
      className="flex flex-col gap-1"
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <EditableText
        className={levelStyle}
        placeholder={`Heading ${level}`}
        value={text}
        autoFocus={autoFocus}
        onChange={(next) => dispatch.updateBlockData(block.id, { text: next }, { text: next })}
        onEnter={() => {
          void dispatch.insertSiblingAfter(block.id, BLOCK_TYPE_PARAGRAPH, { text: "" }, { text: "" });
        }}
        onBackspaceEmpty={handleDelete}
      />
      <NestChildren parentID={block.id} depth={depth} />
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}
