"use client";

import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";

import { DocumentView } from "@/components/blocks/DocumentView";
import { SearchPanel } from "@/components/blocks/search/SearchPanel";
import { BlocksDocHeader } from "@/components/blocks/BlocksDocHeader";
import { CenteredSpinner } from "@/components/ui/spinner";
import { getErrorMessage } from "@/lib/utils";
import { blockstoreApi } from "@/lib/api/facade/blockstoreApi";
import { useSelectPage } from "@/lib/blockstore/useSelectPage";
import { useJumpToBlock } from "@/lib/blockstore/useJumpToBlock";
import { pageDisplayMeta } from "@/lib/blockstore/pageDisplayMeta";
import type { Workspace } from "@/lib/viewModels/blockstore";
import { useBlocks, useBlockstoreStore } from "@/stores/blockstore";
import { useCurrentOrg } from "@/stores/auth";
import "@/stores/blockstoreSubscribe";

export default function BlockstorePage() {
  const t = useTranslations();
  const searchParams = useSearchParams();
  const wsParam = searchParams.get("ws");
  const pageParam = searchParams.get("page");
  const currentOrg = useCurrentOrg();
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [searchOpen, setSearchOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const blocks = useBlocks();
  const selectPage = useSelectPage();
  const jumpToBlock = useJumpToBlock();

  const hydrate = async (ws: Workspace) => {
    setWorkspace(ws);
    useBlockstoreStore.getState().actions.setActiveWorkspaceId(ws.id);
    void useBlockstoreStore.getState().actions.loadTypeDefs(ws.id);
  };

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setWorkspace(null);
      try {
        let ws: Workspace | null = null;
        if (wsParam) {
          const list = await blockstoreApi.listWorkspaces();
          ws = list.workspaces.find((w) => w.id === wsParam) ?? null;
        }
        if (!ws) {
          ws = await useBlockstoreStore.getState().actions.ensureDefaultWorkspace();
        }
        if (cancelled) return;
        await hydrate(ws);
      } catch (e) {
        if (!cancelled) setError(getErrorMessage(e, t("blockstore.loadFailed")));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [t, wsParam, currentOrg]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setSearchOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const rootId = workspace?.root_block_id ?? null;
  const selectedPageID = pageParam ?? rootId;

  const rootMeta = useMemo(() => pageDisplayMeta(rootId ? blocks[rootId] : undefined), [rootId, blocks]);
  const currentMeta = useMemo(
    () => pageDisplayMeta(selectedPageID ? blocks[selectedPageID] : undefined),
    [selectedPageID, blocks],
  );

  if (error) return <div className="p-6 text-sm text-destructive">{error}</div>;
  if (!workspace || !rootId || !selectedPageID) return <CenteredSpinner />;

  return (
    <div className="flex h-full min-h-0 w-full">
      <main className="flex min-w-0 flex-1 flex-col">
        <BlocksDocHeader
          rootTitle={rootMeta.title}
          rootIcon={rootMeta.icon}
          currentTitle={currentMeta.title}
          currentIcon={currentMeta.icon}
          isRoot={selectedPageID === rootId}
          onAddBlock={() => setMenuOpen(true)}
          onNavigateRoot={() => selectPage(rootId)}
        />
        <div className="min-h-0 flex-1 overflow-y-auto">
          <DocumentView
            workspaceID={workspace.id}
            rootBlockID={selectedPageID}
            menuOpen={menuOpen}
            onMenuOpenChange={setMenuOpen}
          />
        </div>
      </main>
      <SearchPanel
        workspaceID={workspace.id}
        open={searchOpen}
        onClose={() => setSearchOpen(false)}
        onJumpToBlock={jumpToBlock}
      />
    </div>
  );
}
