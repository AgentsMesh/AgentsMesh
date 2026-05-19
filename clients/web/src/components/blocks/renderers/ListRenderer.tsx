"use client";

import React from "react";

import type { Block } from "@/lib/api/blockstoreTypes";

import { NestChildren } from "../BlockRenderer";
import { BlockChrome } from "../editor/BlockChrome";
import { EditableText } from "../editor/EditableText";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function ListRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const name = (block.data?.name as string | undefined) ?? "";

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  return (
    <BlockChrome
      className="flex flex-col gap-2"
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <EditableText
        className="text-base font-medium outline-none"
        placeholder="List"
        value={name}
        onChange={(next) => dispatch.updateBlockData(block.id, { name: next })}
      />
      <ul className="list-disc space-y-1 pl-5">
        <NestChildren parentID={block.id} depth={depth} />
      </ul>
    </BlockChrome>
  );
}
