"use client";

import React from "react";

import type { Block } from "@/lib/viewModels/blockstore";

import { NestChildren } from "../BlockRenderer";
import { EditableText } from "../editor/EditableText";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";
import { SubPageLink } from "./SubPageLink";

export function PageRenderer({ block, depth }: { block: Block; depth: number }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const title = (block.data?.title as string | undefined) ?? "";

  // A page nested inside another page is an independent document — render it as
  // a navigable sub-page link, never inline its content (Notion semantics).
  if (depth > 0) return <SubPageLink block={block} />;

  return (
    <section className="flex flex-col gap-3 py-6">
      <EditableText
        className="text-3xl font-bold tracking-tight outline-none"
        placeholder="Untitled page"
        value={title}
        onChange={(next) => {
          dispatch.updateBlockData(block.id, { title: next });
        }}
      />
      <NestChildren parentID={block.id} depth={depth} />
    </section>
  );
}
