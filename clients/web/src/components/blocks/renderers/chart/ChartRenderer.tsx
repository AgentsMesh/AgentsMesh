"use client";

import React, { useMemo, useState } from "react";
import { BarChart3, Check, Pencil, X } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";

import { BlockChrome } from "../../editor/BlockChrome";
import { CommentsSection } from "../../editor/CommentsSection";
import { useBlockstoreDispatch } from "../../editor/useBlockstoreDispatch";

import { ChartPreview } from "./ChartPreview";

export function ChartRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const title = (block.data?.title as string | undefined) ?? "Chart";
  const chartType = (block.data?.type as string | undefined) ?? "bar";

  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(() => serialize(block.data));
  const [error, setError] = useState<string | null>(null);

  const parsedDraft = useMemo(() => safeParse(draft), [draft]);
  const previewData = editing && parsedDraft ? parsedDraft : block.data;

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const openEditor = () => {
    setDraft(serialize(block.data));
    setError(null);
    setEditing(true);
  };

  const cancel = () => {
    setEditing(false);
    setError(null);
  };

  const apply = async () => {
    const { ok, value, message } = validate(draft);
    if (!ok) {
      setError(message);
      return;
    }
    const nextTitle = (value.title as string | undefined) ?? "";
    await dispatch.updateBlockData(block.id, value, { text: nextTitle });
    setEditing(false);
    setError(null);
  };

  return (
    <BlockChrome
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <div className="flex flex-col gap-2 rounded-md border border-border bg-muted/10 p-3">
        <header className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2 text-sm">
            <BarChart3 className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            <span className="truncate font-medium">{title}</span>
            <span className="shrink-0 text-xs text-muted-foreground">· {chartType}</span>
          </div>
          {editing ? (
            <EditorActions onCancel={cancel} onApply={apply} />
          ) : (
            <button
              type="button"
              onClick={openEditor}
              className="flex items-center gap-1 rounded border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-muted/50"
            >
              <Pencil className="h-3 w-3" />
              Edit
            </button>
          )}
        </header>

        {editing ? (
          <div className="flex flex-col gap-2 md:flex-row">
            <textarea
              value={draft}
              onChange={(e) => {
                setDraft(e.target.value);
                setError(null);
              }}
              spellCheck={false}
              className={cn(
                "h-[300px] min-w-0 flex-1 rounded-md border border-border bg-background p-2 font-mono text-xs",
                "focus:outline-none focus:ring-1 focus:ring-ring",
              )}
            />
            <div className="min-w-0 flex-1">
              <ChartPreview data={previewData ?? {}} height={300} />
            </div>
          </div>
        ) : (
          <ChartPreview data={block.data ?? {}} />
        )}
        {error && <p className="text-xs text-destructive">{error}</p>}
      </div>
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}

function EditorActions({ onCancel, onApply }: { onCancel: () => void; onApply: () => void }) {
  return (
    <div className="flex items-center gap-1">
      <button
        type="button"
        onClick={onCancel}
        className="flex items-center gap-1 rounded border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-muted/50"
      >
        <X className="h-3 w-3" />
        Cancel
      </button>
      <button
        type="button"
        onClick={onApply}
        className="flex items-center gap-1 rounded bg-primary px-2 py-1 text-xs text-primary-foreground hover:bg-primary/90"
      >
        <Check className="h-3 w-3" />
        Apply
      </button>
    </div>
  );
}

function serialize(data: unknown): string {
  try {
    return JSON.stringify(data ?? {}, null, 2);
  } catch {
    return "{}";
  }
}

function safeParse(text: string): Record<string, unknown> | null {
  try {
    const v = JSON.parse(text);
    return v && typeof v === "object" && !Array.isArray(v) ? (v as Record<string, unknown>) : null;
  } catch {
    return null;
  }
}

interface ValidationResult {
  ok: boolean;
  value: Record<string, unknown>;
  message: string;
}

function validate(text: string): ValidationResult {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch (e) {
    return { ok: false, value: {}, message: `Invalid JSON: ${(e as Error).message}` };
  }
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { ok: false, value: {}, message: "Top-level must be a JSON object" };
  }
  const obj = parsed as Record<string, unknown>;
  const type = obj.type;
  if (typeof type !== "string" || !["bar", "line", "pie", "area", "scatter", "radar"].includes(type)) {
    return { ok: false, value: {}, message: 'Field "type" must be one of bar/line/pie/area/scatter/radar' };
  }
  if (!Array.isArray(obj.series)) {
    return { ok: false, value: {}, message: 'Field "series" must be an array' };
  }
  return { ok: true, value: obj, message: "" };
}
