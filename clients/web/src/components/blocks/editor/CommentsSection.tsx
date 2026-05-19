"use client";

import { useMemo } from "react";
import { MessageSquare } from "lucide-react";

import { BLOCK_TYPE_COMMENT, REL_COMMENTS_ON } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";
import { useRefs, useBlockstoreStore } from "@/stores/blockstore";

interface Props {
  blockID: string;
  workspaceID: string;
}

export function CommentsSection({ blockID }: Props) {
  const refs = useRefs();
  const activeCommentBlockID = useBlockstoreStore((s) => s.activeCommentBlockID);
  const setActive = useBlockstoreStore((s) => s.actions.setActiveCommentBlockID);

  const commentRefs = useMemo(
    () =>
      Object.values(refs).filter(
        (r) => r.to_id === blockID && r.rel === REL_COMMENTS_ON,
      ),
    [refs, blockID],
  );
  const count = commentRefs.length;

  if (count === 0) return null;

  const isActive = activeCommentBlockID === blockID;
  return (
    <div className="pl-1">
      <button
        type="button"
        onClick={() => setActive(isActive ? null : blockID)}
        className={cn(
          "inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px] transition-colors",
          isActive
            ? "bg-primary/10 text-primary"
            : "text-muted-foreground hover:bg-accent hover:text-foreground",
        )}
        data-testid={`block-comments-tail-${blockID}`}
      >
        <MessageSquare className="h-3 w-3" />
        <span>
          {count} comment{count === 1 ? "" : "s"}
        </span>
      </button>
    </div>
  );
}

export { BLOCK_TYPE_COMMENT };
