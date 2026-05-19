"use client";

import React from "react";
import { Plus, Trash } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";

import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function TableBlockRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const rows = extractRows(block.data?.rows);
  const headerRow = (block.data?.header_row as boolean | undefined) ?? true;

  const commit = (next: string[][]) => {
    void dispatch.updateBlockData(block.id, { rows: next });
  };

  const setCell = (r: number, c: number, value: string) => {
    const next = rows.map((row) => row.slice());
    while (next.length <= r) next.push([]);
    while (next[r].length <= c) next[r].push("");
    next[r][c] = value;
    commit(next);
  };

  const addRow = () => {
    const width = rows[0]?.length ?? 2;
    commit([...rows, new Array(width).fill("")]);
  };

  const addCol = () => {
    commit(rows.map((row) => [...row, ""]));
  };

  const removeRow = (r: number) => {
    commit(rows.filter((_, i) => i !== r));
  };

  const removeCol = (c: number) => {
    commit(rows.map((row) => row.filter((_, i) => i !== c)));
  };

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const effectiveRows = rows.length > 0 ? rows : [["", ""], ["", ""]];

  return (
    <BlockChrome
      className=""
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <div className="overflow-x-auto rounded border border-border">
        <table className="w-full text-sm">
          <tbody>
            {effectiveRows.map((row, r) => (
              <tr key={r} className="group/row">
                {row.map((cell, c) => (
                  <td
                    key={c}
                    className={cn(
                      "border border-border/60 p-0 align-top",
                      headerRow && r === 0 && "bg-muted/40 font-semibold",
                    )}
                  >
                    <input
                      type="text"
                      value={cell}
                      onChange={(e) => setCell(r, c, e.target.value)}
                      className="w-full bg-transparent px-2 py-1 outline-none"
                    />
                  </td>
                ))}
                <td className="w-6 p-0">
                  <button
                    type="button"
                    onClick={() => removeRow(r)}
                    className="invisible px-1 text-muted-foreground hover:text-destructive group-hover/row:visible"
                    aria-label="Remove row"
                  >
                    <Trash className="h-3 w-3" />
                  </button>
                </td>
              </tr>
            ))}
            <tr>
              {(effectiveRows[0] ?? []).map((_, c) => (
                <td key={c} className="p-0">
                  <button
                    type="button"
                    onClick={() => removeCol(c)}
                    className="w-full px-2 py-1 text-xs text-muted-foreground hover:text-destructive"
                    aria-label="Remove column"
                  >
                    ×
                  </button>
                </td>
              ))}
              <td />
            </tr>
          </tbody>
        </table>
      </div>
      <div className="mt-1 flex items-center gap-2 text-xs">
        <button type="button" onClick={addRow} className="flex items-center gap-1 text-muted-foreground hover:text-foreground">
          <Plus className="h-3 w-3" /> Row
        </button>
        <button type="button" onClick={addCol} className="flex items-center gap-1 text-muted-foreground hover:text-foreground">
          <Plus className="h-3 w-3" /> Column
        </button>
        <label className="ml-auto flex items-center gap-1 text-muted-foreground">
          <input
            type="checkbox"
            checked={headerRow}
            onChange={(e) => dispatch.updateBlockData(block.id, { header_row: e.target.checked })}
          />
          Header row
        </label>
      </div>
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}

function extractRows(raw: unknown): string[][] {
  if (!Array.isArray(raw)) return [];
  return raw.map((row) => (Array.isArray(row) ? row.map((cell) => String(cell ?? "")) : []));
}
