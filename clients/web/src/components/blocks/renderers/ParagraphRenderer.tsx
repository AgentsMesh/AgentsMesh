"use client";

import React, { useState } from "react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { BLOCK_TYPE_PARAGRAPH } from "@/lib/api/blockstoreTypes";

import { NestChildren } from "../BlockRenderer";
import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { EditableText } from "../editor/EditableText";
import { SlashMenu } from "../editor/SlashMenu";
import { standardSlashOptions } from "../editor/slashOptions";
import { useAutoFocusIfPending } from "../editor/useAutoFocus";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";
import { readBlockText } from "./readBlockText";

export function ParagraphRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const autoFocus = useAutoFocusIfPending(block.id);
  const text = readBlockText(block);
  const [slashOpen, setSlashOpen] = useState(false);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  return (
    <BlockChrome
      className="flex flex-col gap-1 pl-1 text-sm leading-relaxed"
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <EditableText
        className="outline-none"
        placeholder="Write something or press / for options"
        value={text}
        autoFocus={autoFocus}
        onChange={(next) => dispatch.updateBlockData(block.id, { text: next }, { text: next })}
        onEnter={() => {
          void dispatch.insertSiblingAfter(block.id, BLOCK_TYPE_PARAGRAPH, { text: "" }, { text: "" });
        }}
        onBackspaceEmpty={handleDelete}
        onSlashOnEmpty={() => setSlashOpen(true)}
      />
      {slashOpen && (
        <div className="absolute left-1 top-full z-50 mt-1">
          <SlashMenu
            open
            options={standardSlashOptions(dispatch, block.id)}
            onClose={() => setSlashOpen(false)}
          />
        </div>
      )}
      <NestChildren parentID={block.id} depth={depth} />
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}




