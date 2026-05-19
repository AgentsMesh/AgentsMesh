"use client";

import { useMemo } from "react";

import {
  BLOCK_TYPE_TYPEDEF,
  type Block,
  type BlockTypeSpec,
  type ColumnSpec,
} from "@/lib/api/blockstoreTypes";
import { useBlocks, useBlockstoreStore } from "@/stores/blockstore";

// useBlockTypeSpecs hydrates the workspace's full type registry by scanning
// every block_type_def block currently in the store. Memoised on the ref
// identity of s.blocks so it only recomputes when the block map itself
// changes. Later revisions of the same type_key supersede earlier ones.
export function useBlockTypeSpecs(workspaceID: string): Record<string, BlockTypeSpec> {
  const blocks = useBlocks();
  return useMemo(() => buildSpecMap(blocks, workspaceID), [blocks, workspaceID]);
}

export function useBlockTypeSpec(workspaceID: string, typeKey: string): BlockTypeSpec | null {
  const all = useBlockTypeSpecs(workspaceID);
  return all[typeKey] ?? null;
}

export function buildSpecMap(
  blocks: Record<string, Block>,
  workspaceID: string,
): Record<string, BlockTypeSpec> {
  const out: Record<string, BlockTypeSpec> = {};
  for (const block of Object.values(blocks)) {
    if (block.workspace_id !== workspaceID) continue;
    if (block.type !== BLOCK_TYPE_TYPEDEF) continue;
    const spec = decodeTypeDef(block);
    if (!spec) continue;
    const existing = out[spec.type];
    if (existing && (existing.revision ?? 0) >= (spec.revision ?? 0)) continue;
    out[spec.type] = spec;
  }
  return out;
}

function decodeTypeDef(block: Block): BlockTypeSpec | null {
  const data = block.data;
  if (!data || typeof data !== "object") return null;
  const typeKey = typeof data.type_key === "string" ? data.type_key : "";
  if (!typeKey) return null;
  const columns = Array.isArray(data.columns)
    ? (data.columns as ColumnSpec[]).filter((c) => c && typeof c.key === "string" && typeof c.type === "string")
    : undefined;
  return {
    type: typeKey,
    revision: typeof data.revision === "number" ? data.revision : 0,
    label: typeof data.label === "string" ? data.label : undefined,
    description: typeof data.description === "string" ? data.description : undefined,
    default_view: typeof data.default_view === "string" ? data.default_view : undefined,
    supported_views: Array.isArray(data.supported_views) ? (data.supported_views as string[]) : undefined,
    allowed_children: Array.isArray(data.allowed_children) ? (data.allowed_children as string[]) : undefined,
    columns,
  };
}
