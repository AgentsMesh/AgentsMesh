"use client";

import React from "react";

import type { Block, BlockTypeSpec, ColumnSpec, JSONMap } from "@/lib/api/blockstoreTypes";
import { computeColumn } from "@/lib/blockstore/computeColumn";

import { NestChildren } from "../BlockRenderer";
import { BlockChrome } from "./BlockChrome";
import { CommentsSection } from "./CommentsSection";
import { FieldRenderer } from "./FieldRenderer";
import { useBlockstoreDispatch } from "./useBlockstoreDispatch";

export interface RecordEditorProps {
  block: Block;
  spec: BlockTypeSpec;
  depth: number;
}

// RecordEditor is the generic schema-driven renderer. It is used by
// BlockRenderer whenever a block's type resolves to a spec with columns —
// that is, any Agent- or user-defined indicator. Layout is a simple two-
// column label/field grid; a richer layout (inline columns, grouped
// sections) can wrap this component without changing the field logic.
// Tier 2 hooks:
//   - deprecated columns are skipped so schema evolution keeps working
//     records visible without clutter
//   - computed columns render as read-only derived values
export function RecordEditor({ block, spec, depth }: RecordEditorProps) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const columns = spec.columns ?? [];
  const data = (block.data ?? {}) as JSONMap;
  const visibleColumns = columns.filter((c) => !c.deprecated);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const updateColumn = (key: string, next: unknown) => {
    // Preserve the existing data fields (avoid clobbering columns we didn't
    // render). Also re-derive the top-level text from whichever column the
    // schema marks as the "title" — conventionally the first text column.
    const patch: JSONMap = { [key]: next };
    const titleKey = firstTextColumn(columns);
    const newText = titleKey
      ? asString(titleKey === key ? next : data[titleKey])
      : undefined;
    void dispatch.updateBlockData(
      block.id,
      patch,
      newText !== undefined ? { text: newText } : undefined,
    );
  };

  return (
    <BlockChrome
      className="flex flex-col gap-2"
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <div className="rounded-md border border-border bg-muted/20 p-3">
        <div className="mb-2 flex items-baseline gap-2">
          <span className="rounded bg-muted px-1.5 py-0.5 text-xs uppercase tracking-wide text-muted-foreground">
            {spec.label ?? spec.type}
          </span>
          {spec.description && (
            <span className="text-xs text-muted-foreground">{spec.description}</span>
          )}
        </div>
        <div className="grid grid-cols-[minmax(6rem,max-content)_1fr] items-center gap-x-3 gap-y-2">
          {visibleColumns.map((col) => (
            <React.Fragment key={col.key}>
              <label className="text-xs font-medium text-muted-foreground">
                {col.label ?? col.key}
                {col.required && <span className="ml-0.5 text-destructive">*</span>}
                {col.computed && (
                  <span className="ml-1 text-[10px] uppercase text-muted-foreground/70">
                    (computed)
                  </span>
                )}
              </label>
              {col.computed ? (
                <ComputedValue column={col} data={data} />
              ) : (
                <FieldRenderer
                  column={col}
                  value={data[col.key]}
                  onChange={(next) => updateColumn(col.key, next)}
                />
              )}
            </React.Fragment>
          ))}
        </div>
      </div>
      <NestChildren parentID={block.id} depth={depth} />
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}

function firstTextColumn(columns: BlockTypeSpec["columns"]): string | null {
  if (!columns) return null;
  const textCol = columns.find(
    (c) => c.type === "text" && c.required && !c.deprecated && !c.computed,
  );
  if (textCol) return textCol.key;
  const anyText = columns.find(
    (c) => c.type === "text" && !c.deprecated && !c.computed,
  );
  return anyText ? anyText.key : null;
}

function asString(v: unknown): string {
  return typeof v === "string" ? v : "";
}

// ComputedValue renders the derived value for a computed column. Evaluation
// happens on every render — cheap enough for our expression grammar. Invalid
// expressions show "—" so the schema author sees the broken cell without
// breaking the record editor.
function ComputedValue({ column, data }: { column: ColumnSpec; data: JSONMap }) {
  const value = computeColumn(column.computed ?? "", data);
  return (
    <span className="font-mono text-sm text-muted-foreground">
      {value === null ? "—" : value}
    </span>
  );
}
