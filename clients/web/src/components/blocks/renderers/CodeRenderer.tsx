"use client";

import React, { useState } from "react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { BLOCK_TYPE_PARAGRAPH } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";

import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { useAutoFocusIfPending } from "../editor/useAutoFocus";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

const LANGUAGES = [
  "plain", "bash", "go", "typescript", "javascript", "python",
  "rust", "java", "c", "cpp", "sql", "json", "yaml", "markdown",
];

export function CodeRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const autoFocus = useAutoFocusIfPending(block.id);
  const code = (block.data?.code as string | undefined) ?? "";
  const language = (block.data?.language as string | undefined) ?? "plain";
  const [localCode, setLocalCode] = useState(code);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const flush = (next: string) => {
    if (next !== code) dispatch.updateBlockData(block.id, { code: next }, { text: next });
  };

  return (
    <BlockChrome
      className="flex flex-col gap-1"
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <div className="flex flex-col overflow-hidden rounded-md border border-border bg-muted/30">
        <div className="flex items-center justify-between border-b border-border/60 px-2 py-1 text-xs text-muted-foreground">
          <select
            value={language}
            onChange={(e) => dispatch.updateBlockData(block.id, { language: e.target.value })}
            className="bg-transparent outline-none"
          >
            {LANGUAGES.map((lang) => (
              <option key={lang} value={lang}>{lang}</option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => navigator.clipboard?.writeText(code)}
            className="hover:text-foreground"
          >
            copy
          </button>
        </div>
        <textarea
          autoFocus={autoFocus}
          value={localCode}
          onChange={(e) => setLocalCode(e.target.value)}
          onBlur={(e) => flush(e.target.value)}
          placeholder="Paste or type code…"
          spellCheck={false}
          className={cn(
            "min-h-[4rem] resize-y bg-transparent p-2 font-mono text-sm outline-none",
          )}
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
              e.preventDefault();
              flush((e.target as HTMLTextAreaElement).value);
              void dispatch.insertSiblingAfter(block.id, BLOCK_TYPE_PARAGRAPH, { text: "" }, { text: "" });
            }
          }}
        />
      </div>
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}
