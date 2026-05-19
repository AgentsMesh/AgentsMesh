"use client";

import React from "react";

import type { Block } from "@/lib/api/blockstoreTypes";
import {
  BLOCK_TYPE_BULLETED_LIST_ITEM,
  BLOCK_TYPE_NUMBERED_LIST_ITEM,
} from "@/lib/api/blockstoreTypes";

import { NestChildren } from "../BlockRenderer";
import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { EditableText } from "../editor/EditableText";
import { useAutoFocusIfPending } from "../editor/useAutoFocus";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";
import { useRefs, useNestChildrenIndex, useBlocks } from "@/stores/blockstore";
import { readBlockText } from "./readBlockText";

export function ListItemRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const autoFocus = useAutoFocusIfPending(block.id);
  const text = readBlockText(block);
  const numbered = block.type === BLOCK_TYPE_NUMBERED_LIST_ITEM;
  const marker = useListMarker(block.id, numbered);

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
      <div className="flex items-start gap-2">
        <span className="w-5 shrink-0 select-none pt-0.5 text-right text-sm text-muted-foreground">
          {marker}
        </span>
        <EditableText
          className="flex-1 outline-none"
          placeholder={numbered ? "List item" : "Bullet item"}
          value={text}
          autoFocus={autoFocus}
          onChange={(next) => dispatch.updateBlockData(block.id, { text: next }, { text: next })}
          onEnter={() => {
            const type = numbered ? BLOCK_TYPE_NUMBERED_LIST_ITEM : BLOCK_TYPE_BULLETED_LIST_ITEM;
            void dispatch.insertSiblingAfter(block.id, type, { text: "" }, { text: "" });
          }}
          onBackspaceEmpty={handleDelete}
        />
      </div>
      <div className="pl-7">
        <NestChildren parentID={block.id} depth={depth} />
      </div>
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}

function useListMarker(blockID: string, numbered: boolean): string {
  const refs = useRefs();
  const nestChildren = useNestChildrenIndex();
  const blocks = useBlocks();
  if (!numbered) return "•";
  const selfRef = Object.values(refs).find(
    (r) => r.rel === "nest" && r.to_id === blockID,
  );
  if (!selfRef) return "1.";
  const parentID = selfRef.from_id;
  const childRefIDs: number[] = nestChildren[parentID] ?? [];
  let counter = 0;
  for (const refID of childRefIDs) {
    const ref = refs[refID];
    if (!ref) continue;
    const child = blocks[ref.to_id];
    if (child?.type !== BLOCK_TYPE_NUMBERED_LIST_ITEM) {
      counter = 0;
      continue;
    }
    counter += 1;
    if (child.id === blockID) return `${counter}.`;
  }
  return "1.";
}
