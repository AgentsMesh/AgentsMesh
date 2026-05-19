"use client";

import React from "react";

import type { Block, ViewSpec } from "@/lib/api/blockstoreTypes";

import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

import { ViewHeader } from "./ViewListFallback";
import { SummaryBar } from "./SummaryBar";
import { useViewBlocks } from "./useViewBlocks";

export function GalleryView({ viewBlock, spec }: { viewBlock: Block; spec: ViewSpec }) {
  const items = useViewBlocks(spec, viewBlock.workspace_id);
  const dispatch = useBlockstoreDispatch(viewBlock.workspace_id);

  return (
    <section className="flex flex-col gap-3">
      <ViewHeader title={spec.title ?? `${spec.source_type} · gallery`} count={items.length} />
      <SummaryBar blocks={items} summaryColumns={spec.summary_columns ?? []} />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {items.map((b) => (
          <Card key={b.id} block={b} />
        ))}
        <button
          type="button"
          onClick={() => dispatch.insertChild(viewBlock.id, spec.source_type, { title: "" })}
          className="flex h-40 items-center justify-center rounded-md border border-dashed border-muted-foreground/40 text-xs text-muted-foreground hover:border-muted-foreground/70 hover:text-foreground"
        >
          + Add {spec.source_type}
        </button>
      </div>
    </section>
  );
}

function Card({ block }: { block: Block }) {
  const title =
    (block.data?.title as string | undefined) ??
    (block.data?.name as string | undefined) ??
    block.id.slice(0, 8);
  const cover = block.data?.cover_url as string | undefined;
  return (
    <article className="flex h-40 flex-col overflow-hidden rounded-md border border-border bg-background shadow-sm">
      {cover ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={cover} alt="" className="h-24 w-full object-cover" />
      ) : (
        <div className="h-24 w-full bg-gradient-to-br from-muted to-accent/40" />
      )}
      <div className="flex flex-1 items-center px-3 text-sm font-medium">{title}</div>
    </article>
  );
}
