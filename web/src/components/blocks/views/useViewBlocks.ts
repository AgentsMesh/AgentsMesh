"use client";

import { useMemo } from "react";

import type { Block, ViewFilter, ViewSort, ViewSpec } from "@/lib/api/blockstoreTypes";
import { useBlocks } from "@/stores/blockstore";

/**
 * Resolves the concrete block list that a `view` block should project over.
 * All filtering, sorting and grouping happens client-side against the local
 * store — the server stays dumb about view semantics.
 */
export function useViewBlocks(spec: ViewSpec, workspaceID: string): Block[] {
  const blocks = useBlocks();
  return useMemo(() => {
    // Tier 3: cross-type views union multiple source types. The single
    // source_type stays as the primary/default; source_types extends it.
    const typeSet = new Set<string>();
    if (spec.source_type) typeSet.add(spec.source_type);
    for (const t of spec.source_types ?? []) typeSet.add(t);
    const candidates = Object.values(blocks).filter(
      (b) =>
        b.workspace_id === workspaceID &&
        typeSet.has(b.type) &&
        !b.deleted_at,
    );
    const filtered = spec.filters?.length
      ? candidates.filter((b) => spec.filters!.every((f) => matchFilter(b, f)))
      : candidates;
    const sorted = spec.sort?.length
      ? [...filtered].sort((a, b) => compareBlocks(a, b, spec.sort!))
      : filtered;
    return sorted;
  }, [blocks, workspaceID, spec.source_type, spec.source_types, spec.filters, spec.sort]);
}

/** Groups pre-filtered blocks by the view's `group_by` key. Missing key falls into "". */
export function groupBlocks(blocks: Block[], groupKey: string): Record<string, Block[]> {
  const groups: Record<string, Block[]> = {};
  for (const b of blocks) {
    const k = ((b.data?.[groupKey] as string | undefined) ?? "") || "—";
    (groups[k] ??= []).push(b);
  }
  return groups;
}

function matchFilter(b: Block, f: ViewFilter): boolean {
  const actual = b.data?.[f.key];
  switch (f.op) {
    case "eq":
      return actual === f.value;
    case "ne":
      return actual !== f.value;
    case "contains":
      return typeof actual === "string" && typeof f.value === "string"
        ? actual.includes(f.value)
        : false;
    default:
      return true;
  }
}

function compareBlocks(a: Block, b: Block, sort: ViewSort[]): number {
  for (const s of sort) {
    const av = a.data?.[s.key];
    const bv = b.data?.[s.key];
    if (av === bv) continue;
    const less = (av ?? "") < (bv ?? "") ? -1 : 1;
    return s.direction === "desc" ? -less : less;
  }
  return 0;
}
