"use client";

import React from "react";

import type { AggregateOp, Block, JSONMap, SummaryColumn } from "@/lib/api/blockstoreTypes";

export interface SummaryBarProps {
  blocks: Block[];
  summaryColumns: SummaryColumn[];
}

export function SummaryBar({ blocks, summaryColumns }: SummaryBarProps) {
  if (summaryColumns.length === 0) return null;
  return (
    <div className="mb-2 flex flex-wrap items-baseline gap-4 rounded-md border border-border bg-muted/20 px-3 py-1.5 text-xs">
      {summaryColumns.map((sc, i) => {
        const value = computeAggregate(blocks, sc);
        const label = sc.label ?? deriveLabel(sc);
        return (
          <div key={`${sc.key}-${sc.aggregate}-${i}`} className="flex items-baseline gap-1.5">
            <span className="text-muted-foreground">{label}</span>
            <span className="font-medium">{formatValue(value, sc.format)}</span>
          </div>
        );
      })}
    </div>
  );
}

// computeAggregate walks blocks.data[sc.key] and reduces. Non-numeric values
// fall through for numeric ops (returning NaN so the renderer shows "—").
export function computeAggregate(blocks: Block[], sc: SummaryColumn): number {
  switch (sc.aggregate) {
    case "count":
      return blocks.length;
    case "count_distinct": {
      const seen = new Set<string>();
      for (const b of blocks) {
        const v = (b.data as JSONMap)[sc.key];
        if (v !== undefined && v !== null) seen.add(JSON.stringify(v));
      }
      return seen.size;
    }
    case "sum":
      return numericReduce(blocks, sc.key, 0, (acc, n) => acc + n);
    case "avg": {
      const nums = numericValues(blocks, sc.key);
      return nums.length === 0 ? NaN : nums.reduce((a, b) => a + b, 0) / nums.length;
    }
    case "min": {
      const nums = numericValues(blocks, sc.key);
      return nums.length === 0 ? NaN : Math.min(...nums);
    }
    case "max": {
      const nums = numericValues(blocks, sc.key);
      return nums.length === 0 ? NaN : Math.max(...nums);
    }
    default:
      return NaN;
  }
}

function numericValues(blocks: Block[], key: string): number[] {
  const out: number[] = [];
  for (const b of blocks) {
    const v = (b.data as JSONMap)[key];
    if (typeof v === "number" && Number.isFinite(v)) out.push(v);
  }
  return out;
}

function numericReduce(
  blocks: Block[],
  key: string,
  init: number,
  f: (acc: number, n: number) => number,
): number {
  let acc = init;
  for (const n of numericValues(blocks, key)) acc = f(acc, n);
  return acc;
}

function deriveLabel(sc: SummaryColumn): string {
  if (sc.aggregate === "count") return "Count";
  if (sc.aggregate === "count_distinct") return `Distinct ${sc.key}`;
  const op = sc.aggregate[0].toUpperCase() + sc.aggregate.slice(1);
  return `${op} ${sc.key}`;
}

function formatValue(value: number, format?: SummaryColumn["format"]): string {
  if (!Number.isFinite(value)) return "—";
  switch (format) {
    case "int":
      return Math.round(value).toLocaleString();
    case "percent":
      return `${(value * 100).toFixed(1)}%`;
    case "date":
      return new Date(value).toLocaleDateString();
    default:
      return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
  }
}

export type { AggregateOp };
