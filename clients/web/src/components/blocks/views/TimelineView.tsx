"use client";

import React, { useMemo } from "react";

import type { Block, ViewSpec } from "@/lib/api/blockstoreTypes";

import { ViewHeader } from "./ViewListFallback";
import { SummaryBar } from "./SummaryBar";
import { useViewBlocks } from "./useViewBlocks";

export function TimelineView({ viewBlock, spec }: { viewBlock: Block; spec: ViewSpec }) {
  const items = useViewBlocks(spec, viewBlock.workspace_id);
  const dated = useMemo(() => normalize(items), [items]);
  const { min, max } = useMemo(() => boundsOf(dated), [dated]);
  const spanDays = Math.max(1, Math.round((max - min) / DAY_MS));

  return (
    <section className="flex flex-col gap-3">
      <ViewHeader title={spec.title ?? `${spec.source_type} · timeline`} count={dated.length} />
      <SummaryBar blocks={dated.map((d) => d.block)} summaryColumns={spec.summary_columns ?? []} />
      {dated.length === 0 ? (
        <EmptyTimeline />
      ) : (
        <div className="relative overflow-x-auto rounded border border-border p-3">
          <div className="mb-2 text-xs text-muted-foreground">
            {new Date(min).toISOString().slice(0, 10)} → {new Date(max).toISOString().slice(0, 10)}
            <span className="ml-2">({spanDays} days)</span>
          </div>
          <ol className="flex flex-col gap-1">
            {dated.map(({ block, start, end }) => {
              const leftPct = ((start - min) / (max - min || 1)) * 100;
              const widthPct = Math.max(2, ((end - start) / (max - min || 1)) * 100);
              return (
                <li key={block.id} className="relative h-7 rounded bg-muted/30">
                  <div
                    className="absolute top-0 flex h-full items-center rounded bg-primary/70 px-2 text-xs text-primary-foreground"
                    style={{ left: `${leftPct}%`, width: `${widthPct}%` }}
                  >
                    <span className="truncate">{(block.data?.title as string | undefined) ?? block.id.slice(0, 8)}</span>
                  </div>
                </li>
              );
            })}
          </ol>
        </div>
      )}
    </section>
  );
}

const DAY_MS = 24 * 60 * 60 * 1000;

interface Dated { block: Block; start: number; end: number }

function normalize(items: Block[]): Dated[] {
  const out: Dated[] = [];
  for (const b of items) {
    const s = Date.parse((b.data?.start_date as string | undefined) ?? "");
    const e = Date.parse((b.data?.end_date as string | undefined) ?? "");
    if (Number.isFinite(s) && Number.isFinite(e) && e >= s) {
      out.push({ block: b, start: s, end: e });
    }
  }
  return out;
}

function boundsOf(items: Dated[]): { min: number; max: number } {
  if (items.length === 0) return { min: Date.now(), max: Date.now() + DAY_MS };
  let min = items[0].start;
  let max = items[0].end;
  for (const it of items) {
    if (it.start < min) min = it.start;
    if (it.end > max) max = it.end;
  }
  return { min, max };
}

function EmptyTimeline() {
  return (
    <div className="rounded border border-dashed border-muted-foreground/40 p-3 text-xs text-muted-foreground">
      No blocks with valid <code className="font-mono">start_date</code> / <code className="font-mono">end_date</code> fields.
    </div>
  );
}
