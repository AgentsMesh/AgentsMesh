"use client";

import React, { useCallback, useEffect, useMemo, useRef } from "react";

import type { Block, JSONMap } from "@/lib/api/blockstoreTypes";
import { BLOCK_TYPE_DOCUMENT } from "@/lib/api/blockstoreTypes";
import { BlockEditor } from "@/components/ui/block-editor";

import { BlockChrome } from "../editor/BlockChrome";
import { CommentsSection } from "../editor/CommentsSection";
import { useBlockstoreDispatch } from "../editor/useBlockstoreDispatch";

// BlockNote's onChange fires per keystroke. Without a debounce we'd send one
// applyOps POST per character (plus a WS broadcast per POST), which on flaky
// networks turns into a request pile-up and on busy workspaces amplifies WS
// traffic N×. 300ms is the standard "feels instant" window; the trailing
// edge fires so the final keystroke still flushes without waiting for a
// user idle timer.
const SAVE_DEBOUNCE_MS = 300;

// DocumentRenderer embeds BlockNote as a rich-text editor for a single
// Block Store block. data.blocknote_ast holds the BlockNote JSON tree; the
// top-level Block.text is maintained as a flattened plain string so search
// and semantic embeddings see the document contents.
// Architectural note: this is exactly the split from the Plan — Block Store
// manages structure between document units (nest / mention / ref), and
// BlockNote handles inline formatting + slash commands inside one unit.
export function DocumentRenderer({ block }: { block: Block }) {
  const dispatch = useBlockstoreDispatch(block.workspace_id);

  const initialContent = useMemo(() => {
    const ast = (block.data as JSONMap | undefined)?.blocknote_ast;
    if (!ast) return undefined;
    try {
      return JSON.stringify(ast);
    } catch {
      return undefined;
    }
  }, [block.data]);

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingRef = useRef<{ ast: unknown; text: string } | null>(null);

  const flush = useCallback(() => {
    const pending = pendingRef.current;
    if (!pending) return;
    pendingRef.current = null;
    timerRef.current = null;
    void dispatch.updateBlockData(
      block.id,
      { blocknote_ast: pending.ast },
      { text: pending.text },
    );
  }, [dispatch, block.id]);

  const handleChange = useCallback(
    (json: string) => {
      let parsed: unknown = null;
      try {
        parsed = JSON.parse(json);
      } catch {
        return;
      }
      pendingRef.current = { ast: parsed, text: flattenBlockNoteAST(parsed) };
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(flush, SAVE_DEBOUNCE_MS);
    },
    [flush],
  );

  // Final flush on unmount so the last keystroke is never lost when the user
  // navigates away or the block is deleted mid-edit.
  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        flush();
      }
    };
  }, [flush]);

  const handleDelete = () => {
    void dispatch.detachChild(block.id);
    void dispatch.removeBlock(block.id);
  };

  return (
    <BlockChrome
      className=""
      blockID={block.id}
      onDelete={handleDelete}
      onDuplicate={() => void dispatch.duplicate(block.id)}
      onToggleVisibility={(next) => void dispatch.setBlockVisibility(block.id, next)}
    >
      <div className="rounded-md border border-border bg-muted/10">
        <BlockEditor
          initialContent={initialContent}
          onChange={handleChange}
          placeholder="Write a document…"
          className="min-h-[8rem] p-2"
        />
      </div>
      <CommentsSection blockID={block.id} workspaceID={block.workspace_id} />
    </BlockChrome>
  );
}

// flattenBlockNoteAST walks the BlockNote AST (array of blocks) and
// concatenates every inline text fragment. The result feeds Block.text so
// search / embeddings work over the document body without needing a special
// "document" path. Kept forgiving — any node shape we don't recognise is
// skipped rather than throwing.
function flattenBlockNoteAST(ast: unknown): string {
  const out: string[] = [];
  const visit = (node: unknown) => {
    if (!node || typeof node !== "object") return;
    const n = node as Record<string, unknown>;
    const content = n.content;
    if (typeof n.text === "string") out.push(n.text);
    if (Array.isArray(content)) content.forEach(visit);
    if (Array.isArray(n.children)) n.children.forEach(visit);
  };
  if (Array.isArray(ast)) ast.forEach(visit);
  return out.join("").trim();
}

export { BLOCK_TYPE_DOCUMENT };
