"use client";

import { useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";

import { useBlockstoreStore, useWorkspace } from "@/stores/blockstore";

import { buildPageQuery } from "./pageQuery";

export function useSelectPage(): (pageID: string) => void {
  const router = useRouter();
  const searchParams = useSearchParams();
  const activeWorkspaceId = useBlockstoreStore((s) => s.activeWorkspaceId);
  const rootBlockID = useWorkspace(activeWorkspaceId)?.root_block_id ?? null;

  return useCallback(
    (pageID: string) => {
      router.replace(buildPageQuery(searchParams, pageID, rootBlockID));
    },
    [router, searchParams, rootBlockID],
  );
}
