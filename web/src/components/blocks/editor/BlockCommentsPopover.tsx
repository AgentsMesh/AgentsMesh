"use client";

import { useMemo, useState } from "react";
import { Send, X } from "lucide-react";

import type { BlockRef } from "@/lib/api/blockstoreTypes";
import { REL_COMMENTS_ON } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";
import { useRefs, useBlock } from "@/stores/blockstore";
import { useBlockstoreDispatch } from "./useBlockstoreDispatch";

interface BlockCommentsPopoverProps {
  blockID: string;
  workspaceID: string;
  onClose: () => void;
}

/**
 * Floating comment thread anchored to the right edge of a block. Position is
 * `absolute left-full top-0 ml-3` so it rides with the block as the document
 * scrolls, matching the Notion/Google Docs margin-comment pattern. The parent
 * BlockChrome div is `relative`, so this panel overlays whatever sits in the
 * right margin.
 */
export function BlockCommentsPopover({ blockID, workspaceID, onClose }: BlockCommentsPopoverProps) {
  const refs = useRefs();
  const commentRefs = useMemo(
    () =>
      Object.values(refs).filter(
        (r) => r.to_id === blockID && r.rel === REL_COMMENTS_ON,
      ),
    [refs, blockID],
  );

  return (
    <div
      role="dialog"
      aria-label="Block comments"
      data-testid={`block-comments-popover-${blockID}`}
      className={cn(
        "pointer-events-auto absolute left-full top-0 z-30 ml-3 w-[300px]",
        "flex max-h-[420px] flex-col rounded-lg border border-border bg-popover shadow-md",
      )}
      onClick={(e) => e.stopPropagation()}
      onMouseDown={(e) => e.stopPropagation()}
    >
      <header className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
          {commentRefs.length === 0
            ? "Start a thread"
            : `${commentRefs.length} comment${commentRefs.length === 1 ? "" : "s"}`}
        </span>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close comments"
          className="inline-flex h-5 w-5 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          <X className="h-3 w-3" />
        </button>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-3 py-2">
        {commentRefs.length === 0 ? (
          <p className="py-1 text-[12px] text-muted-foreground">No comments yet.</p>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {commentRefs.map((r) => (
              <CommentItem key={r.id} commentRef={r} />
            ))}
          </ul>
        )}
      </div>

      <Composer blockID={blockID} workspaceID={workspaceID} />
    </div>
  );
}

function CommentItem({ commentRef }: { commentRef: BlockRef }) {
  const comment = useBlock(commentRef.from_id);
  if (!comment) return null;
  const text = (comment.data?.text as string | undefined) ?? "";
  const when = new Date(comment.created_at).toLocaleString();
  return (
    <li className="rounded border border-border bg-background p-2">
      <div className="flex items-baseline justify-between text-[10px] text-muted-foreground">
        <span>user #{comment.created_by}</span>
        <span>{when}</span>
      </div>
      <div className="mt-0.5 whitespace-pre-wrap break-words text-[13px] text-foreground">{text}</div>
    </li>
  );
}

function Composer({ blockID, workspaceID }: { blockID: string; workspaceID: string }) {
  const dispatch = useBlockstoreDispatch(workspaceID);
  const [draft, setDraft] = useState("");
  const canPost = draft.trim().length > 0;

  const post = async () => {
    if (!canPost) return;
    await dispatch.createCommentOn(blockID, draft.trim());
    setDraft("");
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        void post();
      }}
      className="border-t border-border px-2.5 py-2"
    >
      <div className="flex items-end gap-1.5">
        <textarea
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Write a comment…"
          rows={2}
          className="flex-1 resize-none rounded-md border border-border bg-background px-2 py-1 text-[12px] outline-none focus:ring-1 focus:ring-ring"
        />
        <button
          type="submit"
          disabled={!canPost}
          aria-label="Post comment"
          className={cn(
            "inline-flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-md",
            canPost
              ? "bg-primary text-primary-foreground hover:opacity-90"
              : "bg-muted text-muted-foreground",
          )}
        >
          <Send className="h-3 w-3" />
        </button>
      </div>
    </form>
  );
}

export default BlockCommentsPopover;
