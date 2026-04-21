"use client";

import { useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ChevronRight, FileText, Search, Plus, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useBlocks, useRefs, useNestChildrenIndex, useBlockstoreStore, useWorkspace } from "@/stores/blockstore";
import { useBlockTypeSpecs } from "@/lib/blockstore/useBlockTypeSpec";
import { useBlockstoreDispatch } from "@/components/blocks/editor/useBlockstoreDispatch";
import { BLOCK_TYPE_PAGE } from "@/lib/api/blockstoreTypes";
import { buildPageTree, countByType, colorForType, type PageNode } from "@/lib/blockstore/page-tree";
import {
  ContextMenu, ContextMenuTrigger, ContextMenuContent, ContextMenuItem,
} from "@/components/ui/context-menu";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useTranslations } from "next-intl";

/**
 * Master sidebar for the Blocks page. Self-contained: reads the active
 * workspace from the blockstore store and `?page` from the URL. Triggering
 * search dispatches the same ⌘K shortcut the page already listens for, so
 * neither side needs prop-drilling.
 *
 * Layout follows `design/pages/blocks-document.pastel`:
 *   Search pill → PAGES tree → INDICATOR TYPES list → Triggers footer.
 */
export function BlocksSidebar() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const pageParam = searchParams.get("page");
  const activeWorkspaceId = useBlockstoreStore((s) => s.activeWorkspaceId);
  const workspace = useWorkspace(activeWorkspaceId);
  const blocks = useBlocks();
  const refs = useRefs();
  const nestChildren = useNestChildrenIndex();

  const rootBlockID = workspace?.root_block_id ?? null;
  const workspaceID = workspace?.id ?? null;
  const selectedPageID = pageParam ?? rootBlockID;
  const dispatch = useBlockstoreDispatch(workspaceID ?? "");
  const t = useTranslations();
  const [pendingDelete, setPendingDelete] = useState<PageNode | null>(null);

  const typeSpecs = useBlockTypeSpecs(workspaceID ?? "");

  const tree = useMemo(
    () => (rootBlockID ? buildPageTree(blocks, refs, nestChildren, rootBlockID) : []),
    [blocks, refs, nestChildren, rootBlockID],
  );
  const typeCounts = useMemo(
    () => (workspaceID ? countByType(blocks, workspaceID) : {}),
    [blocks, workspaceID],
  );
  const triggerCount = typeCounts["trigger_def"] ?? 0;
  const typeEntries = Object.values(typeSpecs);

  const handleSelectPage = (id: string) => {
    const next = new URLSearchParams(Array.from(searchParams.entries()));
    if (id === rootBlockID) next.delete("page");
    else next.set("page", id);
    const qs = next.toString();
    router.replace(qs ? `?${qs}` : "?");
  };

  const handleAddPage = async () => {
    if (!rootBlockID) return;
    const newID = await dispatch.insertChild(
      rootBlockID,
      BLOCK_TYPE_PAGE,
      { title: "Untitled" },
      { text: "Untitled" },
    );
    if (newID) handleSelectPage(newID);
  };

  const handleOpenSearch = () => {
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true, bubbles: true }));
  };

  if (!workspace) {
    return (
      <aside className="flex h-full w-full flex-col items-center justify-center text-[12px] text-muted-foreground">
        Loading workspace…
      </aside>
    );
  }

  return (
    <aside className="flex h-full w-full flex-col">
      <div className="p-3">
        <button
          type="button"
          onClick={handleOpenSearch}
          className="flex h-[30px] w-full items-center gap-2 rounded-md border border-border bg-background px-2.5 text-[12px] text-muted-foreground transition-colors hover:bg-muted/50"
          data-testid="blocks-sidebar-search"
        >
          <Search className="h-3 w-3" />
          <span className="flex-1 text-left">Search blocks · semantic</span>
          <kbd className="rounded border bg-muted px-1 text-[10px]">⌘K</kbd>
        </button>
      </div>

      <SectionHeader title="PAGES" onAdd={handleAddPage} />
      <div className="flex flex-col gap-[1px] px-1">
        {tree.length === 0 ? (
          <p className="px-4 py-2 text-[12px] text-muted-foreground">No pages yet</p>
        ) : (
          tree.map((node) => (
            <PageTreeItem
              key={node.id}
              node={node}
              depth={0}
              selectedId={selectedPageID}
              onSelect={handleSelectPage}
              onDelete={setPendingDelete}
            />
          ))
        )}
      </div>

      <SectionHeader title="INDICATOR TYPES" className="mt-4" />
      <div className="flex flex-col gap-1 px-4 pb-2">
        {typeEntries.length === 0 ? (
          <p className="text-[12px] text-muted-foreground">No types yet</p>
        ) : (
          typeEntries.map((spec) => (
            <div key={spec.type} className="flex items-center gap-2 py-[3px]">
              <span
                className="h-2 w-2 rounded-sm"
                style={{ background: colorForType(spec.type) }}
              />
              <span className="font-mono text-[12px] font-medium text-foreground">{spec.type}</span>
              <span className="ml-auto font-mono text-[10px] text-muted-foreground">
                {typeCounts[spec.type] ?? 0}
              </span>
            </div>
          ))
        )}
      </div>

      <div className="flex-1" />

      <div className="flex items-center border-t border-border px-3 py-2.5 text-[12px] text-muted-foreground">
        <span>Triggers · {triggerCount} active</span>
      </div>

      <AlertDialog open={!!pendingDelete} onOpenChange={(open) => !open && setPendingDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("blocks.sidebar.deletePageTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("blocks.sidebar.deletePageDesc", { title: pendingDelete?.title ?? "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              data-testid="blocks-sidebar-delete-confirm"
              onClick={async () => {
                if (!pendingDelete) return;
                const victim = pendingDelete;
                setPendingDelete(null);
                await dispatch.removeBlock(victim.id);
                // Main BlocksPage effect watches `?page=` and falls back
                // to the workspace root when the target block disappears.
              }}
            >
              {t("common.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </aside>
  );
}

function SectionHeader({
  title,
  className,
  onAdd,
}: {
  title: string;
  className?: string;
  onAdd?: () => void;
}) {
  return (
    <div className={cn("flex items-center justify-between px-4 pb-1.5 pt-1", className)}>
      <span className="text-[10px] font-semibold uppercase tracking-[0.15em] text-muted-foreground">
        {title}
      </span>
      <button
        type="button"
        onClick={onAdd}
        disabled={!onAdd}
        aria-label={`Add ${title.toLowerCase()}`}
        className="inline-flex h-4 w-4 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-40 disabled:hover:bg-transparent"
      >
        <Plus className="h-3 w-3" />
      </button>
    </div>
  );
}

function PageTreeItem({
  node,
  depth,
  selectedId,
  onSelect,
  onDelete,
}: {
  node: PageNode;
  depth: number;
  selectedId: string | null;
  onSelect: (id: string) => void;
  onDelete: (node: PageNode) => void;
}) {
  const t = useTranslations();
  const isActive = selectedId === node.id;
  return (
    <>
      <ContextMenu>
        <ContextMenuTrigger asChild>
          <button
            type="button"
            onClick={() => onSelect(node.id)}
            className={cn(
              "flex w-full items-center gap-1.5 rounded-md py-1 pr-2 text-left text-[13px] transition-colors",
              isActive ? "bg-muted font-medium text-foreground" : "text-foreground hover:bg-muted/50",
            )}
            style={{ paddingLeft: `${8 + depth * 12}px` }}
            data-testid={`blocks-sidebar-page-${node.id}`}
          >
            <ChevronRight className="h-3 w-3 flex-shrink-0 text-muted-foreground" />
            <span aria-hidden="true" className="flex-shrink-0 text-muted-foreground">
              {node.icon ?? <FileText className="inline h-3 w-3" />}
            </span>
            <span className="truncate">{node.title}</span>
          </button>
        </ContextMenuTrigger>
        <ContextMenuContent>
          <ContextMenuItem
            className="text-destructive focus:text-destructive"
            onSelect={() => onDelete(node)}
            data-testid={`blocks-sidebar-page-${node.id}-delete`}
          >
            <Trash2 className="mr-2 h-3.5 w-3.5" />
            {t("blocks.sidebar.deletePage")}
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
      {node.children.map((child) => (
        <PageTreeItem
          key={child.id}
          node={child}
          depth={depth + 1}
          selectedId={selectedId}
          onSelect={onSelect}
          onDelete={onDelete}
        />
      ))}
    </>
  );
}

export default BlocksSidebar;
