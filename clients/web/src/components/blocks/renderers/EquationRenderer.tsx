"use client";

import React, { useEffect, useRef } from "react";
import katex from "katex";
import "katex/dist/katex.min.css";
import { Sigma } from "lucide-react";

import type { Block } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";

import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

export function EquationRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);
  const latex = (block.data?.latex as string | undefined) ?? "";
  const display = (block.data?.display as string | undefined) === "inline" ? "inline" : "block";
  const hostRef = useRef<HTMLSpanElement | null>(null);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    if (!latex) {
      host.textContent = "";
      return;
    }
    try {
      katex.render(latex, host, {
        displayMode: display === "block",
        throwOnError: false,
      });
    } catch {
      host.textContent = latex;
    }
  }, [latex, display]);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  const onEdit = () => {
    const next = window.prompt("LaTeX:", latex);
    if (next !== null) {
      dispatch.updateBlockData(block.id, { latex: next }, { text: next });
    }
  };

  return (
    <BlockChrome
      className=""
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <button
        type="button"
        onClick={onEdit}
        className={cn(
          "flex items-center gap-2 rounded-md border border-border bg-muted/20 px-3 py-2 text-sm hover:bg-muted/40",
          display === "block" && "w-full justify-center",
        )}
      >
        <Sigma className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        {latex ? (
          <span ref={hostRef} />
        ) : (
          <span className="font-mono text-muted-foreground">(empty equation — click to edit)</span>
        )}
      </button>
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}
