"use client";

import React from "react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";

import { NestChildren } from "../BlockRenderer";
import { EditableText } from "../editor/EditableText";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function PageRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const title = (block.data?.title as string | undefined) ?? "";

  return (
    <section className={cn("flex flex-col gap-3", depth === 0 && "py-6")}>
      <EditableText
        className={cn(
          "outline-none",
          depth === 0 ? "text-3xl font-bold tracking-tight" : "text-lg font-semibold",
        )}
        placeholder={depth === 0 ? "Untitled page" : "Subpage"}
        value={title}
        onChange={(next) => {
          dispatch.updateBlockData(block.id, { title: next });
        }}
      />
      <NestChildren parentID={block.id} depth={depth} />
    </section>
  );
}
