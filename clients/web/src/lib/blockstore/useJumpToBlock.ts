"use client";

import { useCallback } from "react";
import { useSearchParams } from "next/navigation";

import {
  readBlocks,
  readNestChildren,
  readRefs,
  useBlockstoreStore,
  useWorkspace,
} from "@/stores/blockstore";

import { findOwnerPage } from "./findOwnerPage";
import { useSelectPage } from "./useSelectPage";

export function useJumpToBlock(): (blockID: string) => void {
  const selectPage = useSelectPage();
  const searchParams = useSearchParams();
  const activeWorkspaceId = useBlockstoreStore((s) => s.activeWorkspaceId);
  const rootBlockID = useWorkspace(activeWorkspaceId)?.root_block_id ?? null;

  return useCallback(
    (blockID: string) => {
      const owner =
        findOwnerPage(blockID, readBlocks(), readRefs(), readNestChildren()) ?? rootBlockID;
      const currentPage = searchParams.get("page") ?? rootBlockID;
      if (owner && owner !== currentPage) selectPage(owner);
      scrollToBlockWhenReady(blockID);
    },
    [selectPage, searchParams, rootBlockID],
  );
}

// After a cross-page jump the target only enters the DOM once the new page
// commits, so poll a bounded number of frames rather than scrolling eagerly.
function scrollToBlockWhenReady(blockID: string, attempts = 60): void {
  const el = document.getElementById(`block-${blockID}`);
  if (el) {
    el.scrollIntoView({ behavior: "smooth", block: "center" });
    el.classList.add("ring-2", "ring-primary");
    setTimeout(() => el.classList.remove("ring-2", "ring-primary"), 1500);
    return;
  }
  if (attempts <= 0) return;
  requestAnimationFrame(() => scrollToBlockWhenReady(blockID, attempts - 1));
}
